package data

import (
	"badminton-booking/badminton/misc"
	"errors"
	"fmt"
	"os"
	"sort"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func init() {
	if err := DBGet().AutoMigrate(&Venue{}); err != nil {
		panic(err)
	}
}

type Venue struct {
	gorm.Model
	CreatedAt    time.Time `gorm:"index"`
	Name         string
	Day          string
	State        VenueState `gorm:"index"`
	Amount       int
	Limit        int
	Desc         string
	Owner        uint `gorm:"index"`
	Fee          int64
	BallFee      int64
	TrainingFee  int64
	Notification bool `gorm:"default:true"`
	FillType     int  `gorm:"default:1"`
}

const LogDir = "logs"

func (this *Venue) GetLabel() string {
	return fmt.Sprintf("%s %s %s %d/%d", this.Name, this.Day, this.Desc, this.Amount, this.Limit)
}

func (this *Venue) Log(userName string, event string, time time.Time) string {
	msg := fmt.Sprintf("[%s %s] [%s] %s %s", this.Name, this.Day, userName, event, time.Format("2006-01-02 15:04:05"))

	if _, err := os.Stat(LogDir); os.IsNotExist(err) {
		os.Mkdir(LogDir, 0755)
	}

	fileName := fmt.Sprintf("%s/%d.log", LogDir, this.ID)

	if userName == "" && event == "" {
		if err := os.Remove(fileName); err != nil {
			panic(err)
		}

		return msg
	}

	f, err := os.OpenFile(fileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)

	if err != nil {
		panic(err)
		return msg
	}

	defer f.Close()

	_, err = f.WriteString(msg + "\n")

	if err != nil {
		panic(err)
	}

	return msg
}

func (this *Venue) NotificationMessage(message string) {
	if !this.Notification {
		return
	}
	go misc.LarkMarkdown(message)
}

type VenueState int

const (
	VenueStateRunning VenueState = iota + 1
	VenueStateDone
	VenueStateCancel
)

const (
	VenueFillManual = iota + 1
	VenueFillAuto
)

func VenueFetchById(id uint) *Venue {
	var venue Venue

	tx := DBGet().First(&venue, id)

	if tx.Error != nil && !errors.Is(tx.Error, gorm.ErrRecordNotFound) {
		panic(tx.Error)
	}

	if tx.RowsAffected == 0 {
		return nil
	}

	return &venue
}

func VenueFetchByState(state VenueState) ([]uint, []*Venue) {

	var venues []*Venue

	tx := DBGet().Where("state = ?", state).Order("day asc").Find(&venues)

	if tx.Error != nil {
		panic(tx.Error)
	}

	var ids []uint

	for _, venue := range venues {
		ids = append(ids, venue.ID)
	}

	return ids, venues
}

func VenueAutoFill(venue *Venue) {
	var venues []Venue
	var venueIds []uint

	tx := DBGet().Select("ID, day").Order("ID DESC").Limit(5).Find(&venues, "state != ?", VenueStateCancel)

	if tx.Error != nil && errors.Is(tx.Error, gorm.ErrRecordNotFound) {
		panic(tx.Error)
	}

	for _, venue := range venues {
		venueIds = append(venueIds, venue.ID)
	}

	var bookings []Booking

	tx = DBGet().Find(&bookings, "venue_id IN ? and worker = ''", venueIds)

	if tx.Error != nil {
		panic(tx.Error)
	}

	var userCounter = make(map[uint]int)

	for _, booking := range bookings {
		if booking.State != BookingStateOK {
			continue
		}

		if _, ok := userCounter[booking.UserID]; !ok {
			userCounter[booking.UserID] = 0
		}

		userCounter[booking.UserID] += 1
	}

	var users []*User
	userList := UserFetchAll()

	for _, user := range userList {
		if !user.IsActive() {
			continue
		}

		users = append(users, user)
	}

	sort.Slice(users, func(i, j int) bool {
		var iCount, jCount int

		if count, ok := userCounter[users[i].ID]; ok {
			iCount = count
		}

		if count, ok := userCounter[users[j].ID]; ok {
			jCount = count
		}

		return iCount < jCount
	})

	amount := 0

	for _, user := range users {
		state := BookingStateOK

		if amount >= venue.Limit {
			state = BookingStateAuto
		}

		tx := DBGet().Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "venue_id"}, {Name: "user_id"}, {Name: "worker"}},
			UpdateAll: true,
		}).Create(&Booking{
			VenueID: venue.ID,
			UserID:  user.ID,
			Worker:  "",
			State:   state,
			Time:    time.Now().Unix(),
		})

		selectionDesc := venue.Log(user.GetName(""), fmt.Sprintf("系统自动报名 %s", BookingStateMap[state]), time.Now())

		if venue.Notification {
			misc.LarkMarkdownChan(selectionDesc)
		}

		amount++

		if tx.Error != nil {
			misc.LarkMarkdown(fmt.Sprintf("create booking failed: %s", tx.Error.Error()))
			return
		}
	}
}
