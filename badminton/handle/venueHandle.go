package handle

import (
	"badminton-booking/badminton/data"
	"badminton-booking/badminton/misc"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"strings"
	"time"
)

func init() {
	r := RouterGet()
	r.POST("/venue/create", handleVenueCreate)
	r.POST("/venue/booking", handleVenueBooking)
	r.POST("/venue/limit", handleVenueLimit)
	r.POST("/venue/done", handleVenueDone)
	r.POST("/venue/depart", handleVenueDepart)
	r.POST("/venue/stat", handleVenueStat)
}

func handleVenueCreate(c *gin.Context) {
	data.Locker.Lock()
	defer data.Locker.Unlock()

	ticket := c.PostForm("ticket")

	if ticket == "" {
		c.Status(http.StatusServiceUnavailable)
		return
	}

	user := data.UserFetchByTicket(ticket)

	if user == nil {
		c.Status(http.StatusServiceUnavailable)
		return
	}

	name := c.PostForm("name")
	day := c.PostForm("day")
	desc := c.PostForm("desc")

	if id, err := data.VenueCreate(user.UID, name, day, data.VenueStateRunning, 0, 0, desc); err != nil {
		c.String(http.StatusOK, fmt.Sprintf("create venue failed: %s", err.Error()))
		return
	} else {
		venue := data.VenueFetchById(id)
		msg := venue.Log(user.Name, "创建了场地", time.Now())

		go misc.LarkMarkdown(msg)
		go misc.Wechat(msg)
		go misc.LarkMarkdown(fmt.Sprintf("create <at user_id=\"all\">everyone</at>"))
	}

	c.Redirect(http.StatusMovedPermanently, c.Request.Referer())
}

func handleVenueBooking(c *gin.Context) {
	data.Locker.Lock()
	defer data.Locker.Unlock()

	ticket := c.PostForm("ticket")
	_venueId := c.PostForm("venueId")
	venueId := misc.ToINT(_venueId)
	worker := c.PostForm("worker")
	_state := c.PostForm("state")
	state := data.BookingState(misc.ToINT(_state))

	changed := false
	calculate := false
	var user *data.User
	var venue *data.Venue

	defer func() {
		if changed && user != nil {
			if err := data.BookingCreate(user.UID, venueId, state, time.Now().Unix(), worker); err != nil {
				c.String(http.StatusOK, fmt.Sprintf("create booking failed: %s", err.Error()))
				return
			}

			if calculate && venue != nil {
				bookingSummary := data.BookingSummaryByVenueId(venue.ID)
				bookingSummary.Adjust(venue)
			}
		}

		c.Redirect(http.StatusMovedPermanently, c.Request.Referer())
	}()

	if ticket == "" {
		c.Status(http.StatusServiceUnavailable)
		return
	}

	user = data.UserFetchByTicket(ticket)

	if user == nil {
		c.Status(http.StatusServiceUnavailable)
		return
	}

	venue = data.VenueFetchById(venueId)

	if venue == nil || venue.State != data.VenueStateRunning {
		c.Status(http.StatusServiceUnavailable)
		return
	}

	selectionDesc := venue.Log(user.GetName(worker), fmt.Sprintf("选择了 %s", data.BookingStateMap[state]), time.Now())

	bookingSummary := data.BookingSummaryByVenueId(venueId)

	checkMultipleOfFour := func() {
		okAmount := bookingSummary.AnswerCounter[data.BookingStateMap[data.BookingStateOK]] + 1

		if okAmount%4 != 0 {
			bookingMsg := fmt.Sprintf("[%s %s %s] [%s] 确认. 人数: %d, Limit(%d/%d)", venue.Name, venue.Day, misc.GetWeekDay(venue.Day), user.GetName(worker), okAmount, venue.Amount, venue.Limit)

			go misc.LarkMarkdown(bookingMsg)

			return
		}

		list := bookingSummary.AnswerResponses[data.BookingStateMap[data.BookingStateOK]]
		list = append(list, user.GetName(worker), "4的倍数")

		msg := fmt.Sprintf("[%s %s %s %s] 报名通知 by [%s]\n\n名单：%s", venue.Name, venue.Day, misc.GetWeekDay(venue.Day), venue.Desc, user.Name, strings.Join(list, ", "))
		go misc.LarkMarkdown(msg)
	}

	if venue.Amount == 0 && venue.Limit == 0 {
		changed = true

		if state == data.BookingStateOK {
			checkMultipleOfFour()
		}

		return
	}

	oldState, oldOK := bookingSummary.AnswerValues[user.GetName(worker)]

	answerCounter := bookingSummary.AnswerCounter

	changed = true

	if oldOK && oldState == data.BookingStateOK {
		if oldState == state {
			changed = false
			return
		}

		if answerCounter[data.BookingStateMap[oldState]] <= venue.Amount && answerCounter[data.BookingStateMap[data.BookingStateAuto]] <= 0 {
			state = data.BookingStateExiting
		}

		msg := venue.Log(user.GetName(worker), fmt.Sprintf("From %s To %s", data.BookingStateMap[data.BookingStateOK], data.BookingStateMap[state]), time.Now())

		go misc.LarkMarkdown(msg)

		if answerCounter[data.BookingStateMap[data.BookingStateAuto]] > 0 {
			calculate = true
		}

		return
	}

	if oldOK && oldState == data.BookingStateExiting {
		if oldState == state || state != data.BookingStateOK {
			changed = false

			msg := venue.Log(user.GetName(worker), fmt.Sprintf("下车中禁止操作"), time.Now())
			go misc.LarkMarkdown(msg)

			return
		}
	}

	if state == data.BookingStateOK {
		if answerCounter[data.BookingStateMap[data.BookingStateOK]] >= venue.Limit {
			state = data.BookingStateAuto

			msg := venue.Log(user.GetName(worker), fmt.Sprintf("人员已满 自动变更为 %s", data.BookingStateMap[state]), time.Now())
			go misc.LarkMarkdown(msg)
		} else {
			checkMultipleOfFour()
		}

		if answerCounter[data.BookingStateMap[data.BookingStateExiting]] > 0 {
			calculate = true
		}

		return
	}

	go misc.LarkMarkdown(selectionDesc)
}

