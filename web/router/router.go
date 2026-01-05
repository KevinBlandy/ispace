package router

import (
	"ispace/web/filter"
	"ispace/web/handler"
	"ispace/web/handler/api"
	"net/http"

	"github.com/gin-gonic/gin"
)

func New() http.Handler {
	router := gin.New()
	router.RedirectTrailingSlash = false
	router.HandleMethodNotAllowed = true

	// 404
	router.NoRoute(handler.NotFound)
	// 405
	router.NoMethod(handler.MethodNotAllowed)

	// 请求 ID
	router.Use(filter.NewRequestIdFilter())

	// 图形验证码
	router.GET("/captcha", handler.DefaultCaptcha.Serve)

	// 静态资源在最后
	router.Use(handler.DefaultFsHandler)

	// 登录
	router.POST("/api/sign-in",
		// H(handler.DefaultCaptcha.Validate),
		H(api.NewSignInApi().Serve),
	)

	return router
}
