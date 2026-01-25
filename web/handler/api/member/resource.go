package member

import (
	"context"
	"errors"
	"ispace/common"
	"ispace/common/constant"
	"ispace/common/response"
	"ispace/common/util"
	"ispace/db"
	"ispace/repo/model"
	"ispace/store"
	"ispace/web/handler/api"
	"ispace/web/service"
	"mime"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ResourceApi struct {
}

func NewResourceApi() *ResourceApi {
	return &ResourceApi{}
}

// Tree 完整的文件树
func (r ResourceApi) Tree(ctx *gin.Context) (any, error) {
	memberId := ctx.GetInt64(constant.CtxKeySubject)
	result, err := db.Transaction(ctx.Request.Context(), func(ctx context.Context) ([]*api.ResourceTreeResponse, error) {
		return service.DefaultResourceService.Tree(ctx, memberId)
	}, db.TxReadOnly)
	if err != nil {
		return nil, err
	}
	return response.Ok(result), nil
}

// List 资源列表
func (r ResourceApi) List(ctx *gin.Context) (any, error) {

	parentId, err := strconv.ParseInt(ctx.Query("parentId"), 10, 64)
	if err != nil || parentId < 0 {
		parentId = model.DefaultResourceParentId
	}

	request := &api.ResourceListRequest{
		MemberId: ctx.GetInt64(constant.CtxKeySubject),
		ParentId: parentId,
		Dir:      util.BoolQuery(ctx.GetQuery("dir")),
	}

	result, err := db.Transaction(ctx.Request.Context(), func(ctx context.Context) ([]*api.ResourceListResponse, error) {
		return service.DefaultResourceService.List(ctx, request)
	}, db.TxReadOnly)
	if err != nil {
		return nil, err
	}
	return response.Ok(result), nil
}

// Upload 上传文件
func (r ResourceApi) Upload(ctx *gin.Context) (any, error) {

	defer util.SafeClose(ctx.Request.Body)

	// TODO 对于大文件，可以考虑流式处理

	multipartForm, err := ctx.MultipartForm()
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = multipartForm.RemoveAll()
	}()

	// 上传目录
	var parentId = model.DefaultResourceParentId

	parentId, _ = strconv.ParseInt(ctx.Query("parentId"), 10, 64)
	if parentId <= 0 {
		parentId = model.DefaultResourceParentId
	}

	// 会员 ID
	var memberId = ctx.GetInt64(constant.CtxKeySubject)

	for _, files := range multipartForm.File {
		for _, file := range files {
			_, err = db.Transaction(ctx.Request.Context(), func(ctx context.Context) (any, error) {
				return nil, service.DefaultResourceService.Upload(ctx, memberId, parentId, file)
			})
			if err != nil {
				return nil, err
			}
		}
	}
	return response.Ok(nil), nil
}

