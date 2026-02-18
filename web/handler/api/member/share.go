package member

import (
	"context"
	"ispace/common"
	"ispace/common/constant"
	"ispace/common/page"
	"ispace/common/response"
	"ispace/common/types"
	"ispace/db"
	"ispace/repo/model"
	"ispace/web/handler/api"
	"ispace/web/service"
	"net/http"
	"strconv"

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
	if request.ParentId == 0 {
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

func NewShareApi(service *service.ShareService) *ShareApi {
	return &ShareApi{service: service}
}

var DefaultShareApi = NewShareApi(service.DefaultShareService)
