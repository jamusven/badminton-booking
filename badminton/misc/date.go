package misc

import (
	"fmt"
	"time"
)

func GetWeekDay(day string) string {
	t, err := time.Parse(time.DateOnly, day)

	if err != nil {
		fmt.Println(err)
	}

	return t.Weekday().String()
}

func Now() int64 {
	return time.Now().Unix()
}

func DiffDayByLabel(t1, t2 string) int {
	var ta, tb time.Time

	if len(t1) == len(time.DateOnly) {
		ta, _ = time.Parse(time.DateOnly, t1)
	} else {
		ta, _ = time.Parse(time.DateTime, t1)
	}

	if len(t2) == len(time.DateOnly) {
		tb, _ = time.Parse(time.DateOnly, t2)
	} else {
		tb, _ = time.Parse(time.DateTime, t2)
	}

	difference := ta.Sub(tb)

	return int(difference.Hours() / 24)
}