// Get 读取资
func (r ResourceApi) Get(ctx *gin.Context) (any, error) {
	// 会员 ID
	var memberId = ctx.GetInt64(constant.CtxKeySubject)
	// 资源 ID
	resourceId, _ := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if resourceId < 0 {
		return nil, common.NewServiceError(http.StatusBadRequest, response.Fail(response.CodeBadRequest).WithMessage("非法请求"))
	}

	resource, err := db.Transaction(ctx.Request.Context(), func(ctx context.Context) (struct {
		Title       string
		Compression model.ObjectCompression
		ContentType string
		Status      model.ObjectStatus
		Path        string
	}, error) {
		return service.DefaultResourceService.Get(ctx, memberId, resourceId)
	}, db.TxReadOnly)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			err = common.NewServiceError(http.StatusNotFound, response.Fail(response.CodeNotFound).WithMessage("资源不存在"))
		}
		return nil, err
	}

	// 资源状态判断
	if resource.Status == model.ObjectStatusDisabled {
		return nil, common.NewServiceError(http.StatusBadRequest, response.Fail(response.CodeBadRequest).WithMessage("该资源已被屏蔽"))
	}

	// 打开资源文件
	file, err := store.DefaultStore().Open(resource.Path)
	if err != nil {
		return nil, err
	}
	defer util.SafeClose(file)

	stat, err := file.Stat()
	if err != nil {
		return nil, err
	}

	// 响应客户端
	//ctx.Header("Content-Length", strconv.FormatInt(stat.Size(), 10))
	ctx.Header("Content-Type", resource.ContentType)
	if resource.Compression != model.ObjectCompressionNone {
		ctx.Header("Content-Encoding", string(resource.Compression))
	}
	download := util.BoolQuery(ctx.GetQuery("download"))
	if download != nil && *download {
		ctx.Header("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": resource.Title}))
	}

	/*
		1. 直接 IO Gzip 文件，会导致不能正确处理 Range 请求，例如，会影响到视频文件播放的 Range 请求，且不支持缓存协商。
		2. 使用 http.ServeContent 响应，可以正确处理 Range 和缓存协商，但是问题在于 磁盘文件已经是 gzip 格式，Range 本身没意义。
	*/

	http.ServeContent(ctx.Writer, ctx.Request, resource.Title, stat.ModTime(), file)

	//_, _ = io.Copy(ctx.Writer, file)

	ctx.Abort()
	return nil, nil
}

// MkDir 创建文件夹
func (r ResourceApi) MkDir(ctx *gin.Context) (any, error) {
	request := &api.ResourceMkdirRequest{MemberId: ctx.GetInt64(constant.CtxKeySubject)}
	if err := ctx.ShouldBindJSON(request); err != nil {
		return nil, err
	}
	err := db.TransactionWithOutResult(ctx.Request.Context(), func(ctx context.Context) error {
		return service.DefaultResourceService.Mkdir(ctx, request)
	})
	if err != nil {
		return nil, err
	}
	return response.Ok(nil), nil
}

// Rename 重命名资源
func (r ResourceApi) Rename(ctx *gin.Context) (any, error) {

	id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		return nil, common.NewServiceError(http.StatusBadRequest, response.Fail(response.CodeBadRequest).WithMessage("非法请求"))
	}
	var request = &api.ResourceRenameRequest{
		Id:       id,
		MemberId: ctx.GetInt64(constant.CtxKeySubject),
	}
	if err := ctx.ShouldBindJSON(request); err != nil {
		return nil, err
	}
	err = db.TransactionWithOutResult(ctx.Request.Context(), func(ctx context.Context) error {
		return service.DefaultResourceService.Rename(ctx, request)
	})
	if err != nil {
		return nil, err
	}
	return response.Ok(nil), nil
}

// Delete 删除资源
func (r ResourceApi) Delete(ctx *gin.Context) (any, error) {
	request := &api.ResourceDeleteRequest{
		MemberId: ctx.GetInt64(constant.CtxKeySubject),
	}
	if err := ctx.ShouldBindJSON(request); err != nil {
		return nil, err
	}
	err := db.TransactionWithOutResult(ctx.Request.Context(), func(ctx context.Context) error {
		return service.DefaultResourceService.Delete(ctx, request)
	})
	if err != nil {
		return nil, err
	}
	return response.Ok(nil), nil
}

// Move 移动资源
func (r ResourceApi) Move(ctx *gin.Context) (any, error) {
	request := &api.ResourceMoveRequest{
		MemberId: ctx.GetInt64(constant.CtxKeySubject),
	}

	if err := ctx.ShouldBindJSON(request); err != nil {
		return nil, err
	}
	err := db.TransactionWithOutResult(ctx.Request.Context(), func(ctx context.Context) error {
		return service.DefaultResourceService.Move(ctx, request)
	})
	if err != nil {
		return nil, err
	}
	return response.Ok(nil), nil
}

