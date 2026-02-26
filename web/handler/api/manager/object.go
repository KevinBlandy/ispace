package manager

import (
	"context"
	"errors"
	"ispace/common"
	"ispace/common/page"
	"ispace/common/response"
	"ispace/db"
	"ispace/repo/model"
	"ispace/store"
	"ispace/web/handler/api"
	"ispace/web/service"
	"net/http"
	"path"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ObjectApi struct {
	objectService *service.ObjectService
}

func NewObjectApi(objectService *service.ObjectService) *ObjectApi {
	return &ObjectApi{objectService: objectService}
}

// List 查询资源列表
func (o *ObjectApi) List(c *gin.Context) (any, error) {
	var request = new(api.ObjectListRequest)
	request.Status = c.Query("status")
	request.Pager = page.NewPagerFromQuery(c.Request.URL.Query())

	// 排序
	request.Pager.Sort = []page.Sort{{Field: "create_time", Order: "DESC"}}

	result, err := db.Transaction(c.Request.Context(), func(ctx context.Context) (*page.Pagination[*api.ObjectListResponse], error) {
		return o.objectService.List(ctx, request)
	}, db.TxReadOnly)
	if err != nil {
		return nil, err
	}
	return response.Ok(result), nil
}

// Update 更新资源信息
func (o *ObjectApi) Update(g *gin.Context) (any, error) {
	var request = new(api.ObjectUpdateRequest)
	if err := g.ShouldBindJSON(&request); err != nil {
		return nil, err
	}
	request.Id, _ = strconv.ParseInt(g.Param("id"), 10, 64)
	if request.Id < 1 {
		return nil, common.NewServiceError(http.StatusBadRequest, response.Fail(response.CodeBadRequest).WithMessage("非法请求"))
	}
	err := db.TransactionWithOutResult(g.Request.Context(), func(ctx context.Context) error {
		return o.objectService.Update(ctx, request)
	}, db.TxReadOnly)
	if err != nil {
		return nil, err
	}
	return response.Ok(nil), nil
}

func (o *ObjectApi) Delete(g *gin.Context) (any, error) {
	var request = new(api.ObjectDeleteRequest)
	if err := g.ShouldBindJSON(&request); err != nil {
		return nil, err
	}
	err := db.TransactionWithOutResult(g.Request.Context(), func(ctx context.Context) error {
		return o.objectService.Delete(ctx, request)
	}, db.TxReadOnly)
	if err != nil {
		return nil, err
	}
	return response.Ok(nil), nil
}

func (o *ObjectApi) Content(g *gin.Context) (any, error) {
	objectId, _ := strconv.ParseInt(g.Param("id"), 10, 64)
	if objectId < 1 {
		return nil, common.NewServiceError(http.StatusBadRequest, response.Fail(response.CodeBadRequest).WithCode("非法请求"))
	}

	ret, err := db.Transaction(g.Request.Context(), func(ctx context.Context) (*model.Object, error) {
		return o.objectService.GetById(ctx, objectId, "id", "compression", "content_type", "path")
	}, db.TxReadOnly)

	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if ret.Id == 0 {
		return nil, common.NewServiceError(http.StatusNotFound, response.Fail(response.CodeNotFound).WithCode("对象不存在"))
	}

	err = store.DefaultStore().ServeContent(g.Writer, g.Request, &store.File{
		Title:       path.Base(ret.Path),
		Compression: ret.Compression,
		ContentType: ret.ContentType,
		Path:        ret.Path,
	})
	if err != nil {
		return nil, err
	}
	g.Abort()
	return nil, nil
}

var DefaultObjectApi = NewObjectApi(service.DefaultObjectService)
