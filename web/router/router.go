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

	none := func(context *gin.Context) {}

	// 文件 API 接口
	{
		apiRouter.GET("/resources/tree", H(api.DefaultResourceApi().Tree))          // 完整的文件树
		apiRouter.GET("/resources", H(api.DefaultResourceApi().List))               // 资源列表
		apiRouter.GET("/resources/:id", H(api.DefaultResourceApi().Get))            // 读取资源
		apiRouter.POST("/resources", H(api.DefaultResourceApi().Upload))            // 上传资源
		apiRouter.POST("/resources/mkdir", H(api.DefaultResourceApi().MkDir))       // 创建目录
		apiRouter.POST("/resources/:id/rename", H(api.DefaultResourceApi().Rename)) // 重命名资源
		apiRouter.DELETE("/resources", H(api.DefaultResourceApi().Delete))          // 删除资源
		apiRouter.POST("/resources/move", H(api.DefaultResourceApi().Move))         // 移动资源
		apiRouter.GET("/resources/download", none)                                  // 下载资源
		apiRouter.POST("/resources/archive", none)                                  // 归档资源
		apiRouter.GET("/resources/search", none)                                    // 搜索资源
		apiRouter.POST("/resources/share", none)                                    // 分享资源
		apiRouter.GET("/resources/group/:type", none)                               // 资源分组
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
