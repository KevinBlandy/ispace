package filter

import (
	"ispace/common"
	"ispace/common/constant"
	"ispace/common/response"
	"ispace/web/service"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type AuthFilter struct {
	header, cookie string
	sessionService *service.SessionService
}

func NewAuthFilter(cookie, header string, ss *service.SessionService) *AuthFilter {
	return &AuthFilter{
		header:         header,
		cookie:         cookie,
		sessionService: ss,
	}
}

func (a *AuthFilter) Serve(c *gin.Context) (any, error) {

	token, _ := c.Cookie(a.cookie)
	if token == "" {
		token = c.GetHeader(a.header)
	}
	if token == "" {
		return nil, common.NewServiceError(http.StatusUnauthorized, response.Fail(response.CodeUnauthorized).WithCode("Token Required"))
	}

	session, err := a.sessionService.Parse(c.Request.Context(), token)
	if err != nil {
		return nil, err
	}

	// session 快到期，续约
	if session.ExpireAt.Sub(time.Now()) < time.Hour {
		go func() {
			if _, err := a.sessionService.Renewal(c.Request.Context(), session); err != nil {
				slog.ErrorContext(c.Request.Context(), "session renewal error", slog.String("err", err.Error()))
			}
		}()
	}

	c.Set(constant.CtxKeySubject, session.Subject)
	c.Set(constant.CtxKeySession, session)
	return nil, nil
}

var DefaultMemberAuthFilter = sync.OnceValue(func() *AuthFilter {
	return NewAuthFilter(
		constant.HttpCookieMemberToken,
		constant.HttpHeaderMemberToken,
		service.DefaultMemberSessionService(),
	)
})

var DefaultManagerAuthFilter = sync.OnceValue(func() *AuthFilter {
	return NewAuthFilter(
		constant.HttpCookieManagerToken,
		constant.HttpHeaderManagerToken,
		service.DefaultManagerSessionService(),
	)
})
