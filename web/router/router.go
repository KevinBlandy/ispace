package router

import (
	"io/fs"
	"ispace/common/constant"
	"ispace/common/types"
	"ispace/common/util"
	"ispace/config"
	"ispace/web"
	"ispace/web/filter"
	"ispace/web/handler"
	"ispace/web/handler/api/manager"
	"ispace/web/handler/api/member"
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

	memberApi := router.Group("/api")

	// 登录
	memberApi.POST("/sign-in",
		H(handler.DefaultCaptcha().Validate), // 验证码
		H(member.DefaultSignInApi.Serve),
	)

	memberApi.Use(
		func(context *gin.Context) {
			context.Set(constant.CtxKeySubject, int64(10000))
		},
		//H(filter.DefaultMemberAuthFilter().Serve), // 认证
	)

	// 文件 API 接口
	{
		memberApi.GET("/resources/tree", H(member.DefaultResourceApi().Tree))                 // 完整的文件树
		memberApi.GET("/resources", H(member.DefaultResourceApi().List))                      // 资源列表
		memberApi.GET("/resources/:id", H(member.DefaultResourceApi().Get))                   // 读取资源
		memberApi.POST("/resources/upload", H(member.DefaultResourceApi().Upload))            // 上传单个资源
		memberApi.POST("/resources/upload/flash", H(member.DefaultResourceApi().FlashUpload)) // 快传资源
		memberApi.POST("/resources/upload/dir", H(member.DefaultResourceApi().UploadDir))     // 上传文件夹
		memberApi.POST("/resources/mkdir", H(member.DefaultResourceApi().MkDir))              // 创建目录
		memberApi.POST("/resources/:id/rename", H(member.DefaultResourceApi().Rename))        // 重命名资源
		memberApi.DELETE("/resources/delete", H(member.DefaultResourceApi().Delete))          // 删除资源
		memberApi.POST("/resources/move", H(member.DefaultResourceApi().Move))                // 移动资源
		memberApi.GET("/resources/download", H(member.DefaultResourceApi().Download))         // 下载资源
		memberApi.GET("/resources/unarchive/:id", H(member.DefaultResourceApi().Unarchive))   // 解压资源
		memberApi.GET("/resources/search", NoContent)                                         // 搜索资源
		memberApi.POST("/resources/share", NoContent)                                         // 分享资源
		memberApi.GET("/resources/group", NoContent)                                          // 资源分组
	}

	// 文件 Api 接口
	{
		memberApi.GET("/objects/sha256/:hash", H(member.DefaultObjectApi.Hash)) // 根据 Hash 查询文件是否存在
	}

	// ======================================================================
	// Manager Api 接口
	// ======================================================================
	managerApi := router.Group("/manager-api")

	managerApi.POST("/sign-in",
		//H(handler.DefaultCaptcha().Validate),
		H(member.DefaultSignInApi.Serve),
	)

	managerApi.Use(H(filter.DefaultManagerAuthFilter().Serve))

	// 会员管理
	{
		managerApi.GET("/members", H(manager.DefaultMemberApi.List))         // 会员列表
		managerApi.POST("/members", H(manager.DefaultMemberApi.Create))      // 创建会员
		managerApi.PATCH("/members/:id", H(manager.DefaultMemberApi.Update)) // 更新会员
		managerApi.DELETE("/members", H(manager.DefaultMemberApi.Delete))    // 删除会员
	}

	// 资源管理
	{
		managerApi.GET("/objects", H(manager.DefaultObjectApi.List))         // 资源列表
		managerApi.PATCH("/objects/:id", H(manager.DefaultObjectApi.Update)) // 更新资源
		managerApi.DELETE("/objects", H(manager.DefaultObjectApi.Delete))    // 删除资源
	}

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
