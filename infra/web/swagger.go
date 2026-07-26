package web

import (
	"github.com/KNattawat89/hospital-middleware-system/docs"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// registerSwagger serves the generated swagger docs (see `task gen-swagger`)
// at /swagger/index.html.
func registerSwagger(engine *gin.Engine) {
	docs.SwaggerInfo.BasePath = "/"
	engine.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
}
