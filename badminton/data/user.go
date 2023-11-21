package data

import (
	"fmt"
	_ "github.com/mattn/go-sqlite3"
)

func init() {
	rows, err := DBGet().Query("SELECT name FROM sqlite_master WHERE type='table' AND name='users'")

	if err != nil {
		panic(err)
	}

	defer rows.Close()

	if !rows.Next() {
		if _, err := DBGet().Exec("CREATE TABLE IF NOT EXISTS users (uid INTEGER PRIMARY KEY, name TEXT constraint users_name_unique unique, mobile TEXT, state INTEGER, venue_fee REAL default 0, ball_fee REAL default 0, training_fee REAL default 0)"); err != nil {
			panic(err)
		}

		if _, err := DBGet().Exec("create index users_state_index on users (state);"); err != nil {
			panic(err)
		}
	}

	UserFetchAll()
}

type User struct {
	UID         int
	Name        string
	Mobile      string
	State       UserState
	VenueFee    float32
	BallFee     float32
	TrainingFee float32
}

func (this *User) saveCache() {
	userCache[this.UID] = this
	userIdCacheName[this.Name] = this.UID
}

func (this *User) GetName(worker string) string {
	if worker == "" {
		return this.Name
	} else {
		return fmt.Sprintf("%s(%s)", worker, this.Name)
	}
}

type UserState int

const (
	UserStateActive UserState = iota + 1
	UserStateAdmin
	UserStateZombie
)

var UserStateMap = map[UserState]string{
	UserStateActive: "活跃",
	UserStateAdmin:  "管理员",
	UserStateZombie: "僵尸",
}

var userCache = make(map[int]*User)
var userIdCacheName = make(map[string]int)

var tickets = make(map[string]string)
var admin = &User{UID: 0, Name: GodTicket, State: UserStateAdmin}

func UserCreate(name string, mobile string, state UserState) error {
	result, err := DBGet().Exec("INSERT INTO users (name, mobile, state) VALUES (?, ?, ?)", name, mobile, state)

	if err == nil {
		user := &User{}
		user.Name = name
		user.Mobile = mobile
		user.State = state

		if uid, err := result.LastInsertId(); err == nil {
			user.UID = int(uid)
			user.saveCache()
		} else {
			return err
		}
	}

	return err
}

func UserUpdate(name string, mobile string, state UserState) error {
	_, err := DBGet().Exec("UPDATE users SET mobile = ?, state = ? WHERE name = ?", mobile, state, name)

	return err
}

func UserUpdateFee(name string, venueFee, ballFee, trainingFee float32) error {
	_, err := DBGet().Exec("UPDATE users SET venue_fee = ?, ball_fee = ?, training_fee = ? WHERE name = ?", venueFee, ballFee, trainingFee, name)

	return err
}

func UserFetchById(uid int) *User {
	if uid == 0 {
		return admin
	}

	if _, ok := userCache[uid]; ok {
		return userCache[uid]
	}

	rows, err := DBGet().Query("SELECT uid, name, mobile FROM users WHERE uid = ?", uid)

	if err != nil {
		panic(err)
	}

	defer rows.Close()

	if !rows.Next() {
		return nil
	}

	user := &User{}

	if err := rows.Scan(&user.UID, &user.Name, &user.Mobile); err != nil {
		panic(err)
	}

	user.saveCache()

	return user
}

func UserFetchByName(name string) *User {
	if uid, ok := userIdCacheName[name]; ok {
		if _, ok := userCache[uid]; ok {
			return userCache[uid]
		}
	}

	rows, err := DBGet().Query("SELECT uid, name, mobile FROM users WHERE name = ?", name)

	if err != nil {
		panic(err)
	}

	defer rows.Close()

	if !rows.Next() {
		return nil
	}

	user := &User{}

	if err := rows.Scan(&user.UID, &user.Name, &user.Mobile); err != nil {
		panic(err)
	}

	user.saveCache()

	return user
}

func UserFetchByTicket(ticket string) *User {
	if ticket == GodTicket {
		return admin
	}

	name, _ := tickets[ticket]

	if name == "" {
		return nil
	}

	return UserFetchByName(name)
}

func UserFetchAll() []*User {
	var users []*User

	if len(userCache) > 0 {
		for _, user := range userCache {
			users = append(users, user)
		}

		return users
	}

	rows, err := DBGet().Query("SELECT uid, name, mobile, state, venue_fee, ball_fee, training_fee FROM users")

	if err != nil {
		panic(err)
	}

	defer rows.Close()

	for rows.Next() {
		user := &User{}
		if err := rows.Scan(&user.UID, &user.Name, &user.Mobile, &user.State, &user.VenueFee, &user.BallFee, &user.TrainingFee); err != nil {
			panic(err)
		}

		users = append(users, user)

		user.saveCache()
	}

	return users
}

func UserNameMapGet() map[string]int {
	return userIdCacheName
}

func TicketSet(ticket string, value string) {
	for k, v := range tickets {
		if v == value {
			delete(tickets, k)
		}
	}

	tickets[ticket] = value
}