func handleVenueLimit(c *gin.Context) {
	data.Locker.Lock()
	defer data.Locker.Unlock()

	ticket := c.PostForm("ticket")
	_venueId := c.PostForm("venueId")
	venueId := misc.ToINT(_venueId)
	_amount := c.PostForm("amount")
	amount := misc.ToINT(_amount)
	_limit := c.PostForm("limit")
	limit := misc.ToINT(_limit)
	name := c.PostForm("name")
	day := c.PostForm("day")
	desc := c.PostForm("desc")

	if ticket == "" {
		c.Status(http.StatusServiceUnavailable)
		return
	}

	user := data.UserFetchByTicket(ticket)

	if user == nil {
		c.Status(http.StatusServiceUnavailable)
		return
	}

	venue := data.VenueFetchById(venueId)

	if venue == nil || venue.State != data.VenueStateRunning {
		c.Status(http.StatusServiceUnavailable)
		return
	}

	if user.State != data.UserStateAdmin && user.UID != venue.Owner {
		c.Status(http.StatusServiceUnavailable)
		return
	}

	if err := data.VenueUpdateDetail(venueId, amount, limit, name, day, desc); err != nil {
		c.String(http.StatusOK, fmt.Sprintf("update venue limit failed: %s", err.Error()))
		return
	}

	venue.Name = name
	venue.Desc = desc
	venue.Day = day

	msg := venue.Log(user.Name, fmt.Sprintf("更新了场地信息 limit(%d/%d) desc(%s)", amount, limit, desc), time.Now())

	if venue.Amount != amount || venue.Limit != limit {
		go misc.LarkMarkdown(msg)

		venue.Amount = amount
		venue.Limit = limit

		bookingSummary := data.BookingSummaryByVenueId(venueId)
		bookingSummary.Adjust(venue)
	}

	c.Redirect(http.StatusMovedPermanently, c.Request.Referer())
	return
}

func handleVenueDone(c *gin.Context) {
	data.Locker.Lock()
	defer data.Locker.Unlock()

	ticket := c.PostForm("ticket")
	_venueId := c.PostForm("venueId")
	venueId := misc.ToINT(_venueId)
	venueFee := misc.ToFloat32(c.PostForm("venueFee"))
	ballFee := misc.ToFloat32(c.PostForm("ballFee"))
	trainingFee := misc.ToFloat32(c.PostForm("trainingFee"))

	if ticket == "" {
		c.Status(http.StatusServiceUnavailable)
		return
	}

	user := data.UserFetchByTicket(ticket)

	if user == nil {
		c.Status(http.StatusServiceUnavailable)
		return
	}

	venue := data.VenueFetchById(venueId)

	if venue == nil || venue.State != data.VenueStateRunning {
		c.Status(http.StatusServiceUnavailable)
		return
	}

	if user.State != data.UserStateAdmin && user.UID != venue.Owner {
		c.Status(http.StatusServiceUnavailable)
		return
	}

	if venueFee+ballFee+trainingFee == 0 {
		venue.State = data.VenueStateCancel

		if err := data.VenueStateUpdate(venueId, data.VenueStateCancel, 0, 0, 0); err != nil {
			panic(err)
		} else {
			if err := data.BookingDelByVenueId(venueId); err != nil {
				panic(err)
			}
		}

		msg := venue.Log(user.Name, fmt.Sprintf("场地已取消"), time.Now())

		venue.Log("", "", time.Now())

		go misc.LarkMarkdown(msg)
	} else {
		venue.State = data.VenueStateDone
		venue.Fee = venueFee
		venue.BallFee = ballFee
		venue.TrainingFee = trainingFee

		if err := data.VenueStateUpdate(venueId, data.VenueStateDone, venueFee, ballFee, trainingFee); err != nil {
			panic(err)
		}

		bookingSummary := data.BookingSummaryByVenueId(venueId)

		list := bookingSummary.AnswerResponses[data.BookingStateMap[data.BookingStateOK]]
		list = append(list, bookingSummary.AnswerResponses[data.BookingStateMap[data.BookingStateExiting]]...)

		avgVenueFee := venueFee / float32(len(list))
		avgBallFee := ballFee / float32(len(list))
		avgTrainingFee := trainingFee / float32(len(list))

		msg := venue.Log(user.Name, fmt.Sprintf("场地已结束，人均约 %.2f 元. 人员：%s", avgVenueFee+ballFee, strings.Join(list, ", ")), time.Now())

		for _, name := range list {
			participant := data.UserFetchByName(name)

			if participant == nil {
				continue
			}

			participant.VenueFee += avgVenueFee
			participant.BallFee += avgBallFee
			participant.TrainingFee += avgTrainingFee

			if err := data.UserUpdateFee(name, participant.VenueFee, participant.BallFee, participant.TrainingFee); err != nil {
				panic(err)
			}
		}

		go misc.LarkMarkdown(msg)
	}

	c.Redirect(http.StatusMovedPermanently, c.Request.Referer())
}

