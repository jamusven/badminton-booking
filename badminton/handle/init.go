package handle

import "github.com/gin-gonic/gin"

var router *gin.Engine

const title = "羽林军"

func RouterGet() *gin.Engine {
	if router != nil {
		return router
	}

	router = gin.Default()

	return router
}
