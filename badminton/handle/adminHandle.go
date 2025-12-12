package handle

import (
	"badminton-booking/badminton/data"
	"badminton-booking/badminton/misc"
	"badminton-booking/badminton/shard"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

func init() {
	r := RouterGet()

	r.GET("/admin", handleAdmin)
	r.POST("/admin/user/create", handleUserCreate)
	r.POST("/admin/user/feeUpdate", handleFeeUpdate)
	r.POST("/admin/user/careerPeriodUpdate", handleCareerPeriodUpdate)
	r.POST("/admin/setting/update", handleSettingUpdate)
	r.POST("/admin/sql/query", handleSqlQuery)
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

	var balanceAmount int64
	var fareBalanceAmount int64
	var ballFeeAmount int64
	var trainingFeeAmount int64
	var venueFeeAmount int64

	for _, user := range users {
		userTotalAmt++

		if !user.IsActive() {
			userZombieAmt++
		} else {
			userActiveAmt++
		}

		balanceAmount += user.Balance
		fareBalanceAmount += user.FareBalance
		ballFeeAmount += user.BallFee
		trainingFeeAmount += user.TrainingFee
		venueFeeAmount += user.VenueFee
	}

	sort.Slice(users, func(i, j int) bool {
		iUser := users[i]
		jUser := users[j]
		iStat := stats[iUser.ID]
		jStat := stats[jUser.ID]

		iWeight := (iStat.Day7 << 8) | (iStat.Day14 << 4) | iStat.Day30
		jWeight := (jStat.Day7 << 8) | (jStat.Day14 << 4) | jStat.Day30

		if iWeight != jWeight {
			return iWeight > jWeight
		}

		return iStat.LastTime > jStat.LastTime
	})

	h := gin.H{
		"Title":  title,
		"Ticket": ticket,

		"UserTotalAmt":  userTotalAmt,
		"UserActiveAmt": userActiveAmt,
		"UserZombieAmt": userZombieAmt,

		"BalanceDetail": map[string]int64{
			"Balance":     balanceAmount,
			"FareBalance": fareBalanceAmount,
			"BallFee":     ballFeeAmount,
			"TrainingFee": trainingFeeAmount,
			"VenueFee":    venueFeeAmount,
		},

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
	}

	_json := c.Query("_json")

	if _json != "" {
		c.JSON(http.StatusOK, h)
		return
	}

	c.HTML(http.StatusOK, "admin.html", h)
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

	name := strings.TrimSpace(c.PostForm("name"))
	mobile := strings.TrimSpace(c.PostForm("mobile"))
	state := data.UserState(misc.ToINT(c.PostForm("state")))

	if name == "" {
		c.String(http.StatusOK, "name is empty")
		return
	}

	user := data.UserFetchByName(name)

	if user != nil {
		result := data.DBGet().Model(user).Updates(map[string]interface{}{
			"mobile": mobile,
			"state":  state,
		})

		if result.Error != nil {
			c.String(http.StatusOK, fmt.Sprintf("update user failed: %s", result.Error.Error()))
			return
		}

		user.Mobile = mobile
		user.State = state
		user.SaveCache()

		c.Redirect(http.StatusMovedPermanently, c.Request.Referer())

		return
	}

	user = &data.User{}
	user.Name = name
	user.Mobile = mobile
	user.State = state

	result := data.DBGet().Create(user)

	if result.Error != nil {
		c.String(http.StatusOK, fmt.Sprintf("create user failed: %s", result.Error.Error()))
		return
	}

	user.SaveCache()

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
	venueFee := int64(misc.ToFloat32(c.PostForm("venueFee")) * data.TransactionCents)
	ballFee := int64(misc.ToFloat32(c.PostForm("ballFee")) * data.TransactionCents)
	trainingFee := int64(misc.ToFloat32(c.PostForm("trainingFee")) * data.TransactionCents)
	balance := int64(misc.ToFloat32(c.PostForm("balance")) * data.TransactionCents)
	fareBalance := int64(misc.ToFloat32(c.PostForm("fareBalance")) * data.TransactionCents)

	user := data.UserFetchById(uint(uid))

	if user != nil {
		changes := map[string]interface{}{}

		if balance != 0 {
			changes["balance"] = user.Balance + balance
		}

		if ballFee != 0 {
			changes["ball_fee"] = user.BallFee + ballFee
		}

		if trainingFee != 0 {
			changes["training_fee"] = user.TrainingFee + trainingFee
		}

		if venueFee != 0 {
			changes["venue_fee"] = user.VenueFee + venueFee
		}

		if fareBalance != 0 {
			changes["fare_balance"] = user.FareBalance + fareBalance
		}

		tx := data.DBGet().Model(user).Updates(changes)

		if tx.Error != nil {
			c.String(http.StatusOK, fmt.Sprintf("update user failed: %s", tx.Error.Error()))
			return
		}

		if venueFee != 0 {
			if err := data.CreateTransaction(admin.ID, user.ID, 0, data.TransactionTypeVenue, venueFee, user.VenueFee, fmt.Sprintf("admin %s", admin.Name)); err != nil {
				c.String(http.StatusOK, fmt.Sprintf("create transaction failed: %s", tx.Error.Error()))
				return
			}
		}

		if ballFee != 0 {
			if err := data.CreateTransaction(admin.ID, user.ID, 0, data.TransactionTypeBall, ballFee, user.BallFee, fmt.Sprintf("admin %s", admin.Name)); err != nil {
				c.String(http.StatusOK, fmt.Sprintf("create transaction failed: %s", tx.Error.Error()))
				return
			}
		}

		if trainingFee != 0 {
			if err := data.CreateTransaction(admin.ID, user.ID, 0, data.TransactionTypeTraining, trainingFee, user.TrainingFee, fmt.Sprintf("admin %s", admin.Name)); err != nil {
				c.String(http.StatusOK, fmt.Sprintf("create transaction failed: %s", tx.Error.Error()))
				return
			}
		}

		if balance != 0 {
			if err := data.CreateTransaction(admin.ID, user.ID, 0, data.TransactionTypeBalance, balance, user.Balance, fmt.Sprintf("admin %s", admin.Name)); err != nil {
				c.String(http.StatusOK, fmt.Sprintf("create transaction failed: %s", tx.Error.Error()))
				return
			}

			go misc.LarkMarkdownChan(fmt.Sprintf("%s 的 %s 金额变动 %.2f 当前 %.2f by %s", user.Name, data.TransactionTypeMap[data.TransactionTypeBalance], misc.Cent2Yuan(balance, data.TransactionCents), misc.Cent2Yuan(user.Balance, data.TransactionCents), admin.Name))
		}

		if fareBalance != 0 {
			if err := data.CreateTransaction(admin.ID, user.ID, 0, data.TransactionTypeFare, fareBalance, user.FareBalance, fmt.Sprintf("admin %s", admin.Name)); err != nil {
				c.String(http.StatusOK, fmt.Sprintf("create transaction failed: %s", tx.Error.Error()))
				return
			}

			go misc.LarkMarkdownChan(fmt.Sprintf("%s 的 %s 金额变动 %.2f 当前 %.2f by %s", user.Name, data.TransactionTypeMap[data.TransactionTypeFare], misc.Cent2Yuan(fareBalance, data.TransactionCents), misc.Cent2Yuan(user.FareBalance, data.TransactionCents), admin.Name))
		}
	}

	c.Redirect(http.StatusMovedPermanently, c.Request.Referer())
}

func handleCareerPeriodUpdate(c *gin.Context) {
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
	careerPeriod := strings.TrimSpace(c.PostForm("careerPeriods"))

	user := data.UserFetchById(uint(uid))

	if user != nil {
		tx := data.DBGet().Model(user).Updates(map[string]interface{}{
			"career_periods": careerPeriod,
		})

		if tx.Error != nil {
			c.String(http.StatusOK, fmt.Sprintf("update user failed: %s", tx.Error.Error()))
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

func handleSqlQuery(c *gin.Context) {
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

	rawSql := c.PostForm("sql")
	tx := data.DBGet().Exec(rawSql)

	if tx.Error != nil {
		c.String(http.StatusOK, fmt.Sprintf("exec sql failed: %s", tx.Error.Error()))
		return
	}

	c.String(http.StatusOK, fmt.Sprintf("exec sql success: %d", tx.RowsAffected))
}
