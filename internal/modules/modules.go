package modules

import "github.com/gin-gonic/gin"

type Module interface {
	RegisterRoutes(public, protected, admin *gin.RouterGroup)
}
