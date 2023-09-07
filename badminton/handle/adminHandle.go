package handle

import (
	"badminton-booking/badminton/data"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
)

func init() {
	r := RouterGet()

	r.GET("/admin", handleAdmin)
	r.POST("/admin/user/create", handleUserCreate)
}

func handleAdmin(c *gin.Context) {
	ticket := c.Query("ticket")

	if ticket == "" {
		c.Status(http.StatusServiceUnavailable)
		return
	}

	admin := data.UserFetchByTicket(ticket)

	if admin == nil || admin.State != data.UserStateAdmin {
		c.Status(http.StatusServiceUnavailable)
		return
	}

	venueAmount, stats := data.BookingStats()

	c.HTML(http.StatusOK, "admin.html", gin.H{
		"Title":  title,
		"Ticket": ticket,

		"VenueAmount": venueAmount,
		"Users":       data.UserFetchAll(),
		"Stats":       stats,

		"UserStateActive": data.UserStateActive,
		"UserStateAdmin":  data.UserStateAdmin,
		"UserStateZombie": data.UserStateZombie,
		"UserStateMap":    data.UserStateMap,

		"BookingStateOK":     data.BookingStateOK,
		"BookingStateNO":     data.BookingStateNO,
		"BookingStateAuto":   data.BookingStateAuto,
		"BookingStateManual": data.BookingStateManual,
		"BookingStateMap":    data.BookingStateMap,
	})
}

func handleUserCreate(c *gin.Context) {
	data.Locker.Lock()
	defer data.Locker.Unlock()

	ticket := c.PostForm("ticket")

	if ticket == "" {
		c.Status(http.StatusServiceUnavailable)
		return
	}

	admin := data.UserFetchByTicket(ticket)

	if admin == nil || admin.State != data.UserStateAdmin {
		c.Status(http.StatusServiceUnavailable)
		return
	}

	name := c.PostForm("name")
	mobile := c.PostForm("mobile")
	state1, _ := strconv.Atoi(c.PostForm("state"))
	state := data.UserState(state1)

	if name == "" {
		c.String(http.StatusOK, "name is empty")
		return
	}

	user := data.UserFetchByName(name)

	if user != nil {
		if err := data.UserUpdate(name, mobile, state); err != nil {
			c.String(http.StatusOK, fmt.Sprintf("update user failed: %s", err.Error()))
			return
		} else {
			user.Mobile = mobile
			user.State = state
		}

		c.Redirect(http.StatusMovedPermanently, c.Request.Referer())

		return
	}

	if err := data.UserCreate(name, mobile, state); err != nil {
		c.String(http.StatusOK, fmt.Sprintf("create user failed: %s", err.Error()))
		return
	}

	c.Redirect(http.StatusMovedPermanently, c.Request.Referer())
}