func handleVenueDepart(c *gin.Context) {
	data.Locker.Lock()
	defer data.Locker.Unlock()

	ticket := c.PostForm("ticket")
	_venueId := c.PostForm("venueId")
	venueId := misc.ToINT(_venueId)

	if ticket == "" {
		c.Status(http.StatusServiceUnavailable)
		return
	}

	user := data.UserFetchByTicket(ticket)

	if user == nil {
		c.Status(http.StatusServiceUnavailable)
		return
	}

	venue := data.VenueFetchById(venueId)

	if venue == nil || venue.State != data.VenueStateRunning {
		c.Status(http.StatusServiceUnavailable)
		return
	}

	bookingSummary := data.BookingSummaryByVenueId(venueId)

	list := bookingSummary.AnswerResponses[data.BookingStateMap[data.BookingStateOK]]

	if bookingSummary.AnswerCounter[data.BookingStateMap[data.BookingStateOK]]%2 != 0 {
		list = append(list, "奇数出发")
	} else {
		list = append(list, "偶数出发")
	}

	list = append(list, fmt.Sprintf("%d人", bookingSummary.AnswerCounter[data.BookingStateMap[data.BookingStateOK]]))

	msg := fmt.Sprintf("[%s %s %s %s] 出发通知 by [%s]\n\n名单：%s", venue.Name, venue.Day, misc.GetWeekDay(venue.Day), venue.Desc, user.Name, strings.Join(list, ", "))

	go misc.LarkMarkdown(msg)

	c.Redirect(http.StatusMovedPermanently, c.Request.Referer())
}

func handleVenueStat(c *gin.Context) {
	data.Locker.Lock()
	defer data.Locker.Unlock()

	ticket := c.PostForm("ticket")
	_venueId := c.PostForm("venueId")
	venueId := misc.ToINT(_venueId)

	if ticket == "" {
		c.Status(http.StatusServiceUnavailable)
		return
	}

	user := data.UserFetchByTicket(ticket)

	if user == nil {
		c.Status(http.StatusServiceUnavailable)
		return
	}

	venue := data.VenueFetchById(venueId)

	if venue == nil || venue.State != data.VenueStateRunning {
		c.Status(http.StatusServiceUnavailable)
		return
	}

	bookingSummary := data.BookingSummaryByVenueId(venueId)

	users := data.UserFetchAll()
	var unselected []string

	for _, user := range users {
		if user.State == data.UserStateZombie {
			continue
		}

		if _, ok := bookingSummary.AnswerValues[user.Name]; !ok {
			unselected = append(unselected, user.Name)
		}
	}

	unselectedDesc := ""

	okAmount := bookingSummary.AnswerCounter[data.BookingStateMap[data.BookingStateOK]]
	autoAmount := bookingSummary.AnswerCounter[data.BookingStateMap[data.BookingStateAuto]]

	if venue.Limit == 0 {
		unselectedDesc = "此时报名无限制"
	} else {
		unselectedDesc = fmt.Sprintf("还余 %d 个位置", venue.Limit-okAmount-autoAmount)
	}

	unselected = append(unselected, unselectedDesc)

	if autoAmount > 0 {
		if autoAmount%2 != 0 {
			unselected = append(unselected, "替补奇数")
		} else {
			unselected = append(unselected, "替补偶数")
		}
	}

	if okAmount%2 != 0 {
		unselected = append(unselected, "确认奇数")
	} else {
		unselected = append(unselected, "确认偶数")
	}

	go misc.LarkMarkdown(fmt.Sprintf(
		"[%s %s %s %s] 统计通知 by [%s]\nLimit: %d/%d\n\n统计：%s\n名单：%s\n未选择：%s",
		venue.Name,
		venue.Day,
		misc.GetWeekDay(venue.Day),
		venue.Desc,
		user.Name,
		venue.Amount,
		venue.Limit,
		misc.ToJson(bookingSummary.AnswerCounter),
		misc.ToJson(bookingSummary.AnswerResponses),
		strings.Join(unselected, ", "),
	))

	c.Redirect(http.StatusMovedPermanently, c.Request.Referer())
}
