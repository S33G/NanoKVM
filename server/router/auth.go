package router

import (
	"github.com/gin-gonic/gin"

	"NanoKVM-Server/middleware"
	"NanoKVM-Server/service/auth"
)

func authRouter(r *gin.Engine) {
	service := auth.NewService()

	r.POST("/api/auth/login", service.Login) // login
	r.GET("/api/auth/config", service.GetPublicConfig)
	r.GET("/api/auth/session", service.GetSession)
	r.GET("/api/auth/oidc/login", service.OIDCLogin)
	r.GET("/api/auth/oidc/callback", service.OIDCCallback)

	api := r.Group("/api").Use(middleware.CheckToken())

	api.GET("/auth/password", service.IsPasswordUpdated) // is password updated
	api.GET("/auth/account", service.GetAccount)         // get account
	api.POST("/auth/password", service.ChangePassword)   // change password
	api.POST("/auth/logout", service.Logout)             // logout
}
