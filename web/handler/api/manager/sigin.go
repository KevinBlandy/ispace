package manager

import (
	"context"
	"ispace/common/constant"
	"ispace/common/response"
	"ispace/db"
	"ispace/repo/model"
	"ispace/web/handler/api"
	"ispace/web/service"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// 管理后台登录

type SignInApi struct {
	adminService *service.AdminService
}

func (a *SignInApi) SignIn(ctx *gin.Context) (any, error) {
	var request = api.ManagerSignInRequest{}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		return nil, err
	}

	admin, err := db.Transaction(ctx.Request.Context(), func(ctx context.Context) (*model.Admin, error) {
		return a.adminService.Login(ctx, &request)
	})
	if err != nil {
		return nil, err
	}
	signed, err := service.DefaultManagerSessionService().Issue(ctx.Request.Context(), admin.Id)
	if err != nil {
		return nil, err
	}
	ctx.SetCookieData(&http.Cookie{
		Name:     constant.HttpCookieManagerToken,
		Value:    signed,
		Path:     "/",
		MaxAge:   int((time.Hour * 24 * 365).Seconds()),
		Secure:   false,
		HttpOnly: true,
		SameSite: http.SameSiteDefaultMode,
	})

	return response.Ok(nil), nil
}

func NewSignInApi(adminService *service.AdminService) *SignInApi {
	return &SignInApi{adminService: adminService}
}

var DefaultSignInApi = NewSignInApi(service.DefaultAdminService)
