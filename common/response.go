package common

import "net/http"

type Response[T any] struct {
	Code    ResponseCode `json:"code"`    // 业务状态码
	Success bool         `json:"success"` // 是否成功
	Message string       `json:"message"` // 提示消息
	Data    T            `json:"data"`    // 数据
}

func (r *Response[T]) WithCode(code ResponseCode) *Response[T] {
	r.Code = code
	return r
}
func (r *Response[T]) WithSuccess() *Response[T] {
	r.Success = true
	return r
}
func (r *Response[T]) WithMessage(message string) *Response[T] {
	r.Message = message
	return r
}
func (r *Response[T]) WithData(data T) *Response[T] {
	r.Data = data
	return r
}

func NewOkResponse[T any](data T) *Response[T] {
	return &Response[T]{
		Code:    ResponseCodeOk,
		Data:    data,
		Message: "success",
		Success: true,
	}
}

func NewFailResponse[T any](code ResponseCode, message string) *Response[T] {
	return &Response[T]{
		Code:    code,
		Message: message,
		Success: false,
	}
}

type ResponseCode string

var (
	// ResponseCodeOk 状态码
	ResponseCodeOk ResponseCode = "OK"
	// ResponseCodeBadRequest 客户端异常
	ResponseCodeBadRequest ResponseCode = "BAD_REQUEST"
	// ResponseCodeNotFound 没找到
	ResponseCodeNotFound ResponseCode = "NOT_FOUND"
	// ResponseCodeServerError 服务器异常
	ResponseCodeServerError ResponseCode = "SERVER_ERROR"
)

// HttpStatus 获取业务状态码对应的 HTTP 状态码
func (r ResponseCode) HttpStatus() int {
	switch r {
	case ResponseCodeBadRequest:
		return http.StatusBadRequest
	case ResponseCodeNotFound:
		return http.StatusNotFound
	case ResponseCodeServerError:
		return http.StatusInternalServerError
	default:
		return http.StatusOK // 默认 OK
	}
}
