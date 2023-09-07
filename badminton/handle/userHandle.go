package handle

import (
	"badminton-booking/badminton/data"
	"badminton-booking/badminton/misc"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"time"
)

func init() {
	r := RouterGet()

	r.POST("/user/login", handleUserLogin)
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
