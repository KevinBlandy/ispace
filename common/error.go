package common

import "ispace/common/response"

type ServiceError interface {
	StatusCode() int              // Http 状态码
	Response() *response.Response // 响应
}

type defaultServiceError struct {
	statusCode int
	response   *response.Response
}

func (d defaultServiceError) Error() string {
	return d.response.Message
}
func (d defaultServiceError) StatusCode() int {
	return d.statusCode
}
func (d defaultServiceError) Response() *response.Response {
	return d.response
}

func NewServiceError(statusCode int, response *response.Response) ServiceError {
	return &defaultServiceError{
		statusCode: statusCode,
		response:   response,
	}
}
