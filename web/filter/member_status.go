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

type MemberStatusCheckFilter struct {
	optional bool
	service  *service.MemberService
}

func NewMemberStatusCheckFilter(optional bool, service *service.MemberService) *MemberStatusCheckFilter {
	return &MemberStatusCheckFilter{optional, service}
}

func (m *MemberStatusCheckFilter) Serve(g *gin.Context) (any, error) {
	memberId := g.GetInt64(constant.CtxKeySubject)
	if memberId == 0 {
		if m.optional {
			return nil, nil
		}
		return nil, common.NewServiceError(http.StatusUnauthorized, response.Fail(response.CodeUnauthorized).WithCode("Authorization Required"))
	}

	// 检索会员状态
	member, err := db.Transaction(g.Request.Context(), func(ctx context.Context) (*model.Member, error) {
		return m.service.GetMemberById(ctx, memberId, "id", "enabled")
	})
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if member.Id == 0 {
		return nil, common.NewServiceError(http.StatusBadRequest, response.Fail(response.CodeBadRequest).WithCode("用户信息错误"))
	}

	if !member.Enabled {
		return nil, common.NewServiceError(http.StatusForbidden, response.Fail(response.CodeForbidden).WithCode("账户被禁用"))
	}
	return nil, nil
}
