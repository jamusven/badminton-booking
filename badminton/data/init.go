package data

import (
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"sync"
)

const dataBaseFile string = "database.db"
const GodTicket = "sven666"

var Locker = &sync.Mutex{}

var db *gorm.DB

func DBGet() *gorm.DB {
	if db != nil {
		return db
	}

	var err error

	db, err = gorm.Open(sqlite.Open(dataBaseFile), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}
	return db
}
