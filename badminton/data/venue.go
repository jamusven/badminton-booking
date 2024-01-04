package data

import (
	"badminton-booking/badminton/misc"
	"errors"
	"fmt"
	"gorm.io/gorm"
	"os"
	"time"
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
	Fee          float32
	BallFee      float32
	TrainingFee  float32
	Notification bool `gorm:"default:true"`
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

	fileName := fmt.Sprintf("%s/%s.log", LogDir, misc.Sha1(misc.ToString(int(this.ID))))

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

type VenueSummary struct {
	Venue    *Venue
	Bookings []*Booking
}

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

	tx := DBGet().Where("state = ?", state).Find(&venues)

	if tx.Error != nil {
		panic(tx.Error)
	}

	var ids []uint

	for _, venue := range venues {
		ids = append(ids, venue.ID)
	}

	return ids, venues
}
