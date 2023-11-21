package data

import (
	"badminton-booking/badminton/misc"
	"fmt"
	"os"
	"time"
)

func init() {
	rows, err := DBGet().Query("SELECT name FROM sqlite_master WHERE type='table' AND name='venues'")

	if err != nil {
		panic(err)
	}

	defer rows.Close()

	if !rows.Next() {
		if _, err := DBGet().Exec("CREATE TABLE IF NOT EXISTS venues (id INTEGER PRIMARY KEY, name TEXT, day TEXT, state INTEGER, amount INTEGER, `limit` INTEGER, desc TEXT, owner INTEGER, fee REAL DEFAULT 0, ball_fee REAL DEFAULT 0)"); err != nil {
			panic(err)
		}

		if _, err := DBGet().Exec("create index venues_state_index on venues (state);"); err != nil {
			panic(err)
		}

		if _, err := DBGet().Exec("create index venues_owner_index on venues (owner);"); err != nil {
			panic(err)
		}
	}
}

type Venue struct {
	ID      int
	Name    string
	Day     string
	State   VenueState
	Amount  int
	Limit   int
	Desc    string
	Owner   int
	Fee     float32
	BallFee float32
}

const LogDir = "logs"

func (this *Venue) Log(userName string, event string, time time.Time) string {
	msg := fmt.Sprintf("[%s %s] [%s] %s %s", this.Name, this.Day, userName, event, time.Format("2006-01-02 15:04:05"))

	if _, err := os.Stat(LogDir); os.IsNotExist(err) {
		os.Mkdir(LogDir, 0755)
	}

	fileName := fmt.Sprintf("%s/%s.log", LogDir, misc.Sha1(misc.ToString(this.ID)))

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

func VenueCreate(owner int, name string, day string, state VenueState, amount, limit int, desc string) (int, error) {
	if result, err := DBGet().Exec("INSERT INTO venues (name, day, state, amount, `limit`, desc, owner) VALUES (?, ?, ?, ?, ?, ?, ?)", name, day, state, amount, limit, desc, owner); err != nil {
		return 0, err
	} else {
		id, _ := result.LastInsertId()

		return int(id), nil
	}
}

func VenueFetchById(id int) *Venue {
	rows, err := DBGet().Query("SELECT id, name, day, state, amount, `limit`, desc, owner, fee, ball_fee FROM venues WHERE id = ?", id)

	if err != nil {
		panic(err)
	}

	defer rows.Close()

	if !rows.Next() {
		return nil
	}

	venue := &Venue{}

	if err := rows.Scan(&venue.ID, &venue.Name, &venue.Day, &venue.State, &venue.Amount, &venue.Limit, &venue.Desc, &venue.Owner, &venue.Fee, &venue.BallFee); err != nil {
		panic(err)
	}

	return venue
}

func VenueFetchByState(state VenueState) ([]int, []*Venue) {
	rows, err := DBGet().Query("SELECT id, name, day, state, amount, `limit`, desc, owner, fee, ball_fee FROM venues WHERE state = ? order by day asc", state)

	if err != nil {
		panic(err)
	}

	defer rows.Close()

	var ids []int

	var venues []*Venue
	for rows.Next() {
		var venue Venue
		if err := rows.Scan(&venue.ID, &venue.Name, &venue.Day, &venue.State, &venue.Amount, &venue.Limit, &venue.Desc, &venue.Owner, &venue.Fee, &venue.BallFee); err != nil {
			panic(err)
		}

		venues = append(venues, &venue)
		ids = append(ids, venue.ID)
	}

	return ids, venues
}

func VenueStateUpdate(id int, state VenueState, fee, ballFee float32) error {
	if _, err := DBGet().Exec("update venues set state = ?, fee = ?, ball_fee = ? where id = ?", state, fee, ballFee, id); err != nil {
		return err
	}

	return nil
}

func VenueUpdateDetail(id, amount, limit int, name, day, desc string) error {
	if _, err := DBGet().Exec("update venues set amount = ? , `limit` = ?, name = ?, day = ?, desc = ? where id = ?", amount, limit, name, day, desc, id); err != nil {
		return err
	}

	return nil
}

func VenueCounter() int {
	rows, err := DBGet().Query("SELECT count(1) FROM venues WHERE state IN (?, ?)", VenueStateRunning, VenueStateDone)

	if err != nil {
		panic(err)
	}

	defer rows.Close()

	if !rows.Next() {
		return 0
	}

	amount := 0

	if err := rows.Scan(&amount); err != nil {
		panic(err)
	}

	return amount
}
