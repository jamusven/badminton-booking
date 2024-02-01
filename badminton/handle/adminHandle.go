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
	"strings"
)

func init() {
	r := RouterGet()

	r.GET("/admin", handleAdmin)
	r.POST("/admin/user/create", handleUserCreate)
	r.POST("/admin/user/feeUpdate", handleFeeUpdate)
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

	var balanceAmount float32
	var fareBalanceAmount float32
	var ballFeeAmount float32
	var trainingFeeAmount float32
	var venueFeeAmount float32

	for _, user := range users {
		userTotalAmt++

		if user.State == data.UserStateZombie {
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

		"BalanceDetail": map[string]float32{
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
	venueFee := misc.ToFloat32(c.PostForm("venueFee"))
	ballFee := misc.ToFloat32(c.PostForm("ballFee"))
	trainingFee := misc.ToFloat32(c.PostForm("trainingFee"))
	balance := misc.ToFloat32(c.PostForm("balance"))
	fareBalance := misc.ToFloat32(c.PostForm("fareBalance"))

	user := data.UserFetchById(uint(uid))

	if user != nil {
		tx := data.DBGet().Model(user).Updates(map[string]interface{}{
			"venue_fee":    user.VenueFee + venueFee,
			"ball_fee":     user.BallFee + ballFee,
			"training_fee": user.TrainingFee + trainingFee,
			"fare_balance": user.FareBalance + fareBalance,
			"balance":      user.Balance + balance,
		})

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

			go misc.LarkMarkdownChan(fmt.Sprintf("%s 的 %s 金额变动 %.2f 当前 %.2f by %s", user.Name, data.TransactionTypeMap[data.TransactionTypeBalance], balance, user.Balance, admin.Name))
		}

		if fareBalance != 0 {
			if err := data.CreateTransaction(admin.ID, user.ID, 0, data.TransactionTypeFare, fareBalance, user.FareBalance, fmt.Sprintf("admin %s", admin.Name)); err != nil {
				c.String(http.StatusOK, fmt.Sprintf("create transaction failed: %s", tx.Error.Error()))
				return
			}

			go misc.LarkMarkdownChan(fmt.Sprintf("%s 的 %s 金额变动 %.2f 当前 %.2f by %s", user.Name, data.TransactionTypeMap[data.TransactionTypeFare], fareBalance, user.FareBalance, admin.Name))
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
