package member

import (
	"context"
	"ispace/common"
	"ispace/common/constant"
	"ispace/common/page"
	"ispace/common/response"
	"ispace/db"
	"ispace/web/handler/api"
	"ispace/web/service"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type RecycleBinApi struct {
	service *service.RecycleBinService
}

// List 回收站项目列表
func (r RecycleBinApi) List(c *gin.Context) (any, error) {
	var request = new(api.RecycleBinListRequest)
	request.Pager = page.NewPagerFromQuery(c.Request.URL.Query())
	request.MemberId = c.GetInt64(constant.CtxKeySubject)
	request.Title = c.Query("title")

	result, err := db.Transaction(c.Request.Context(), func(ctx context.Context) (*page.Pagination[*api.RecycleBinListResponse], error) {
		return r.service.List(ctx, request)
	}, db.TxReadOnly)
	if err != nil {
		return nil, err
	}
	return response.Ok(result), nil
}

// Delete 删除项目
func (r RecycleBinApi) Delete(g *gin.Context) (any, error) {
	var request = &api.RecycleBinDeleteRequest{}
	if err := g.ShouldBindJSON(request); err != nil {
		return nil, err
	}
	request.MemberId = g.GetInt64(constant.CtxKeySubject)
	err := db.TransactionWithOutResult(g.Request.Context(), func(ctx context.Context) error {
		return r.service.Delete(ctx, request)
	})
	if err != nil {
		return nil, err
	}
	return response.Ok(nil), nil
}

// Restore 恢复文件
func (r RecycleBinApi) Restore(g *gin.Context) (any, error) {
	var request = new(api.RecycleBinRestoreRequest)
	if err := g.ShouldBindJSON(request); err != nil {
		return nil, err
	}

	request.MemberId = g.GetInt64(constant.CtxKeySubject)

	err := db.TransactionWithOutResult(g.Request.Context(), func(ctx context.Context) error {
		return r.service.Restore(ctx, request)
	})
	if err != nil {
		return nil, err
	}
	return response.Ok(nil), nil
}

// Entries 子项目
func (r RecycleBinApi) Entries(g *gin.Context) (any, error) {
	var memberId = g.GetInt64(constant.CtxKeySubject)
	rId, err := strconv.ParseInt(g.Param("id"), 10, 64)
	if err != nil {
		return nil, common.NewServiceError(http.StatusBadRequest, response.Fail(response.CodeBadRequest).WithMessage("非法请求"))
	}
	result, err := db.Transaction(g.Request.Context(), func(ctx context.Context) ([]*api.RecycleBinEntryResponse, error) {
		return r.service.Entries(ctx, memberId, rId)
	})
	if err != nil {
		return nil, err
	}
	return response.Ok(result), nil
}

func NewRecycleBinApi(binService *service.RecycleBinService) *RecycleBinApi {
	return &RecycleBinApi{service: binService}
}

var DefaultRecycleBinApi = NewRecycleBinApi(service.DefaultRecycleBinService)
