package member

import (
	"context"
	"ispace/common/constant"
	"ispace/common/response"
	"ispace/db"
	"ispace/web/handler/api"
	"ispace/web/service"

	"github.com/gin-gonic/gin"
)

type AccountSettingApi struct {
	ms *service.MemberService
}

func NewAccountSettingApi(ms *service.MemberService) *AccountSettingApi {
	return &AccountSettingApi{ms}
}

// UpdatePassword 更新密码
func (a *AccountSettingApi) UpdatePassword(g *gin.Context) (any, error) {
	var request = new(api.PasswordUpdateRequest)
	if err := g.ShouldBindJSON(&request); err != nil {
		return nil, err
	}
	request.MemberId = g.GetInt64(constant.CtxKeySubject)

	err := db.TransactionWithOutResult(g.Request.Context(), func(ctx context.Context) error {
		return a.ms.UpdatePassword(ctx, request)
	})
	if err != nil {
		return nil, err
	}

	// TODO 驱逐其他 Session

	return response.Ok(nil), nil
}

var DefaultAccountSettingApi = NewAccountSettingApi(service.DefaultMemberService)
