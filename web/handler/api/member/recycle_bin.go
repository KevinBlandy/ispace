package member

import (
	"context"
	"database/sql"
	"errors"
	"ispace/common"
	"ispace/common/constant"
	"ispace/common/page"
	"ispace/common/response"
	"ispace/db"
	"ispace/repo/model"
	"ispace/store"
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

// Content 项目内容
func (r RecycleBinApi) Content(g *gin.Context) (any, error) {
	var memberId = g.GetInt64(constant.CtxKeySubject)
	rId, err := strconv.ParseInt(g.Param("id"), 10, 64)
	if err != nil {
		return nil, common.NewServiceError(http.StatusBadRequest, response.Fail(response.CodeBadRequest).WithMessage("非法请求"))
	}
	result, err := db.Transaction(g.Request.Context(), func(ctx context.Context) (struct {
		Id          int64
		Title       string
		Status      model.ObjectStatus
		Path        string
		Compression model.ObjectCompression
	}, error) {
		return r.service.GetObject(ctx, memberId, rId)
	}, db.TxReadOnly)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	if result.Id < 1 {
		return nil, common.NewServiceError(http.StatusNotFound, response.Fail(response.CodeNotFound).WithMessage("资源不存在"))
	}

	// 资源状态判断
	if result.Status == model.ObjectStatusDisabled {
		return nil, common.NewServiceError(http.StatusBadRequest, response.Fail(response.CodeBadRequest).WithMessage("该资源已被屏蔽"))
	}

	if err := store.DefaultStore().ServeContent(g.Writer, g.Request, &store.File{
		Title:       result.Title,
		Compression: result.Compression,
		Path:        result.Path,
	}); err != nil {
		return nil, err
	}
	g.Abort()
	return nil, nil
}

func NewRecycleBinApi(binService *service.RecycleBinService) *RecycleBinApi {
	return &RecycleBinApi{service: binService}
}

var DefaultRecycleBinApi = NewRecycleBinApi(service.DefaultRecycleBinService)
