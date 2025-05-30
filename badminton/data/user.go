package data

import (
	"fmt"
	"gorm.io/gorm"
)

func init() {
	if err := DBGet().AutoMigrate(&User{}); err != nil {
		panic(err)
	}

	UserFetchAll()
}

type User struct {
	gorm.Model
	Name          string `gorm:"uniqueIndex"`
	Mobile        string
	State         UserState
	VenueFee      float32
	BallFee       float32
	TrainingFee   float32
	Balance       float32
	FareBalance   float32
	FareFee       float32
	CareerPeriods string
}

func (this *User) SaveCache() {
	userCache[this.ID] = this
	userIdCacheName[this.Name] = this.ID
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

var userCache = make(map[uint]*User)
var userIdCacheName = make(map[string]uint)

var tickets = make(map[string]string)
var admin = &User{Name: GodTicket, State: UserStateAdmin}

func UserFetchById(uid uint) *User {
	if uid == 0 {
		return admin
	}

	if _, ok := userCache[uid]; ok {
		return userCache[uid]
	}

	return nil
}

func UserFetchByName(name string) *User {
	if uid, ok := userIdCacheName[name]; ok {
		if _, ok := userCache[uid]; ok {
			return userCache[uid]
		}
	}

	return nil
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

	result := DBGet().Find(&users)

	if result.Error != nil {
		panic(result.Error)
	}

	for _, user := range users {
		user.SaveCache()
	}

	return users
}

func UserNameMapGet() map[string]uint {
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
