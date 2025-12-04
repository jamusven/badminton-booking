package main

import (
	"badminton-booking/badminton/data"
	"badminton-booking/badminton/handle"
	"badminton-booking/badminton/misc"
	"badminton-booking/badminton/shard"
	"badminton-booking/static"
	"flag"
	"fmt"
	"html/template"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
)

var debug = flag.Bool("debug", false, "debug mode")

func init() {
	gin.SetMode(gin.ReleaseMode)
}

func main() {
	flag.Parse()

	r := handle.RouterGet()

	templ := template.New("").Funcs(template.FuncMap{
		"sha1":          misc.Sha1,
		"toString":      misc.ToString,
		"toIntFromUint": misc.ToIntFromUint,
		"toJson":        misc.ToJson,
		"getWeekDay":    misc.GetWeekDay,
		"now":           misc.Now,
		"cent2Yuan":     misc.Cent2Yuan,
		"substr":        misc.Substr,
		"str2Up":        strings.ToUpper,
	})

	if *debug {
		templ = template.Must(templ.ParseGlob("../static/templates/*.html"))
		r.SetHTMLTemplate(templ)

		r.Static("/static", "../static")
	} else {
		templ, _ = templ.ParseFS(static.FS, "templates/*.html")
		r.SetHTMLTemplate(templ)

		r.StaticFS("/static", http.FS(static.FS))
	}

	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})

	r.GET("/favicon.ico", func(context *gin.Context) {
		context.Data(http.StatusOK, "image/x-icon", static.FaviconBytes)
	})

	r.Static(data.LogDir, data.LogDir)

	r.Run(fmt.Sprintf(":%d", shard.SettingInstance.Port))
}
