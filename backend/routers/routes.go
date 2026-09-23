package routers

import (
	"doc-manager/controllers"
	_ "doc-manager/docs"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func InitRouters() *gin.Engine {
	r := gin.Default()

	// Swagger documentation endpoint
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	apis := r.Group("/api")
	{
		apis.POST("/documents", controllers.AcceptDocument)
		apis.GET("/documents/:id/events", controllers.GetDocumentEvents)
	}

	return r
}
