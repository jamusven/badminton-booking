package data

import (
	"badminton-booking/badminton/misc"
	"fmt"
	"strings"
	"time"
)

func init() {
	rows, err := DBGet().Query("SELECT name FROM sqlite_master WHERE type='table' AND name='bookings'")

	if err != nil {
		panic(err)
	}

	defer rows.Close()

	if !rows.Next() {
		if _, err := DBGet().Exec("CREATE TABLE IF NOT EXISTS bookings (venue_id INTEGER, user_id INTEGER, worker TEXT, state INTEGER, time INTEGER, PRIMARY KEY (venue_id, user_id, worker))"); err != nil {
			panic(err)
		}

		if _, err := DBGet().Exec("create index bookings_venue_id_index on bookings (venue_id)"); err != nil {
			panic(err)
		}

		if _, err := DBGet().Exec("create index bookings_user_id_index on bookings (user_id)"); err != nil {
			panic(err)
		}

		if _, err := DBGet().Exec("create index bookings_worker_index on bookings (worker)"); err != nil {
			panic(err)
		}

	}
}

type Booking struct {
	VenueID int
	UserID  int
	Worker  string
	State   BookingState
	Time    int64
}

func (this *Booking) getUniqueKey() string {
	return fmt.Sprintf("%d_%d_%s", this.VenueID, this.UserID, this.Worker)
}

type BookingState int

const (
	BookingStateOK BookingState = iota + 1
	BookingStateAuto
	BookingStateManual
	BookingStateNO
	BookingStateExiting
)

var BookingStateMap = map[BookingState]string{
	BookingStateOK:      "确认",
	BookingStateNO:      "拒绝",
	BookingStateAuto:    "替补自动",
	BookingStateManual:  "替补手动",
	BookingStateExiting: "下车中",
}

type BookingSummary struct {
	VenueID         int
	Answers         []*Booking
	AnswerCounter   map[string]int
	AnswerResponses map[string][]string
	AnswerValues    map[string]BookingState
}

func (this *BookingSummary) Adjust(venue *Venue) bool {
	if venue.Limit == 0 {
		return false
	}

	details := []string{}
	details = append(details, fmt.Sprintf("[%s %s %s] 动态调整通知", venue.Name, venue.Day, venue.Desc))

	amount := venue.Limit

	marked := make(map[string]bool)

	answerCounter := make(map[BookingState]int)

	for _, booking := range this.Answers {
		if amount == 0 {
			break
		}

		key := booking.getUniqueKey()

		if _, ok := marked[key]; ok {
			continue
		}

		if booking.State == BookingStateOK {
			amount--
			marked[key] = true

			answerCounter[booking.State]++
		}
	}

	for _, booking := range this.Answers {
		if amount == 0 {
			break
		}

		key := booking.getUniqueKey()

		if _, ok := marked[key]; ok {
			continue
		}

		if booking.State == BookingStateAuto {
			amount--
			marked[key] = true

			booking.State = BookingStateOK
			answerCounter[booking.State]++

			if err := BookingCreate(booking.UserID, booking.VenueID, booking.State, booking.Time, booking.Worker); err != nil {
				panic(err)
			}

			msg := venue.Log(UserFetchById(booking.UserID).GetName(booking.Worker), fmt.Sprintf("From %s To %s", BookingStateMap[BookingStateAuto], BookingStateMap[booking.State]), time.Now())

			details = append(details, msg)
		}
	}

	for _, booking := range this.Answers {
		key := booking.getUniqueKey()

		if _, ok := marked[key]; ok {
			continue
		}

		if booking.State == BookingStateOK {
			booking.State = BookingStateAuto
			marked[key] = true

			answerCounter[booking.State]++

			if err := BookingCreate(booking.UserID, booking.VenueID, booking.State, booking.Time, booking.Worker); err != nil {
				panic(err)
			}

			msg := venue.Log(UserFetchById(booking.UserID).GetName(booking.Worker), fmt.Sprintf("From %s To %s", BookingStateMap[BookingStateOK], BookingStateMap[booking.State]), time.Now())

			details = append(details, msg)
		}
	}

	for _, booking := range this.Answers {
		key := booking.getUniqueKey()

		if _, ok := marked[key]; ok {
			continue
		}

		if answerCounter[BookingStateOK]+answerCounter[BookingStateAuto] >= venue.Amount && booking.State == BookingStateExiting {
			booking.State = BookingStateNO
			marked[key] = true

			if err := BookingCreate(booking.UserID, booking.VenueID, booking.State, time.Now().Unix(), booking.Worker); err != nil {
				panic(err)
			}

			msg := venue.Log(UserFetchById(booking.UserID).GetName(booking.Worker), fmt.Sprintf("From %s To %s", BookingStateMap[BookingStateExiting], BookingStateMap[booking.State]), time.Now())

			details = append(details, msg)
		}
	}

	if len(details) > 1 {
		go misc.LarkMarkdown(strings.Join(details, "\n"))

		return true
	}

	return false
}

