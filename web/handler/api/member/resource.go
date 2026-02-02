package member

import (
	"archive/zip"
	"compress/gzip"
	"context"
	"errors"
	"io"
	"ispace/common"
	"ispace/common/constant"
	"ispace/common/response"
	"ispace/common/util"
	"ispace/db"
	"ispace/repo/model"
	"ispace/store"
	"ispace/web/handler/api"
	"ispace/web/service"
	"log/slog"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
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
		Keywords: ctx.Query("keywords"),
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

// FlashUpload ⚡️秒传
func (r ResourceApi) FlashUpload(g *gin.Context) (any, error) {
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

	// 读文件
	objectFile, err := store.DefaultStore().Open(resource.Path)
	if err != nil {
		return nil, err
	}
	defer util.SafeClose(objectFile)

	// 文件本身是否经过了压缩存储
	switch resource.Compression {
	case model.ObjectCompressionNone:
	case model.ObjectCompressionGzip:
		// 创建临时文件
		tmpFile, err := os.CreateTemp("", strconv.FormatInt(resourceId, 10))
		if err != nil {
			return nil, err
		}
		defer func() {
			if err := os.Remove(tmpFile.Name()); err != nil {
				slog.ErrorContext(g.Request.Context(), "临时文件删除异常", slog.String("err", err.Error()))
			}
		}()
		defer util.SafeClose(tmpFile)
		gzipReader, err := gzip.NewReader(objectFile)
		if err != nil {
			return nil, err
		}
		defer util.SafeClose(gzipReader)
		if _, err := io.Copy(tmpFile, gzipReader); err != nil {
			return nil, err
		}
		objectFile = tmpFile
	default:
		return nil, common.NewServiceError(http.StatusBadRequest, response.Fail(response.CodeBadRequest).WithMessage("未实现的压缩格式"))
	}

	stat, err := objectFile.Stat()
	if err != nil {
		return nil, err
	}
	zipReader, err := zip.NewReader(objectFile, stat.Size())
	if err != nil {
		return nil, err
	}

	file := strings.TrimSpace(g.Query("file"))

	// 返回树结构
	if file == "" {
		return response.Ok(r.zipTree(zipReader)), nil
	}

	// 查询某个文件
	for _, f := range zipReader.File {
		if f.Name == file && !f.FileInfo().IsDir() {
			err := func(f *zip.File) error {
				contentType := mime.TypeByExtension(filepath.Ext(f.FileInfo().Name()))
				if contentType == "" {
					contentType = "application/octet-stream"
				}

				g.Header("Content-Type", contentType)
				g.Header("Content-Length", strconv.FormatUint(f.UncompressedSize64, 10))
				download := util.BoolQuery(g.GetQuery("download"))
				if download != nil && *download {
					g.Header("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": f.FileInfo().Name()}))
				}
				fileReader, err := f.Open()
				if err != nil {
					return err
				}
				defer util.SafeClose(fileReader)

				_, _ = io.Copy(g.Writer, fileReader)

				//http.ServeContent(g.Writer, g.Request, filepath.Base(f.Name), f.Modified, fileReader)
				g.Abort()
				return nil
			}(f)

			if err != nil {
				return nil, err
			}
			return nil, nil
		}
	}
	return response.Fail(response.CodeBadRequest).WithMessage("文件 " + file + " 不存在"), nil
}

// zipTree 读取 zip 文件并将其转换为树形结构
// gen by Gemini
func (r ResourceApi) zipTree(z *zip.Reader) []*api.ResourceUnarchiveResponse {

	var rootEntries []*api.ResourceUnarchiveResponse
	// 用于存放已创建的目录节点，Key 为完整路径 File
	nodesMap := make(map[string]*api.ResourceUnarchiveResponse)

	for _, f := range z.File {
		// 1. 标准化路径并去除末尾斜杠
		fullPath := strings.Trim(filepath.ToSlash(f.Name), "/")
		if fullPath == "" {
			continue
		}

		parts := strings.Split(fullPath, "/")
		var currentPath string

		// 2. 逐层处理路径
		for i, part := range parts {
			parentPath := currentPath
			if currentPath == "" {
				currentPath = part
			} else {
				currentPath = currentPath + "/" + part
			}

			// 判断是否为该条目自身的终点
			isLastPart := i == len(parts)-1

			// 如果该节点已存在，直接跳过进入下一层
			if _, exists := nodesMap[currentPath]; exists {
				continue
			}

			// 3. 创建新节点
			newNode := &api.ResourceUnarchiveResponse{
				File:    currentPath,
				Title:   part,
				Dir:     !isLastPart || f.FileInfo().IsDir(),
				Entries: []*api.ResourceUnarchiveResponse{},
			}

			// 只有是文件且是路径终点时，才记录大小
			if isLastPart && !f.FileInfo().IsDir() {
				newNode.Size = f.UncompressedSize64
			}

			// 4. 挂载节点
			if parentPath == "" {
				// 顶层目录
				rootEntries = append(rootEntries, newNode)
			} else {
				// 挂载到父节点
				if parentNode, ok := nodesMap[parentPath]; ok {
					parentNode.Entries = append(parentNode.Entries, newNode)
					parentNode.Dir = true // 确保父节点被标记为目录
				}
			}

			// 5. 缓存目录节点（文件节点理论上不需要缓存，但为了逻辑统一也可以放进去）
			nodesMap[currentPath] = newNode
		}
	}
	return rootEntries
}

var DefaultResourceApi = sync.OnceValue(func() *ResourceApi {
	return NewResourceApi()
})