// UploadDir 上传文件夹
func (r ResourceApi) UploadDir(c *gin.Context) (any, error) {
	defer util.SafeClose(c.Request.Body)
	form, err := c.MultipartForm()
	if err != nil {
		return nil, err
	}

	defer func() {
		_ = form.RemoveAll()
	}()

	// 上传目录
	var parentId = model.DefaultResourceParentId

	parentId, _ = strconv.ParseInt(c.Query("parentId"), 10, 64)
	if parentId <= 0 {
		parentId = model.DefaultResourceParentId
	}

	// 会员 ID
	memberId := c.GetInt64(constant.CtxKeySubject)

	// 目录 & 文件
	var dirs = make(map[string][]*multipart.FileHeader)

	for _, files := range form.File {

		// rootDir 根目录
		var commonRoot = ""

		// 目录下的文件列表
		var fileSlice = make([]*multipart.FileHeader, 0)

		for _, file := range files {
			// 从 Header 中解析出完整带路径的文件名称
			_, params, err := mime.ParseMediaType(file.Header.Get("Content-Disposition"))
			if err != nil {
				return nil, err // 参数解析失败
			}

			// 获取文件的原始名称，且处理跨平台换行符为 /
			filename := strings.ReplaceAll(params["filename"], "\\", constant.Slash)
			if filename == "" {
				return nil, common.NewServiceError(http.StatusBadRequest, response.Fail(response.CodeBadRequest).WithMessage("文件名称不能为空"))
			}

			// 拆分路径中的每个 part
			parts := strings.Split(filename, constant.Slash)

			// 必须要目录开头
			if len(parts) == 0 {
				return nil, common.NewServiceError(http.StatusBadRequest, response.Fail(response.CodeBadRequest).WithMessage("目录不能为空"))
			}

			// 当前根目录
			currentRoot := parts[0]

			// 确定所有文件是否相同的公共目录
			if commonRoot == "" {
				commonRoot = currentRoot
				if strings.TrimSpace(commonRoot) == "" {
					return nil, common.NewServiceError(http.StatusBadRequest, response.Fail(response.CodeBadRequest).WithMessage("目录名称不能为空"))
				}
			} else if commonRoot != currentRoot {
				return nil, common.NewServiceError(http.StatusBadRequest, response.Fail(response.CodeBadRequest).WithMessage("不能包含多个根目录"))
			}

			// 去掉公共目录
			file.Filename = strings.Join(parts[1:], "/")

			fileSlice = append(fileSlice, file)
		}

		if len(fileSlice) == 0 {
			return nil, common.NewServiceError(http.StatusBadRequest, response.Fail(response.CodeBadRequest).WithMessage("文件项目不能为空"))
		}

		if _, ok := dirs[commonRoot]; ok {
			return nil, common.NewServiceError(http.StatusBadRequest,
				response.Fail(response.CodeBadRequest).WithMessage("包含重名目录"))
		}
		dirs[commonRoot] = fileSlice
	}

	err = db.TransactionWithOutResult(c.Request.Context(), func(ctx context.Context) error {
		return service.DefaultResourceService.UploadDir(ctx, memberId, parentId, dirs)
	})
	if err != nil {
		return nil, err
	}
	return response.Ok(nil), nil
}

// Download 下载资源
func (r ResourceApi) Download(g *gin.Context) (any, error) {
	ids := g.QueryArray("resourceId") // 可以有多个
	if len(ids) == 0 {
		return nil, common.NewServiceError(http.StatusBadRequest, response.Fail(response.CodeBadRequest).WithMessage("下载资源不能为空"))
	}
	var resourceId = make([]int64, 0)
	for _, v := range ids {
		id, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return nil, common.NewServiceError(http.StatusBadRequest, response.Fail(response.CodeBadRequest).WithMessage("资源 ID 错误"))
		}
		resourceId = append(resourceId, id)
	}

	// TODO

	return nil, nil
}

var DefaultResourceApi = sync.OnceValue(func() *ResourceApi {
	return NewResourceApi()
})
