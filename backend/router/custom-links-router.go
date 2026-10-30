package router

import (
	"github.com/QuantumNous/new-api/controller"
	"github.com/QuantumNous/new-api/middleware"

	"github.com/gin-gonic/gin"
)

// SetCustomLinksRouter registers the same-origin proxy for admin-configured
// sidebar links. Registered before the SPA fallback so /ext/* reaches the
// proxy handler instead of the embedded console.
func SetCustomLinksRouter(router *gin.Engine) {
	ext := router.Group("/ext")
	ext.Use(middleware.AccessTokenAudit())
	{
		ext.GET("/:id", controller.ProxyCustomLink)
		ext.GET("/:id/*path", controller.ProxyCustomLink)
	}
}