func BookingCreate(userID int, venueID int, state BookingState, time int64, worker string) error {
	_, err := DBGet().Exec("REPLACE INTO bookings (venue_id, user_id, worker, state, time) VALUES (?, ?, ?, ?, ?)", venueID, userID, worker, state, time)

	return err
}

func BookingDelByVenueId(venueID int) error {
	_, err := DBGet().Exec("DELETE FROM bookings where venue_id = ?", venueID)

	return err
}

func BookingSummaryByVenueIds(ids []int) map[int]*BookingSummary {
	bookingSummaries := make(map[int]*BookingSummary)

	if len(ids) == 0 {
		return bookingSummaries
	}

	_ids := []string{}

	for _, v := range ids {
		_ids = append(_ids, misc.ToString(v))

		bookingSummary := &BookingSummary{
			VenueID:         v,
			Answers:         []*Booking{},
			AnswerCounter:   make(map[string]int),
			AnswerResponses: map[string][]string{},
			AnswerValues:    make(map[string]BookingState),
		}

		for _, answer := range BookingStateMap {
			bookingSummary.AnswerResponses[answer] = []string{}
			bookingSummary.AnswerCounter[answer] = 0
		}

		bookingSummaries[v] = bookingSummary
	}

	rows, err := DBGet().Query(fmt.Sprintf("SELECT venue_id, user_id, worker, state, time FROM bookings WHERE venue_id in (%s) order by time asc", strings.Join(_ids, ",")))

	if err != nil {
		panic(err)
	}

	defer rows.Close()

	for rows.Next() {
		booking := &Booking{}

		if err := rows.Scan(&booking.VenueID, &booking.UserID, &booking.Worker, &booking.State, &booking.Time); err != nil {
			panic(err)
		} else {
			bookingSummary, _ := bookingSummaries[booking.VenueID]

			user := UserFetchById(booking.UserID)
			answer := BookingStateMap[booking.State]

			bookingSummary.Answers = append(bookingSummary.Answers, booking)
			bookingSummary.AnswerCounter[answer]++
			bookingSummary.AnswerResponses[answer] = append(bookingSummary.AnswerResponses[answer], user.GetName(booking.Worker))
			bookingSummary.AnswerValues[user.GetName(booking.Worker)] = booking.State
		}
	}

	return bookingSummaries
}

func BookingSummaryByVenueId(id int) *BookingSummary {
	return BookingSummaryByVenueIds([]int{id})[id]
}

type BookingStat struct {
	UID           int
	ValueMap      map[string]interface{}
	Participation map[BookingState]string
	FirstTime     string
	LastTime      string
}

func BookingStats() (int, map[int]*BookingStat) {
	userStats := make(map[int]*BookingStat)

	users := UserFetchAll()

	for _, user := range users {
		stat := &BookingStat{
			ValueMap: make(map[string]interface{}),
			UID:      user.UID,
			LastTime: "",
		}

		stat.ValueMap["ok"] = 0
		stat.ValueMap["okPercent"] = "0.00%"

		userStats[user.UID] = stat
	}

	venueAmount := VenueCounter()

	if venueAmount == 0 {
		return venueAmount, userStats
	}

	now := time.Now().Unix()
	nowYmd := time.Now().Format(time.DateOnly)

	rows, err := DBGet().Query(fmt.Sprintf("select user_id, sum(case when state = %d then 1 else 0 end) as ok, sum(case when state = %d and time > %d then 1 else 0 end) as day7, sum(case when state = %d and time > %d then 1 else 0 end) as day14, sum(case when state = %d and time > %d then 1 else 0 end) as day30, max(time) as lastTime, min(time) as firstTime from bookings where user_id > 0 and worker = '' group by user_id", BookingStateOK, BookingStateOK, now-7*86400, BookingStateOK, now-14*86400, BookingStateOK, now-30*86400))

	if err != nil {
		panic(err)
	}

	defer rows.Close()

	for rows.Next() {
		var uid, ok, day7, day14, day30, lastTime, firstTime int

		if err := rows.Scan(&uid, &ok, &day7, &day14, &day30, &lastTime, &firstTime); err != nil {
			panic(err)
		} else {
			stat := userStats[uid]
			stat.FirstTime = time.Unix(int64(firstTime), 0).Format(time.DateTime)
			stat.LastTime = time.Unix(int64(lastTime), 0).Format(time.DateTime)

			stat.ValueMap["ok"] = ok
			stat.ValueMap["okPercent"] = fmt.Sprintf("%.2f%%", float32(ok)/float32(venueAmount)*100)
			stat.ValueMap["day7"] = day7
			stat.ValueMap["day14"] = day14
			stat.ValueMap["day30"] = day30
			stat.ValueMap["firstPast"] = misc.DiffDayByLabel(nowYmd, stat.FirstTime)
			stat.ValueMap["lastPast"] = misc.DiffDayByLabel(nowYmd, stat.LastTime)
		}
	}

	return venueAmount, userStats
}
