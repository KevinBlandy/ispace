package api

import (
	"context"
	"ispace/common/constant"
	"ispace/common/response"
	"ispace/db"
	"ispace/repo/model"
	"ispace/web"
	"ispace/web/service"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type SignInApi struct{}

func NewSignInApi() *SignInApi {
	return &SignInApi{}
}

func (s SignInApi) Serve(ctx *gin.Context) (any, error) {

	var request = &web.SignInApiRequest{}
	if err := ctx.ShouldBindBodyWithJSON(request); err != nil {
		return nil, err
	}

	member, err := db.Transaction(ctx.Request.Context(), func(ctx context.Context) (*model.Member, error) {
		return service.DefaultMemberService.Login(ctx, request)
	}, db.TxReadOnly)

	if err != nil {
		return nil, err
	}

	signed, err := service.DefaultMemberSessionService().Issue(ctx.Request.Context(), member.Id)
	if err != nil {
		return nil, err
	}

	ctx.SetCookieData(&http.Cookie{
		Name:     constant.HttpCookieMemberToken,
		Value:    signed,
		Path:     "/",
		MaxAge:   int((time.Hour * 24 * 365).Seconds()),
		Secure:   false,
		HttpOnly: true,
		SameSite: http.SameSiteDefaultMode,
	})
	return response.Ok(nil), nil
}
