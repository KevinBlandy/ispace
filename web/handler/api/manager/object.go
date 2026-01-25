package manager

import (
	"context"
	"ispace/common"
	"ispace/common/page"
	"ispace/common/response"
	"ispace/db"
	"ispace/web/handler/api"
	"ispace/web/service"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
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

var DefaultObjectApi = NewObjectApi(service.NewObjectService())
