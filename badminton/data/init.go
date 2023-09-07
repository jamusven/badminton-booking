package data

import (
	"database/sql"
	"sync"
)

const dataBaseFile string = "database.db"
const GodTicket = "sven666"

var Locker = &sync.Mutex{}

var db *sql.DB

func DBGet() *sql.DB {
	if db != nil {
		return db
	}

	var err error

	if db, err = sql.Open("sqlite3", dataBaseFile); err != nil {
		panic(err)
	}

	return db
}
