package member

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

type SignInApi struct {
	memberService *service.MemberService
}

func NewSignInApi(memberService *service.MemberService) *SignInApi {
	return &SignInApi{memberService: memberService}
}

func (s *SignInApi) Serve(ctx *gin.Context) (any, error) {

	var request = &api.MemberSignInRequest{}
	if err := ctx.ShouldBindBodyWithJSON(request); err != nil {
		return nil, err
	}

	member, err := db.Transaction(ctx.Request.Context(), func(ctx context.Context) (*model.Member, error) {
		return s.memberService.Login(ctx, request)
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
		HttpOnly: true,
		SameSite: http.SameSiteDefaultMode,
	})
	return response.Ok(nil), nil
}

var DefaultSignInApi = NewSignInApi(service.DefaultMemberService)
