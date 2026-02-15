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

	// Global Filter
	router.Use(
		gin.Recovery(),
		filter.NewRequestIdFilter(), // 请求 ID
		filter.RequestTime,          // 请求时间
		filter.TimeZone( // 客户端时区
			constant.Location,
			constant.HttpHeaderTimeZone,
		),
		filter.Cors, // Cors 跨域
	)

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
		memberApi.POST("/resources/upload/flash", H(member.DefaultResourceApi().UploadFlash)) // 快传资源
		memberApi.POST("/resources/upload/dir", H(member.DefaultResourceApi().UploadDir))     // 上传文件夹
		memberApi.POST("/resources/upload/get", H(member.DefaultResourceApi().UploadGet))     // 下载远程资源
		memberApi.POST("/resources/mkdir", H(member.DefaultResourceApi().MkDir))              // 创建目录
		memberApi.POST("/resources/:id/rename", H(member.DefaultResourceApi().Rename))        // 重命名资源
		memberApi.DELETE("/resources", H(member.DefaultResourceApi().Delete))                 // 删除资源
		memberApi.POST("/resources/move", H(member.DefaultResourceApi().Move))                // 移动资源
		memberApi.GET("/resources/download", H(member.DefaultResourceApi().Download))         // 下载资源
		memberApi.GET("/resources/:id/unarchive", H(member.DefaultResourceApi().Unarchive))   // 解压资源
		memberApi.GET("/resources/search", H(member.DefaultResourceApi().Search))             // 搜索资源
		memberApi.GET("/resources/recent", H(member.DefaultResourceApi().Recent))             // 最近上传
		memberApi.GET("/resources/group", H(member.DefaultResourceApi().Group))               // 资源分组
	}

	// TODO 分享 Api
	{

	}

	// 回收站 API
	{
		memberApi.GET("/recycle-bin", H(member.DefaultRecycleBinApi.List)) // 回收站列表
		// TODO 查询资源目录树
		memberApi.DELETE("/recycle-bin", H(member.DefaultRecycleBinApi.Delete))        // 彻底删除文件
		memberApi.POST("/recycle-bin/restore", H(member.DefaultRecycleBinApi.Restore)) // 恢复文件
	}

	// 对象 Api 接口
	{
		memberApi.GET("/objects/sha256/:hash", H(member.DefaultObjectApi.Hash)) // 根据 Hash 查询文件是否存在
	}

	// 设置 Api 接口
	{
		memberApi.GET("/profile", H(member.DefaultProfileApi.Profile))                         // 查询个人信息
		memberApi.PATCH("/profile", H(member.DefaultProfileApi.Update))                        // 更新个人信息
		memberApi.POST("/account/password", H(member.DefaultAccountSettingApi.UpdatePassword)) // 修改账户的密码
	}

	// ======================================================================
	// Manager Api 接口
	// ======================================================================
	managerApi := router.Group("/manager-api")

	managerApi.POST("/sign-in",
		//H(handler.DefaultCaptcha().Validate),
		H(member.DefaultSignInApi.Serve),
	)

	managerApi.Use(
		// H(filter.DefaultManagerAuthFilter().Serve),
		func(context *gin.Context) {
			context.Set(constant.CtxKeySubject, int64(139669356857524224))
		},
	)

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

	// 设置 Api 接口
	{
		managerApi.GET("/profile", H(manager.DefaultProfileApi.Profile))                         // 查询个人信息
		managerApi.PATCH("/profile", H(manager.DefaultProfileApi.Update))                        // 更新个人信息
		managerApi.POST("/account/password", H(manager.DefaultAccountSettingApi.UpdatePassword)) // 修改账户的密码
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
