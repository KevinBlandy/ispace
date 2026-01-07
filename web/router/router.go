package router

import (
	"io/fs"
	"ispace/common/types"
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
	router.MaxMultipartMemory = int64(types.MB) // 超过 1Mb 则 io 到磁盘

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

	// Api 接口
	apiRouter := router.Group("/api", H(filter.AuthFilter))

	// 文件 API 接口
	{
		apiRouter.GET("/resources", H(api.DefaultResourceApi().List))    // 文件列表
		apiRouter.POST("/resources", H(api.DefaultResourceApi().Upload)) // 上传文件
	}

	// 静态资源在最后
	router.Use(handler.NewFsHandler(
		http.Dir(*config.PublicDir), // 指定的公共目录优先级最高
		http.FS(util.Require(func() (fs.FS, error) {
			return fs.Sub(web.Resource, "resource/public")
		})),
	))
	return router
}
