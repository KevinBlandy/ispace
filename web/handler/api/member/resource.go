package member

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"ispace/common"
	"ispace/common/constant"
	"ispace/common/page"
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
	"net/url"
	"os"
	"path"
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
	if err != nil || parentId < 1 {
		parentId = model.DefaultResourceParentId
	}

	request := &api.ResourceListRequest{
		MemberId: ctx.GetInt64(constant.CtxKeySubject),
		ParentId: parentId,
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
				return nil, service.DefaultResourceService.UploadMultipart(ctx, memberId, parentId, file)
			})
			if err != nil {
				return nil, err
			}
		}
	}
	return response.Ok(nil), nil
}

// Content 读取资
func (r ResourceApi) Content(ctx *gin.Context) (any, error) {
	// 会员 ID
	var memberId = ctx.GetInt64(constant.CtxKeySubject)
	// 资源 ID
	resourceId, _ := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if resourceId < 0 {
		return nil, common.NewServiceError(http.StatusBadRequest, response.Fail(response.CodeBadRequest).WithMessage("非法请求"))
	}

	resource, err := db.Transaction(ctx.Request.Context(), func(ctx context.Context) (struct {
		Id          int64
		Title       string
		Compression model.ObjectCompression
		ContentType string
		Status      model.ObjectStatus
		Path        string
	}, error) {
		return service.DefaultResourceService.Get(ctx, memberId, resourceId)
	}, db.TxReadOnly)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	if resource.Id < 1 {
		return nil, common.NewServiceError(http.StatusNotFound, response.Fail(response.CodeNotFound).WithMessage("资源不存在"))
	}

	// 资源状态判断
	if resource.Status == model.ObjectStatusDisabled {
		return nil, common.NewServiceError(http.StatusForbidden, response.Fail(response.CodeForbidden).WithMessage("该资源已被屏蔽"))
	}

	if err := store.DefaultStore().ServeContent(ctx.Writer, ctx.Request, &store.File{
		Title:       resource.Title,
		Compression: resource.Compression,
		ContentType: resource.ContentType,
		Path:        resource.Path,
	}); err != nil {
		return nil, err
	}
	ctx.Abort()
	return nil, nil
}

