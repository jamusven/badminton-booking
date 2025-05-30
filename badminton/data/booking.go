package data

import (
	"badminton-booking/badminton/misc"
	"errors"
	"fmt"
	"gorm.io/gorm"
	"strings"
	"time"
)

func init() {
	err := DBGet().AutoMigrate(&Booking{})

	if err != nil {
		panic(err)
	}
}

type Booking struct {
	gorm.Model
	VenueID uint   `gorm:"uniqueIndex:venue_user_worker"`
	UserID  uint   `gorm:"uniqueIndex:venue_user_worker"`
	Worker  string `gorm:"uniqueIndex:venue_user_worker"`
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
	VenueID         uint
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

		if booking.State == BookingStateNO || booking.State == BookingStateManual {
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

			tx := DBGet().Updates(booking)

			if tx.Error != nil {
				panic(tx.Error)
			}

			userName := UserFetchById(booking.UserID).GetName(booking.Worker)

			msg := venue.Log(userName, fmt.Sprintf("From %s To %s", BookingStateMap[BookingStateAuto], BookingStateMap[booking.State]), time.Now())

			go misc.WechatSingle(userName, msg)

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

			tx := DBGet().Updates(booking)

			if tx.Error != nil {
				panic(tx.Error)
			}

			userName := UserFetchById(booking.UserID).GetName(booking.Worker)

			msg := venue.Log(userName, fmt.Sprintf("From %s To %s", BookingStateMap[BookingStateOK], BookingStateMap[booking.State]), time.Now())

			go misc.WechatSingle(userName, msg)

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
			booking.Time = time.Now().Unix()
			marked[key] = true

			tx := DBGet().Updates(booking)

			if tx.Error != nil {
				panic(tx.Error)
			}
		}

		if booking.State == BookingStateAuto {
			continue
		}

		userName := UserFetchById(booking.UserID).GetName(booking.Worker)

		msg := venue.Log(userName, fmt.Sprintf("From %s To %s", BookingStateMap[BookingStateExiting], BookingStateMap[booking.State]), time.Now())

		go misc.WechatSingle(userName, msg)

		details = append(details, msg)
	}

	if len(details) > 1 {
		venue.NotificationMessage(strings.Join(details, "\n"))

		return true
	}

	return false
}

