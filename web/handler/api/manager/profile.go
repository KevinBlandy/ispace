package manager

import (
	"context"
	"ispace/common/constant"
	"ispace/common/response"
	"ispace/db"
	"ispace/web/handler/api"
	"ispace/web/service"

	"github.com/gin-gonic/gin"
)

type ProfileApi struct {
	as *service.AdminService
}

func NewProfileApi(m *service.AdminService) *ProfileApi {
	return &ProfileApi{as: m}
}

// Profile 查询自己的基础信息
func (p *ProfileApi) Profile(g *gin.Context) (any, error) {
	memberId := g.GetInt64(constant.CtxKeySubject)
	ret, err := db.Transaction(g.Request.Context(), func(ctx context.Context) (*api.AdminProfileResponse, error) {
		return p.as.Profile(ctx, memberId)
	}, db.TxReadOnly)
	if err != nil {
		return nil, err
	}
	return response.Ok(ret), nil
}

// Update 更新
func (p *ProfileApi) Update(g *gin.Context) (any, error) {
	var request = new(api.AdminProfileUpdateRequest)
	if err := g.ShouldBindJSON(request); err != nil {
		return nil, err
	}
	request.AdminId = g.GetInt64(constant.CtxKeySubject)

	err := db.TransactionWithOutResult(g.Request.Context(), func(ctx context.Context) error {
		return p.as.UpdateProfile(ctx, request)
	})
	if err != nil {
		return nil, err
	}
	return response.Ok(nil), nil
}

var DefaultProfileApi = NewProfileApi(service.DefaultAdminService)
