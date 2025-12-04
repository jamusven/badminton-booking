package handle

import (
	"badminton-booking/badminton/data"
	"badminton-booking/badminton/misc"
	"badminton-booking/badminton/shard"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func init() {
	r := RouterGet()

	r.GET("/", handleIndex)
	r.GET("/list", handleList)
	r.GET("/login", handleLogin)
	r.POST("/notification", handleNotification)
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

	json := c.Query("_json")

	user := data.UserFetchByTicket(ticket)

	if user == nil {
		c.Status(http.StatusServiceUnavailable)
		return
	}

	users := data.UserFetchAll()

	venueIds, venues := data.VenueFetchByState(data.VenueStateRunning)

	bookingSummaries := data.BookingSummaryByVenueIds(venueIds)

	h := gin.H{
		"Title":           title,
		"Ticket":          ticket,
		"Alias":           shard.SettingInstance.Alias,
		"VenueBookingMap": shard.SettingInstance.VenueBookingMap,

		"Me": user,

		"NowYMD": time.Now().Format(time.DateOnly),

		"Venues":      venues,
		"Users":       users,
		"UserNameMap": data.UserNameMapGet(),

		"BookingStateOK":      data.BookingStateOK,
		"BookingStateNO":      data.BookingStateNO,
		"BookingStateAuto":    data.BookingStateAuto,
		"BookingStateManual":  data.BookingStateManual,
		"BookingStateExiting": data.BookingStateExiting,
		"BookingStateMap":     data.BookingStateMap,

		"BookingSummaries": bookingSummaries,

		"UserStateActive": data.UserStateActive,
		"UserStateAdmin":  data.UserStateAdmin,
		"UserStateZombie": data.UserStateZombie,
		"UserStateMap":    data.UserStateMap,

		"TransactionTypeVenue":    data.TransactionTypeVenue,
		"TransactionTypeBall":     data.TransactionTypeBall,
		"TransactionTypeTraining": data.TransactionTypeTraining,
		"TransactionTypeBalance":  data.TransactionTypeBalance,
		"TransactionTypeFare":     data.TransactionTypeFare,
		"TransactionTypeMap":      data.TransactionTypeMap,
	}

	if json != "" {
		c.JSON(http.StatusOK, h)
		return
	}

	c.HTML(http.StatusOK, "list1.html", h)
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

func handleNotification(c *gin.Context) {
	ticket := c.PostForm("ticket")
	text := c.PostForm("text")

	user := data.UserFetchByTicket(ticket)

	if user == nil {
		c.Status(http.StatusServiceUnavailable)
		return
	}

	go misc.LarkMarkdown(text)
	c.Redirect(http.StatusMovedPermanently, c.Request.Referer())
}