func BookingSummaryByVenueIds(ids []uint) map[uint]*BookingSummary {
	bookingSummaries := make(map[uint]*BookingSummary)

	if len(ids) == 0 {
		return bookingSummaries
	}

	_ids := []string{}

	for _, v := range ids {
		_ids = append(_ids, misc.ToString(int(v)))

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

	var bookings []Booking

	tx := DBGet().Order("time asc").Find(&bookings, "venue_id in (?)", ids)

	if tx.Error != nil {
		panic(tx.Error)
	}

	for i, booking := range bookings {
		bookingSummary, _ := bookingSummaries[booking.VenueID]

		user := UserFetchById(booking.UserID)
		answer := BookingStateMap[booking.State]

		bookingSummary.Answers = append(bookingSummary.Answers, &bookings[i])
		bookingSummary.AnswerCounter[answer]++
		bookingSummary.AnswerResponses[answer] = append(bookingSummary.AnswerResponses[answer], user.GetName(booking.Worker))
		bookingSummary.AnswerValues[user.GetName(booking.Worker)] = booking.State
	}

	return bookingSummaries
}

func BookingSummaryByVenueId(id uint) *BookingSummary {
	return BookingSummaryByVenueIds([]uint{id})[id]
}

type BookingStat struct {
	UID            uint
	ValueMap       map[string]interface{}
	FirstTime      int64
	LastTime       int64
	VenueAmount    int64
	Day7           int
	Day14          int
	Day30          int
	Day60          int
	ConfirmAmount  int
	ResponseAmount int
}

func BookingStats() map[uint]*BookingStat {
	now := time.Now().Unix()
	nowYmd := time.Unix(now, 0).Format(time.DateOnly)

	userStats := make(map[uint]*BookingStat)

	users := UserFetchAll()

	for _, user := range users {
		stat := &BookingStat{
			ValueMap:  make(map[string]interface{}),
			UID:       user.ID,
			FirstTime: user.CreatedAt.Unix(),
		}

		userStats[user.ID] = stat
	}

	venueDayMap := make(map[uint]int64)

	var venues []Venue

	tx := DBGet().Select("ID, day").Find(&venues, "state != ?", VenueStateCancel)

	if tx.Error != nil && errors.Is(tx.Error, gorm.ErrRecordNotFound) {
		panic(tx.Error)
	}

	for _, venue := range venues {
		dayTime, _ := time.Parse(time.DateOnly, venue.Day)
		venueDayMap[venue.ID] = dayTime.Unix()
	}

	var bookings []Booking

	tx = DBGet().Find(&bookings, "worker = ''")

	if tx.Error != nil {
		panic(tx.Error)
	}

	for _, booking := range bookings {
		var ok bool
		var venueTime int64

		if venueTime, ok = venueDayMap[booking.VenueID]; !ok {
			continue
		}

		var stat *BookingStat

		if stat, ok = userStats[booking.UserID]; !ok {
			continue
		}

		adjustBookingTime := booking.Time

		if adjustBookingTime < venueTime {
			adjustBookingTime = venueTime
		}

		stat.ResponseAmount++

		if booking.State == BookingStateOK {
			stat.ConfirmAmount++

			if now >= venueTime {
				if now-adjustBookingTime <= 86400*7 {
					stat.Day7++
					stat.Day14++
					stat.Day30++
					stat.Day60++
				} else if now-adjustBookingTime <= 86400*14 {
					stat.Day14++
					stat.Day30++
					stat.Day60++
				} else if now-adjustBookingTime <= 86400*30 {
					stat.Day30++
					stat.Day60++
				} else if now-adjustBookingTime <= 86400*60 {
					stat.Day60++
				}
			}
		}

		if stat.FirstTime == 0 {
			stat.FirstTime = booking.Time
		}

		if stat.LastTime == 0 || stat.LastTime < booking.Time {
			stat.LastTime = booking.Time
		}
	}

	for _, user := range users {
		stat, _ := userStats[user.ID]

		var careerPeriods []map[string]int64

		if user.CareerPeriods != "" {
			_careerPeriods := strings.Split(user.CareerPeriods, ";")

			for _, period := range _careerPeriods {
				periods := strings.Split(period, "_")
				if len(periods) != 2 {
					continue
				}

				startTime, err := time.Parse(time.DateOnly, periods[0])
				if err != nil {
					continue
				}

				endTime, err := time.Parse(time.DateOnly, periods[1])
				if err != nil {
					continue
				}

				careerPeriods = append(careerPeriods, map[string]int64{
					"start": startTime.Unix(),
					"end":   endTime.Unix(),
				})
			}
		}

		for _, venueDayTime := range venueDayMap {
			if len(careerPeriods) == 0 {
				if venueDayTime >= user.CreatedAt.Unix() && venueDayTime <= stat.LastTime+14*86400 {
					stat.VenueAmount++
				}
			} else {
				for _, period := range careerPeriods {
					if venueDayTime >= period["start"] && venueDayTime <= period["end"] {
						stat.VenueAmount++
						break
					}
				}
			}
		}

		stat.ValueMap["confirmPercent"] = fmt.Sprintf("%.2f%%", float32(stat.ConfirmAmount)/float32(stat.VenueAmount)*100)
		stat.ValueMap["responsePercent"] = fmt.Sprintf("%.2f%%", float32(stat.ResponseAmount)/float32(stat.VenueAmount)*100)

		stat.ValueMap["firstTime"] = time.Unix(stat.FirstTime, 0).Format(time.DateTime)
		stat.ValueMap["lastTime"] = time.Unix(stat.LastTime, 0).Format(time.DateTime)

		stat.ValueMap["firstPast"] = misc.DiffDayByLabel(nowYmd, time.Unix(stat.FirstTime, 0).Format(time.DateTime))
		stat.ValueMap["lastPast"] = misc.DiffDayByLabel(nowYmd, time.Unix(stat.LastTime, 0).Format(time.DateTime))
	}

	return userStats
}