// MkDir 创建文件夹
func (r ResourceApi) MkDir(ctx *gin.Context) (any, error) {
	request := &api.ResourceMkdirRequest{MemberId: ctx.GetInt64(constant.CtxKeySubject)}
	if err := ctx.ShouldBindJSON(request); err != nil {
		return nil, err
	}

	request.ParentId, _ = strconv.ParseInt(ctx.Query("parentId"), 10, 64)
	if request.ParentId < 1 {
		request.ParentId = model.DefaultResourceParentId
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
		return service.DefaultResourceService.MoveToRecycleBin(ctx, request)
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

// UploadFlash ⚡️秒传
func (r ResourceApi) UploadFlash(g *gin.Context) (any, error) {
	var request = &api.ResourceFlashUploadRequest{}
	if err := g.ShouldBindJSON(request); err != nil {
		return nil, err
	}
	memberId := g.GetInt64(constant.CtxKeySubject)
	parentId, err := strconv.ParseInt(g.Query("parentId"), 10, 64)
	if err != nil || parentId < 0 {
		parentId = model.DefaultResourceParentId
	}
	err = db.TransactionWithOutResult(g.Request.Context(), func(ctx context.Context) error {
		return service.DefaultResourceService.FlashUpload(ctx, request, memberId, parentId)
	})
	if err != nil {
		return nil, err
	}
	return response.Ok(nil), nil
}

// Download 下载资源
func (r ResourceApi) Download(g *gin.Context) (any, error) {
	ids := g.QueryArray("id") // 可以有多个
	if len(ids) == 0 {
		return nil, common.NewServiceError(http.StatusBadRequest, response.Fail(response.CodeBadRequest).WithMessage("下载资源不能为空"))
	}

	// 查询要下载的资源列表
	resourceIds, err := util.Int64SliceQuery(ids)
	if err != nil {
		return nil, common.NewServiceError(http.StatusBadRequest, response.Fail(response.CodeBadRequest).WithMessage("非法请求"))
	}

	// 检索树形结构
	tree, err := db.Transaction(g.Request.Context(), func(ctx context.Context) ([]*store.DownloadTree, error) {
		return service.DefaultResourceService.Download(ctx, g.GetInt64(constant.CtxKeySubject), resourceIds)
	})

	if err != nil {
		return nil, err
	}

	// 执行下载
	if err := store.DefaultStore().Downloads(g.Writer, tree...); err != nil {
		return nil, err
	}

	g.Abort()

	return nil, nil
}

// Unarchive 在线解压资源
func (r ResourceApi) Unarchive(g *gin.Context) (any, error) {
	resourceId, _ := strconv.ParseInt(g.Param("id"), 10, 64)
	if resourceId < 1 {
		return nil, common.NewServiceError(http.StatusBadRequest, response.Fail(response.CodeBadRequest).WithMessage("非法请求"))
	}

	// 检索资源
	resource, err := db.Transaction(g.Request.Context(), func(ctx context.Context) (struct {
		Id          int64
		Title       string
		Compression model.ObjectCompression
		ContentType string
		Status      model.ObjectStatus
		Path        string
	}, error) {
		return service.DefaultResourceService.Get(ctx, g.GetInt64(constant.CtxKeySubject), resourceId)
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
	// 必须是 zip 文件
	if !strings.HasPrefix(resource.ContentType, "application/zip") {
		return nil, common.NewServiceError(http.StatusBadRequest, response.Fail(response.CodeBadRequest).WithMessage("不支持的压缩格式"))
	}

	file := strings.TrimSpace(g.Query("file"))

	// 返回树结构
	if file == "" {
		ret, err := store.DefaultStore().ArchiveTree(resource.Path)
		if err != nil {
			return nil, err
		}
		return response.Ok(ret), nil
	}

	// 读取文件
	g.Abort()

	return nil, store.DefaultStore().ServeArchiveFile(g.Writer, g.Request, resource.Path, file)
}

// UploadGet 从远程服务器下载资源
func (r ResourceApi) UploadGet(g *gin.Context) (any, error) {

	objectUrl, err := url.Parse(g.Query("url"))
	if err != nil {
		return nil, common.NewServiceError(http.StatusBadRequest, response.Fail(response.CodeBadRequest).WithMessage("资源链接解析异常"))
	}

	if objectUrl.Scheme != "http" && objectUrl.Scheme != "https" {
		return nil, common.NewServiceError(http.StatusBadRequest, response.Fail(response.CodeBadRequest).WithMessage("只能下载 http/https 协议的资源"))
	}

	memberId := g.GetInt64(constant.CtxKeySubject)
	parentId, _ := strconv.ParseInt(g.Query("parentId"), 10, 64)
	if parentId < 1 {
		parentId = model.DefaultResourceParentId
	}

	// 创建临时文件
	tmpFile, err := os.CreateTemp("", strconv.FormatInt(memberId, 10)+"-*")
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = os.RemoveAll(tmpFile.Name()) // 始终删除临时文件
	}()
	defer util.SafeClose(tmpFile)

	// 请求资源
	client := &http.Client{}
	req, err := http.NewRequest(http.MethodGet, objectUrl.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "ispace/object-downloader")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer util.SafeClose(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, common.NewServiceError(http.StatusBadRequest, response.Fail(response.CodeBadRequest).WithMessage("资源下载失败，状态码："+resp.Status))
	}

	// io 到临时文件
	if _, err := io.Copy(tmpFile, resp.Body); err != nil {
		return nil, err
	}
	stat, err := tmpFile.Stat()
	if err != nil {
		return nil, err
	}
	if stat.Size() == 0 {
		return nil, common.NewServiceError(http.StatusBadRequest, response.Fail(response.CodeBadRequest).WithMessage("下载文件不能为空"))
	}

	// 解析出文件名称
	fileName := path.Base(objectUrl.Path)
	if fileName == "." || fileName == "/" {
		// 解析不到路径名称的时候，则直接对整个地址进行编码作为文件名称
		objectUrl.RawQuery = ""
		objectUrl.Fragment = ""
		fileName = url.QueryEscape(objectUrl.String())
	}

	err = db.TransactionWithOutResult(g.Request.Context(), func(ctx context.Context) error {
		return service.DefaultResourceService.Upload(ctx, memberId, parentId, service.NewLocalFileResource(stat.Size(), fileName, tmpFile))
	})
	if err != nil {
		return nil, err
	}
	return response.Ok(nil), nil
}

// Search 搜索文件
func (r ResourceApi) Search(g *gin.Context) (any, error) {

	var request = &api.ResourceSearchRequest{}
	request.MemberId = g.GetInt64(constant.CtxKeySubject)
	request.Keywords = strings.TrimSpace(g.Query("keywords"))
	request.Pager = page.NewPagerFromQuery(g.Request.URL.Query())

	if request.Keywords == "" {
		return nil, common.NewServiceError(http.StatusBadRequest, response.Fail(response.CodeBadRequest).WithMessage("关键字不能为空"))
	}

	ret, err := db.Transaction(g.Request.Context(), func(ctx context.Context) (*page.Pagination[*api.ResourceSearchResponse], error) {
		return service.DefaultResourceService.Search(ctx, request)
	})
	if err != nil {
		return nil, err
	}
	return response.Ok(ret), nil
}

// Recent 最近文件
func (r ResourceApi) Recent(g *gin.Context) (any, error) {

	var request = &api.ResourceRecentRequest{}
	request.Pager = page.NewPagerFromQuery(g.Request.URL.Query())
	request.MemberId = g.GetInt64(constant.CtxKeySubject)
	request.ContentType = g.Query("contentType")

	ret, err := db.Transaction(g.Request.Context(), func(ctx context.Context) (any, error) {
		return service.DefaultResourceService.Recent(ctx, request)
	})
	if err != nil {
		return nil, err
	}
	return response.Ok(ret), nil
}

// Group 按照类型分组
func (r ResourceApi) Group(g *gin.Context) (any, error) {
	var request = &api.ResourceGroupRequest{}

	request.Pager = page.NewPagerFromQuery(g.Request.URL.Query())
	request.MemberId = g.GetInt64(constant.CtxKeySubject)
	request.ContentType = g.Query("contentType")
	request.Group = g.Query("group")

	ret, err := db.Transaction(g.Request.Context(), func(ctx context.Context) (any, error) {
		return service.DefaultResourceService.Group(ctx, request)
	})
	if err != nil {
		return nil, err
	}
	return response.Ok(ret), nil
}

// Share 资源分享
func (r ResourceApi) Share(g *gin.Context) (any, error) {
	request := &api.ResourceShareRequest{
		MemberId: g.GetInt64(constant.CtxKeySubject),
	}
	if err := g.ShouldBindJSON(request); err != nil {
		return nil, err
	}
	result, err := db.Transaction(g.Request.Context(), func(ctx context.Context) (any, error) {
		return service.DefaultResourceService.Share(ctx, request)
	})
	if err != nil {
		return nil, err
	}
	return response.Ok(result), nil
}

// Stat 资源统计
// TODO 待优化，单独统计表，空间换时间
func (r ResourceApi) Stat(g *gin.Context) (any, error) {
	memberId := g.GetInt64(constant.CtxKeySubject)
	ret, err := db.Transaction(g.Request.Context(), func(ctx context.Context) (*api.MemberResourceStatResponse, error) {
		return service.DefaultResourceService.TotalSize(ctx, memberId)
	}, db.TxReadOnly)
	if err != nil {
		return nil, err
	}
	return response.Ok(ret), nil
}

var DefaultResourceApi = sync.OnceValue(func() *ResourceApi {
	return NewResourceApi()
})
