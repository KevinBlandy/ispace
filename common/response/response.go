package response

type Response struct {
	Code    Code   `json:"code"`    // 业务状态码
	Success bool   `json:"success"` // 是否成功
	Message string `json:"message"` // 提示消息
	Data    any    `json:"data"`    // 数据
}

func (r *Response) WithCode(code Code) *Response {
	r.Code = code
	return r
}
func (r *Response) WithSuccess() *Response {
	r.Success = true
	return r
}
func (r *Response) WithMessage(message string) *Response {
	r.Message = message
	return r
}
func (r *Response) WithData(data any) *Response {
	r.Data = data
	return r
}

func New(code Code, success bool, message string, data any) *Response {
	return &Response{
		Code:    code,
		Success: success,
		Message: message,
		Data:    data,
	}
}

func Ok(data any) *Response {
	return &Response{
		Code:    CodeOk,
		Data:    data,
		Message: string(CodeOk),
		Success: true,
	}
}

func Fail(code Code) *Response {
	return &Response{
		Code:    code,
		Message: string(code),
		Success: false,
	}
}

// Code 业务状态码
type Code string

var (
	// CodeOk 状态码
	CodeOk Code = "OK"
	// CodeBadRequest 客户端异常
	CodeBadRequest Code = "BAD_REQUEST"
	// CodeNotFound 没找到
	CodeNotFound Code = "NOT_FOUND"
	// CodeCaptchaFailed 验证码错误
	CodeCaptchaFailed Code = "CAPTCHA_FAILED"
	// CodeSharePasswordFailed 口令错误
	CodeSharePasswordFailed Code = "SHARE_PASSWORD_FAILED"
	// CodeUnauthorized 未登录
	CodeUnauthorized Code = "UNAUTHORIZED"
	// CodeForbidden 无权操作
	CodeForbidden Code = "FORBIDDEN"
	// CodeServerError 服务器异常
	CodeServerError Code = "SERVER_ERROR"
)
