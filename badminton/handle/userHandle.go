package handle

import (
	"badminton-booking/badminton/data"
	"badminton-booking/badminton/misc"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"net/http"
	"strings"
	"time"
)

func init() {
	r := RouterGet()

	r.POST("/user/login", handleUserLogin)
	r.GET("/user/transaction", handleUserTransaction)
	r.POST("/user/transfer", handleUserTransfer)
}

func handleUserLogin(c *gin.Context) {
	name := c.PostForm("name")
	mobile := c.PostForm("mobile")

	user := data.UserFetchByName(name)

	if user == nil || user.Mobile != mobile {
		c.Status(http.StatusServiceUnavailable)
		return
	}

	ticket := misc.Sha1(fmt.Sprintf("%s%d", name, time.Now().UnixNano()))

	data.TicketSet(ticket, user.Name)

	c.Redirect(http.StatusMovedPermanently, fmt.Sprintf("/list?ticket=%s", ticket))
}

func handleUserTransaction(c *gin.Context) {
	ticket := c.Query("ticket")
	limit := misc.ToINT(c.Query("limit"))

	if limit == 0 {
		limit = 200
	}

	if ticket == "" {
		c.Status(http.StatusServiceUnavailable)
		return
	}

	user := data.UserFetchByTicket(ticket)

	if user == nil {
		c.Status(http.StatusServiceUnavailable)
		return
	}

	var transactions []*data.Transaction

	tx := data.DBGet().Order("id desc").Limit(limit).Find(&transactions, "uid = ?", user.ID)

	if tx.Error != nil && errors.Is(tx.Error, gorm.ErrRecordNotFound) {
		c.Status(http.StatusServiceUnavailable)
		return
	}

	var records = []string{strings.Join([]string{"时间", "类型", "余额", "金额", "备注"}, ",")}

	for _, transaction := range transactions {
		records = append(records, strings.Join([]string{
			transaction.CreatedAt.Format("2006-01-02 15:04:05"),
			data.TransactionTypeMap[transaction.Type],
			fmt.Sprintf("%.2f", transaction.CurrentAmount),
			fmt.Sprintf("%.2f", transaction.ChangeAmount),
			transaction.Desc,
		}, ","))
	}

	c.String(http.StatusOK, strings.Join(records, "\n"))
}

func handleUserTransfer(c *gin.Context) {
	ticket := c.PostForm("ticket")
	transactionType := data.TransactionType(misc.ToINT(c.PostForm("transactionType")))
	amount := misc.ToFloat32(c.PostForm("amount"))
	targetUID := uint(misc.ToINT(c.PostForm("targetUID")))

	if ticket == "" {
		c.Status(http.StatusServiceUnavailable)
		return
	}

	user := data.UserFetchByTicket(ticket)

	if user == nil {
		c.Status(http.StatusServiceUnavailable)
		return
	}

	targetUser := data.UserFetchById(targetUID)

	if user == nil {
		c.Status(http.StatusServiceUnavailable)
		return
	}

	if user.ID == targetUser.ID {
		c.Status(http.StatusServiceUnavailable)
		return
	}

	var balance, targetBalance float32

	userChanges := map[string]interface{}{}
	targetUserChanges := map[string]interface{}{}

	switch true {
	case transactionType == data.TransactionTypeBalance:
		balance = user.Balance
		targetBalance = targetUser.Balance

		userChanges["balance"] = balance - amount
		targetUserChanges["balance"] = targetUser.Balance + amount

	case transactionType == data.TransactionTypeFare:
		balance = user.FareBalance
		targetBalance = targetUser.FareBalance

		userChanges["fare_balance"] = user.FareBalance - amount
		userChanges["fare_fee"] = user.FareFee + amount

		targetUserChanges["fare_balance"] = targetUser.FareBalance + amount
	default:
		c.Status(http.StatusServiceUnavailable)
		return
	}

	if balance < amount {
		go misc.LarkMarkdownChan(fmt.Sprintf("%s 向 %s %s转账 %.2f 时 余额不足, 当前：%.2f", user.Name, targetUser.Name, data.TransactionTypeMap[transactionType], amount, balance))

		c.String(http.StatusOK, "余额不足")
		return
	}

	misc.LarkMarkdownChan(fmt.Sprintf("%s 向 %s %s转账 %.2f 当前：%.2f", user.Name, targetUser.Name, data.TransactionTypeMap[transactionType], amount, balance-amount))

	if tx := data.DBGet().Model(user).Updates(userChanges); tx.Error != nil {
		c.String(http.StatusOK, tx.Error.Error())
		return
	}

	if tx := data.DBGet().Model(targetUser).Updates(targetUserChanges); tx.Error != nil {
		c.String(http.StatusOK, tx.Error.Error())
		return
	}

	if err := data.CreateTransaction(user.ID, user.ID, 0, transactionType, amount, balance-amount, fmt.Sprintf("转账给 %s", targetUser.Name)); err != nil {
		c.String(http.StatusOK, err.Error())
		return
	}

	if err := data.CreateTransaction(user.ID, targetUser.ID, 0, transactionType, amount, targetBalance+amount, fmt.Sprintf("%s 的转账", user.Name)); err != nil {
		c.String(http.StatusOK, err.Error())
		return
	}

	c.Redirect(http.StatusMovedPermanently, c.Request.Referer())
}
