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
	"os"

	"github.com/gin-gonic/gin"
)

var NoContent = func(context *gin.Context) {
	context.AbortWithStatus(http.StatusNoContent)
}

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
	router.Use(filter.NewRequestIdFilter(), filter.Cors)

	// 图形验证码
	router.GET("/captcha", handler.DefaultCaptcha().Serve)

	// ======================================================================
	// Member Api 接口
	// ======================================================================

	// 登录
	router.POST("/api/sign-in",
		H(handler.DefaultCaptcha().Validate),
		H(api.NewSignInApi().Serve),
	)

	apiRouter := router.Group("/api",
		H(filter.DefaultMemberAuthFilter().Serve), // 认证
	)

	// 文件 API 接口
	{
		apiRouter.GET("/resources/tree", H(api.DefaultResourceApi().Tree))             // 完整的文件树
		apiRouter.GET("/resources", H(api.DefaultResourceApi().List))                  // 资源列表
		apiRouter.GET("/resources/:id", H(api.DefaultResourceApi().Get))               // 读取资源
		apiRouter.POST("/resources/upload", H(api.DefaultResourceApi().Upload))        // 上传单个资源
		apiRouter.POST("/resources/upload/dir", H(api.DefaultResourceApi().UploadDir)) // 上传文件夹
		apiRouter.POST("/resources/mkdir", H(api.DefaultResourceApi().MkDir))          // 创建目录
		apiRouter.POST("/resources/:id/rename", H(api.DefaultResourceApi().Rename))    // 重命名资源
		apiRouter.DELETE("/resources/delete", H(api.DefaultResourceApi().Delete))      // 删除资源
		apiRouter.POST("/resources/move", H(api.DefaultResourceApi().Move))            // 移动资源
		apiRouter.GET("/resources/download", H(api.DefaultResourceApi().Download))     // 下载资源
		apiRouter.POST("/resources/unarchive", NoContent)                              // 解压资源
		apiRouter.GET("/resources/search", NoContent)                                  // 搜索资源
		apiRouter.POST("/resources/share", NoContent)                                  // 分享资源
		apiRouter.GET("/resources/group", NoContent)                                   // 资源分组
	}

	// ======================================================================
	// Manager Api 接口
	// ======================================================================

	// 静态资源在最后
	router.Use(handler.NewFsHandler(
		http.FS(util.Require(func() (*os.Root, error) { // 指定的公共目录优先级最高
			return os.OpenRoot(*config.PublicDir)
		}).FS()),
		http.FS(util.Require(func() (fs.FS, error) { // 嵌入式目录
			return fs.Sub(web.Resource, "resource/public")
		})),
	))
	return router
}
