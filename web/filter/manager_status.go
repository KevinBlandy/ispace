package filter

import (
	"context"
	"errors"
	"ispace/common"
	"ispace/common/constant"
	"ispace/common/response"
	"ispace/db"
	"ispace/repo/model"
	"ispace/web/service"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ManagerStatusCheckFilter struct {
	optional bool
	service  *service.AdminService
}

func NewManagerStatusCheckFilter(optional bool, service *service.AdminService) *ManagerStatusCheckFilter {
	return &ManagerStatusCheckFilter{optional, service}
}

func (m *ManagerStatusCheckFilter) Serve(g *gin.Context) (any, error) {

	adminId := g.GetInt64(constant.CtxKeySubject)

	if adminId == 0 {
		if m.optional {
			return nil, nil
		}
		return nil, common.NewServiceError(http.StatusUnauthorized, response.Fail(response.CodeUnauthorized).WithMessage("Authorization Required"))
	}

	// 检索会员状态
	member, err := db.Transaction(g.Request.Context(), func(ctx context.Context) (*model.Admin, error) {
		return m.service.GetAdminById(ctx, adminId, "id", "enabled")
	})
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if member.Id == 0 {
		return nil, common.NewServiceError(http.StatusBadRequest, response.Fail(response.CodeBadRequest).WithMessage("用户信息错误"))
	}

	if !member.Enabled {
		return nil, common.NewServiceError(http.StatusForbidden, response.Fail(response.CodeForbidden).WithMessage("账户被禁用"))
	}
	return nil, nil
}
