package constant

var (
	HttpHeaderRequestId = "X-Request-Id"
	HttpHeaderCaptchaId = "X-Captcha-Id"

	// HttpHeaderMemberToken 会员 Token
	HttpHeaderMemberToken = "X-Member-Token"
	// HttpHeaderManagerToken 管理员 Token
	HttpHeaderManagerToken = "X-Manager-Token"

	// HttpHeaderTimeZone 客户端时区
	HttpHeaderTimeZone = "X-Time-Zone"

	// HttpCookieMemberToken Cookie Token
	HttpCookieMemberToken = "MEMBER_TOKEN"
	// HttpCookieManagerToken Manager Token
	HttpCookieManagerToken = "MANAGER_TOKEN"
	// HttpCookieShareToken 资源分享 Token
	HttpCookieShareToken = "SHARE_TOKEN"
)
