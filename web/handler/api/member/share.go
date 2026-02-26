package member

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"ispace/common"
	"ispace/common/constant"
	"ispace/common/page"
	"ispace/common/response"
	"ispace/common/types"
	"ispace/db"
	"ispace/repo/model"
	"ispace/store"
	"ispace/web/handler/api"
	"ispace/web/service"
	"maps"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type ShareApi struct {
	service *service.ShareService
}

// List 我的分享列表
func (a ShareApi) List(g *gin.Context) (any, error) {
	var request = new(api.ShareListRequest)
	request.MemberId = g.GetInt64(constant.CtxKeySubject)
	request.Pager = page.NewPagerFromQuery(g.Request.URL.Query())
	request.Title = g.Query("title")

	result, err := db.Transaction(g.Request.Context(), func(ctx context.Context) (*page.Pagination[*api.ShareListResponse], error) {
		return a.service.List(ctx, request)
	})
	if err != nil {
		return nil, err
	}
	return response.Ok(result), nil
}

// Update 更新资源
func (a ShareApi) Update(g *gin.Context) (any, error) {
	var request = new(api.ShareUpdateRequest)
	if err := g.ShouldBindJSON(&request); err != nil {
		return nil, err
	}
	request.MemberId = g.GetInt64(constant.CtxKeySubject)
	request.Id, _ = strconv.ParseInt(g.Param("id"), 10, 64)
	if request.Id < 1 {
		return nil, common.NewServiceError(http.StatusBadRequest, response.Fail(response.CodeBadRequest).WithMessage("非法请求"))
	}

	err := db.TransactionWithOutResult(g.Request.Context(), func(ctx context.Context) error {
		return a.service.Update(ctx, request)
	})
	if err != nil {
		return nil, err
	}
	return response.Ok(nil), nil
}

// Delete 取消分享
func (a ShareApi) Delete(g *gin.Context) (any, error) {
	var request = new(api.ShareDeleteRequest)
	if err := g.ShouldBindJSON(&request); err != nil {
		return nil, err
	}
	request.MemberId = g.GetInt64(constant.CtxKeySubject)

	err := db.TransactionWithOutResult(g.Request.Context(), func(ctx context.Context) error {
		return a.service.Delete(ctx, request)
	})
	if err != nil {
		return nil, err
	}
	return response.Ok(nil), nil
}

// ResourceList 资源列表
func (a ShareApi) ResourceList(g *gin.Context) (any, error) {

	var request = new(api.ShareResourceListRequest)

	request.Identifier = types.Identifier(g.Param("path"))
	if request.Identifier == "" {
		return nil, common.NewServiceError(http.StatusBadRequest, response.Fail(response.CodeBadRequest).WithMessage("非法请求"))
	}
	request.Title = g.Query("title")
	request.ParentId, _ = strconv.ParseInt(g.Query("parentId"), 10, 64)
	if request.ParentId < 1 {
		request.ParentId = model.DefaultResourceParentId
	}

	result, err := db.Transaction(g.Request.Context(), func(ctx context.Context) ([]*api.ShareResourceListResponse, error) {
		return a.service.ResourceList(ctx, request)
	}, db.TxReadOnly)
	if err != nil {
		return nil, err
	}
	return response.Ok(result), nil
}

// Share 分享详情
func (a ShareApi) Share(g *gin.Context) (any, error) {
	identifier := types.Identifier(g.Param("path"))
	if identifier == "" {
		return nil, common.NewServiceError(http.StatusBadRequest, response.Fail(response.CodeBadRequest).WithMessage("非法请求"))
	}
	ret, err := db.Transaction(g.Request.Context(), func(ctx context.Context) (*api.ShareResponse, error) {
		return a.service.Share(ctx, identifier)
	}, db.TxReadOnly)
	if err != nil {
		return nil, err
	}
	return response.Ok(ret), nil
}

