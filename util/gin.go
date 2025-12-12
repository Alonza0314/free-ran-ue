package util

import (
	"net/http"

	"github.com/free-ran-ue/free-ran-ue/constant"
	"github.com/gin-gonic/gin"
)

type Route struct {
	Name        string
	Method      string
	Pattern     string
	HandlerFunc gin.HandlerFunc
}

type Routes []Route

func NewGinRouter(apiPrefix constant.API_PREFIX, routes Routes) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)

	router := gin.Default()
	addMiddleware(router)

	group := router.Group(string(apiPrefix))
	addRoutes(router, group, routes)

	return router
}

func addMiddleware(router *gin.Engine) {
	router.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})
}

func addRoutes(router *gin.Engine, group *gin.RouterGroup, routes Routes) {
	for _, route := range routes {
		switch route.Method {
		case http.MethodGet:
			group.GET(route.Pattern, route.HandlerFunc)
		case http.MethodPost:
			group.POST(route.Pattern, route.HandlerFunc)
		case http.MethodPut:
			group.PUT(route.Pattern, route.HandlerFunc)
		case http.MethodDelete:
			group.DELETE(route.Pattern, route.HandlerFunc)
		case http.MethodPatch:
			group.PATCH(route.Pattern, route.HandlerFunc)
		case http.MethodOptions:
			group.OPTIONS(route.Pattern, route.HandlerFunc)
		}
	}
}
