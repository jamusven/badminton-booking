package handle

import (
	"badminton-booking/badminton/data"
	"github.com/gin-gonic/gin"
	"net/http"
)

func init() {
	r := RouterGet()

	r.GET("/", handleIndex)
	r.GET("/list", handleList)
	r.GET("/login", handleLogin)
}

func handleIndex(c *gin.Context) {
	c.Status(http.StatusServiceUnavailable)
}

func handleList(c *gin.Context) {
	ticket := c.Query("ticket")

	if ticket == "" {
		c.Status(http.StatusServiceUnavailable)
		return
	}

	user := data.UserFetchByTicket(ticket)

	if user == nil {
		c.Status(http.StatusServiceUnavailable)
		return
	}

	users := data.UserFetchAll()

	venueIds, venues := data.VenueFetchByState(data.VenueStateRunning)

	bookingSummaries := data.BookingSummaryByVenueIds(venueIds)

	c.HTML(http.StatusOK, "list.html", gin.H{
		"Title":  title,
		"Ticket": ticket,
		"Me":     user,

		"Venues":      venues,
		"Users":       users,
		"UserNameMap": data.UserNameMapGet(),

		"BookingStateOK":     data.BookingStateOK,
		"BookingStateNO":     data.BookingStateNO,
		"BookingStateAuto":   data.BookingStateAuto,
		"BookingStateManual": data.BookingStateManual,
		"BookingStateMap":    data.BookingStateMap,

		"BookingSummaries": bookingSummaries,

		"UserStateActive": data.UserStateActive,
		"UserStateAdmin":  data.UserStateAdmin,
		"UserStateZombie": data.UserStateZombie,
		"UserStateMap":    data.UserStateMap,
	})
}

func handleLogin(c *gin.Context) {
	ticket := c.Query("ticket")

	if ticket != title {
		c.Status(http.StatusServiceUnavailable)
		return
	}

	c.HTML(http.StatusOK, "login.html", gin.H{
		"Title": title,
	})
}
