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

func Ok(data any) *Response {
	return &Response{
		Code:    CodeOk,
		Data:    data,
		Message: "success",
		Success: true,
	}
}

func Fail(code Code) *Response {
	return &Response{
		Code:    code,
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
	// CodeForbidden 无权操作
	CodeForbidden Code = "FORBIDDEN"
	// CodeServerError 服务器异常
	CodeServerError Code = "SERVER_ERROR"
)

// HttpStatus 获取业务状态码对应的 HTTP 状态码
//func (r Code) HttpStatus() int {
//	switch r {
//	case CodeBadRequest:
//		return http.StatusBadRequest
//	case CodeNotFound:
//		return http.StatusNotFound
//	case CodeServerError:
//		return http.StatusInternalServerError
//	default:
//		return http.StatusOK // 默认 OK
//	}
//}
