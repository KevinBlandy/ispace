package api

import (
	"context"
	"ispace/common"
	"ispace/common/constant"
	"ispace/common/response"
	"ispace/common/util"
	"ispace/config"
	"ispace/db"
	"ispace/repo/model"
	"ispace/web"
	"ispace/web/service"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync"

	"github.com/gin-gonic/gin"
)

type ResourceApi struct {
}

func NewResourceApi() *ResourceApi {
	return &ResourceApi{}
}

// Tree 完整的文件树
func (r ResourceApi) Tree(ctx *gin.Context) (any, error) {
	memberId := ctx.GetInt64(constant.CtxKeySubject)
	result, err := db.Transaction(ctx.Request.Context(), func(ctx context.Context) ([]*web.ResourceTreeApiResponse, error) {
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

	request := &web.ResourceListApiRequest{
		MemberId: ctx.GetInt64(constant.CtxKeySubject),
		ParentId: parentId,
		Dir:      util.BoolQuery(ctx.GetQuery("dir")),
	}

	result, err := db.Transaction(ctx.Request.Context(), func(ctx context.Context) ([]*web.ResourceListApiResponse, error) {
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
	if parentId < 0 {
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
		Path        string
	}, error) {
		return service.DefaultResourceService.Get(ctx, memberId, resourceId)
	}, db.TxReadOnly)
	if err != nil {
		return nil, err
	}

	// 打开资源文件
	file, err := os.Open(filepath.Join(*config.StoreDir, filepath.FromSlash(resource.Path)))
	if err != nil {
		return nil, err
	}
	defer util.SafeClose(file)
	stat, err := file.Stat()
	if err != nil {
		return nil, err
	}

	// 响应客户端
	ctx.Header("Content-Length", strconv.FormatInt(stat.Size(), 10))
	ctx.Header("Content-Type", resource.ContentType)
	if resource.Compression != model.ObjectCompressionNone {
		ctx.Header("Content-Encoding", string(resource.Compression))
	}
	download := util.BoolQuery(ctx.GetQuery("download"))
	if download != nil && *download {
		ctx.Header("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": resource.Title}))
	}

	http.ServeContent(ctx.Writer, ctx.Request, resource.Title, stat.ModTime(), file)

	// _, _ = io.Copy(ctx.Writer, file)

	ctx.Abort()
	return nil, nil
}

// MkDir 创建文件夹
func (r ResourceApi) MkDir(ctx *gin.Context) (any, error) {
	var request = &web.ResourceMkdirRequest{MemberId: ctx.GetInt64(constant.CtxKeySubject)}
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
	var request = &web.ResourceRenameRequest{
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
	var request = &web.ResourceDeleteRequest{
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
	var request = &web.ResourceMoveRequest{
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

var DefaultResourceApi = sync.OnceValue(func() *ResourceApi {
	return NewResourceApi()
})
