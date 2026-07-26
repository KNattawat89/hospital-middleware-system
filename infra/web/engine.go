package web

import "github.com/gin-gonic/gin"

func NewEngine() *gin.Engine {
	engine := gin.New()
	engine.Use(gin.Recovery())
	return engine
}
