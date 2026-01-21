package common

import (
	"ispace/common/response"
)

// ServiceError 业务异常
type ServiceError interface {
	StatusCode() int              // 业务的 Http 状态码
	Response() *response.Response // 业务的响应数据
	error
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

	//statusTxt := http.StatusText(statusCode)
	//if statusTxt == "" {
	//	statusCode = http.StatusInternalServerError
	//}
	return &defaultServiceError{
		statusCode: statusCode,
		response:   response,
	}
}
