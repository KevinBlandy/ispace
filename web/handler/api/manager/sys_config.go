package manager

import (
	"context"
	"ispace/common"
	"ispace/common/response"
	"ispace/db"
	"ispace/repo/model"
	"ispace/web/handler/api"
	"ispace/web/service"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type SysConfigApi struct {
	service *service.SysConfigService
}

// List 查询
func (s *SysConfigApi) List(c *gin.Context) (any, error) {
	ret, err := db.Transaction(c.Request.Context(), func(ctx context.Context) ([]*api.SysConfigListResponse, error) {
		return s.service.List(ctx)
	}, db.TxReadOnly)
	if err != nil {
		return nil, err
	}
	return response.Ok(ret), nil
}

func (s *SysConfigApi) Create(g *gin.Context) (any, error) {
	var request = new(api.SysConfigCreateRequest)
	if err := g.ShouldBindJSON(request); err != nil {
		return nil, err
	}

	err := db.TransactionWithOutResult(g.Request.Context(), func(ctx context.Context) error {
		return s.service.Create(ctx, request)
	})
	if err != nil {
		return nil, err
	}

	g.AbortWithStatusJSON(http.StatusCreated, response.Ok(nil))
	return nil, nil
}

func (s *SysConfigApi) Update(g *gin.Context) (any, error) {
	var request = new(api.SysConfigUpdateRequest)
	if err := g.ShouldBindJSON(request); err != nil {
		return nil, err
	}
	request.Id, _ = strconv.ParseInt(g.Param("id"), 10, 64)
	if request.Id < 1 {
		return nil, common.NewServiceError(http.StatusBadRequest, response.Fail(response.CodeBadRequest).WithMessage("非法请求"))
	}

	before, err := db.Transaction(g.Request.Context(), func(ctx context.Context) (*model.SysConfig, error) {
		return s.service.Update(ctx, request)
	})
	if err != nil {
		return nil, err
	}

	// 更新成功后，移除缓存，包括旧的和新的
	s.service.Remove(g.Request.Context(), before.Key)
	if before.Key != request.Key {
		s.service.Remove(g.Request.Context(), request.Key)
	}
	return response.Ok(nil), nil
}

// Delete 删除
func (s *SysConfigApi) Delete(g *gin.Context) (any, error) {
	var request = new(api.SysConfigDeleteRequest)
	if err := g.ShouldBindJSON(request); err != nil {
		return nil, err
	}
	ret, err := db.Transaction(g.Request.Context(), func(ctx context.Context) ([]*model.SysConfig, error) {
		return s.service.Delete(ctx, request)
	})
	if err != nil {
		return nil, err
	}
	for _, v := range ret {
		s.service.Remove(g.Request.Context(), v.Key)
	}
	return response.Ok(nil), nil
}

func NewSysConfigApi(service *service.SysConfigService) *SysConfigApi {
	return &SysConfigApi{}
}

var DefaultSysConfigApi = NewSysConfigApi(service.DefaultSysConfigService)
