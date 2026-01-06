package router

import (
	"io/fs"
	"ispace/common/util"
	"ispace/config"
	"ispace/web"
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
	router.GET("/captcha", handler.DefaultCaptcha().Serve)

	// 登录
	router.POST("/api/sign-in",
		// H(handler.DefaultCaptcha().Validate),
		H(api.NewSignInApi().Serve),
	)

	// 静态资源在最后
	router.Use(handler.NewFsHandler(
		http.Dir(*config.PublicDir), // 指定的公共目录优先级最高
		http.FS(util.Require(func() (fs.FS, error) {
			return fs.Sub(web.Resource, "resource/public")
		})),
	))
	return router
}
