package handle

import (
	"badminton-booking/badminton/data"
	"badminton-booking/badminton/misc"
	"badminton-booking/badminton/shard"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"os"
	"sort"
	"strconv"
)

func init() {
	r := RouterGet()

	r.GET("/admin", handleAdmin)
	r.POST("/admin/user/create", handleUserCreate)
	r.POST("/admin/user/feeUpdate", handleFeeUpdate)
	r.POST("/admin/setting/update", handleSettingUpdate)
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

	stats := data.BookingStats()

	users := data.UserFetchAll()

	userTotalAmt := 0
	userActiveAmt := 0
	userZombieAmt := 0

	for _, user := range users {
		userTotalAmt++

		if user.State == data.UserStateZombie {
			userZombieAmt++
		} else {
			userActiveAmt++
		}
	}

	sort.Slice(users, func(i, j int) bool {
		iUser := users[i]
		jUser := users[j]
		iStat := stats[iUser.UID]
		jStat := stats[jUser.UID]

		if iStat.Day30 != jStat.Day30 {
			return iStat.Day30 > jStat.Day30
		}

		if iStat.Day14 != jStat.Day14 {
			return iStat.Day14 > jStat.Day14
		}

		if iStat.Day7 != jStat.Day7 {
			return iStat.Day7 > jStat.Day7
		}

		return iUser.TrainingFee+iUser.BallFee+iUser.VenueFee > jUser.TrainingFee+jUser.BallFee+jUser.VenueFee
	})

	c.HTML(http.StatusOK, "admin.html", gin.H{
		"Title":  title,
		"Ticket": ticket,

		"UserTotalAmt":  userTotalAmt,
		"UserActiveAmt": userActiveAmt,
		"UserZombieAmt": userZombieAmt,

		"Users": users,
		"Stats": stats,

		"UserStateActive": data.UserStateActive,
		"UserStateAdmin":  data.UserStateAdmin,
		"UserStateZombie": data.UserStateZombie,
		"UserStateMap":    data.UserStateMap,

		"BookingStateOK":     data.BookingStateOK,
		"BookingStateNO":     data.BookingStateNO,
		"BookingStateAuto":   data.BookingStateAuto,
		"BookingStateManual": data.BookingStateManual,
		"BookingStateMap":    data.BookingStateMap,

		"Settings": shard.SettingExport(),
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

func handleFeeUpdate(c *gin.Context) {
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

	uid, _ := strconv.Atoi(c.PostForm("uid"))
	venueFee := misc.ToFloat32(c.PostForm("venueFee"))
	ballFee := misc.ToFloat32(c.PostForm("ballFee"))
	trainingFee := misc.ToFloat32(c.PostForm("trainingFee"))

	user := data.UserFetchById(uid)

	if user != nil {
		user.VenueFee += venueFee
		user.BallFee += ballFee
		user.TrainingFee += trainingFee

		if err := data.UserUpdateFee(user.Name, user.VenueFee, user.BallFee, user.TrainingFee); err != nil {
			c.String(http.StatusOK, fmt.Sprintf("update user failed: %s", err.Error()))
			return
		}
	}

	c.Redirect(http.StatusMovedPermanently, c.Request.Referer())
}

func handleSettingUpdate(c *gin.Context) {
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

	setting := c.PostForm("setting")

	if !json.Valid([]byte(setting)) {
		c.String(http.StatusOK, fmt.Sprintf("json format failed"))
		return
	}

	err := os.WriteFile("setting.json", []byte(setting), 0644)

	if err != nil {
		c.String(http.StatusOK, fmt.Sprintf("write setting failed: %s", err.Error()))
		return
	}

	shard.SettingReload()

	c.Redirect(http.StatusMovedPermanently, c.Request.Referer())
}