// Verify 分享口令验证
func (a ShareApi) Verify(g *gin.Context) (any, error) {
	var request = new(api.SharePasswordVerifyRequest)
	if err := g.ShouldBindJSON(&request); err != nil {
		return nil, err
	}
	request.Identifier = types.Identifier(g.Param("path"))

	share, err := db.Transaction(g.Request.Context(), func(ctx context.Context) (*model.Share, error) {
		return a.service.GetByIdentifier(ctx, request.Identifier, "id", "password", "path")
	}, db.TxReadOnly)
	if err != nil {
		return nil, err
	}
	if share.Password == "" {
		return nil, common.NewServiceError(http.StatusBadRequest, response.Fail(response.CodeBadRequest).WithMessage("该资源无需密码"))
	}

	if share.Password != request.Password {
		return nil, common.NewServiceError(http.StatusBadRequest, response.Fail(response.CodeBadRequest).WithMessage("密码错误"))
	}

	// Token 有效期 24H
	expireTime := strconv.FormatInt(time.Now().Add(time.Hour*24).UnixMilli(), 10)

	// 签发密钥
	//  sign = shah256(path, timestamp, password)
	hasher := sha256.New()
	hasher.Write([]byte(share.Path))
	hasher.Write([]byte(expireTime))
	hasher.Write([]byte(share.Password))

	sign := hex.EncodeToString(hasher.Sum(nil))

	g.SetCookieData(&http.Cookie{
		Name:     constant.HttpCookieShareToken,
		Value:    strings.Join([]string{sign, expireTime}, "-"), //  "sign-timestamp"
		Path:     "/",
		MaxAge:   60 * 60 * 24 * 30, // Cookie 存放一个月
		HttpOnly: true,
		SameSite: http.SameSiteDefaultMode,
	})

	//  累计访问次数
	go func() {
		_ = db.TransactionWithOutResult(context.Background(), func(ctx context.Context) error {
			return a.service.IncrViews(ctx, share.Id, 1)
		})
	}()

	return response.Ok(nil), nil
}

// Content 读取文件内容
func (a ShareApi) Content(g *gin.Context) (any, error) {

	identifier := types.Identifier(g.Param("path"))
	resourceId, _ := strconv.ParseInt(g.Param("resourceId"), 10, 64)
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
		return a.service.GetResource(ctx, identifier, resourceId)
	}, db.TxReadOnly)

	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	if resource.Id < 1 {
		return nil, common.NewServiceError(http.StatusNotFound, response.Fail(response.CodeNotFound).WithMessage("资源不存在"))
	}

	if resource.Status == model.ObjectStatusDisabled {
		return nil, common.NewServiceError(http.StatusForbidden, response.Fail(response.CodeForbidden).WithMessage("资源被屏蔽"))
	}

	err = store.DefaultStore().ServeContent(g.Writer, g.Request, &store.File{
		Title:       resource.Title,
		Compression: resource.Compression,
		ContentType: resource.ContentType,
		Path:        resource.Path,
	})
	if err != nil {
		return nil, err
	}

	g.Abort()

	return nil, nil
}

// Download 文件下载
func (a ShareApi) Download(g *gin.Context) (any, error) {

	// 要下载的资源 ID
	var idMap = make(map[int64]struct{})
	for _, v := range g.QueryArray("resourceId") {
		resourceId, err := strconv.ParseInt(v, 10, 64)
		if err != nil || resourceId < 1 {
			return nil, common.NewServiceError(http.StatusBadRequest, response.Fail(response.CodeBadRequest).WithMessage("非法请求"))
		}
		idMap[resourceId] = struct{}{}
	}

	// 参数封装
	var request = new(api.ShareResourceDownloadRequest)
	request.Id = slices.Collect(maps.Keys(idMap))
	request.Identifier = types.Identifier(g.Param("path"))

	if len(request.Id) == 0 {
		return nil, common.NewServiceError(http.StatusBadRequest, response.Fail(response.CodeBadRequest).WithMessage("下载资源不能为空"))
	}

	// 检索资源树
	tree, err := db.Transaction(g.Request.Context(), func(ctx context.Context) ([]*store.DownloadTree, error) {
		return a.service.Download(ctx, request)
	}, db.TxReadOnly)

	if err != nil {
		return nil, err
	}
	if err := store.DefaultStore().Downloads(g.Writer, tree...); err != nil {
		return nil, err
	}
	g.Abort()
	return nil, nil
}

// Unarchive 解压文件
func (a ShareApi) Unarchive(g *gin.Context) (any, error) {

	// 请求参数
	identifier := types.Identifier(g.Param("path"))
	resourceId, _ := strconv.ParseInt(g.Param("resourceId"), 10, 64)
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
		return a.service.GetResource(ctx, identifier, resourceId)
	}, db.TxReadOnly)

	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	if resource.Id < 1 {
		return nil, common.NewServiceError(http.StatusNotFound, response.Fail(response.CodeNotFound).WithMessage("资源不存在"))
	}

	if resource.Status == model.ObjectStatusDisabled {
		return nil, common.NewServiceError(http.StatusForbidden, response.Fail(response.CodeForbidden).WithMessage("资源被屏蔽"))
	}

	// 文件类型判断
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

func NewShareApi(service *service.ShareService) *ShareApi {
	return &ShareApi{service: service}
}

var DefaultShareApi = NewShareApi(service.DefaultShareService)
