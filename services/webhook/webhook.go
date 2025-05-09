package webhook

import (
	"Topicgram/services/bot"

	"github.com/gin-gonic/gin"
)

func NewEngine() (router *gin.Engine) {
	router = gin.New()
	router.SetTrustedProxies([]string{"0.0.0.0/0", "::/0"})

	if gin.Mode() == gin.DebugMode {
		router.Use(gin.Logger())
	}

	router.Use(gin.Recovery())

	router.POST("/topicgram/webhook", bot.HookHandler)
	return
}
