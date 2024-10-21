package router

import "github.com/gin-gonic/gin"

type Router interface {
	Routers(route gin.IRouter)
}
