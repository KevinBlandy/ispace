package filter

import (
	"ispace/common"
	"ispace/common/constant"
	"ispace/common/response"
	"ispace/web/service"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type AuthFilter struct {
	header         string
	cookie         string
	sessionService *service.SessionService
	optional       bool
}

func NewAuthFilter(cookie, header string, ss *service.SessionService, optional bool) *AuthFilter {
	return &AuthFilter{
		header:         header,
		cookie:         cookie,
		sessionService: ss,
		optional:       optional,
	}
}

func (a *AuthFilter) Serve(c *gin.Context) (any, error) {

	// 已经登陆了
	if _, ok := c.Get(constant.CtxKeySubject); ok {
		return nil, nil
	}

	token, _ := c.Cookie(a.cookie)
	if token == "" {
		token = c.GetHeader(a.header)
	}
	if token == "" {
		if a.optional { // 非强制的
			return nil, nil
		}
		return nil, common.NewServiceError(http.StatusUnauthorized, response.Fail(response.CodeUnauthorized).WithCode("Token Required"))
	}

	session, err := a.sessionService.Parse(c.Request.Context(), token)
	if err != nil {
		if a.optional {
			return nil, nil
		}
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

func NewManagerAuthFilter(optional bool) *AuthFilter {
	return &AuthFilter{
		constant.HttpCookieManagerToken,
		constant.HttpHeaderManagerToken,
		service.DefaultManagerSessionService(),
		optional,
	}
}

// NewMemberAuthFilter 创建新的会员验证器，允许自定义 “可选” 状态
func NewMemberAuthFilter(optional bool) *AuthFilter {
	return NewAuthFilter(
		constant.HttpCookieMemberToken,
		constant.HttpHeaderMemberToken,
		service.DefaultMemberSessionService(),
		optional,
	)
}
