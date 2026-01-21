package router

import (
	"database/sql"
	"encoding/json"
	"errors"
	"ispace/common"
	"ispace/common/response"
	"log/slog"
	"net/http"
	"os"
	"reflect"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/render"
	"gorm.io/gorm"
)

// H 通用的处理器
func H(fn func(*gin.Context) (any, error)) gin.HandlerFunc {
	return func(c *gin.Context) {

		result, err := fn(c)

		if err != nil {
			errorHandle(err, c)
			c.Abort()
			return
		} else if result != nil {
			resultHandle(result, c)
		} else {
			// 继续执行链
			c.Next()
		}
	}
}

func resultHandle(resp any, ctx *gin.Context) {
	switch r := resp.(type) {
	case render.Render:
		if err := r.Render(ctx.Writer); err != nil {
			slog.Error("[handler] Response Render 异常", slog.String("err", err.Error()))
		}
	case *response.Response:
		ctx.JSON(http.StatusOK, r)
	default:
		ctx.Status(http.StatusNoContent)
		slog.Warn("[handler] 未定义的 Handler 返回类型", slog.String("type", reflect.TypeOf(resp).Name()))
	}
}

func errorHandle(err error, ctx *gin.Context) {

	// 最终都需要转换为业务异常
	var serviceError common.ServiceError

	switch {
	case errors.As(err, &serviceError):
	case os.IsNotExist(err),
		errors.Is(err, gorm.ErrRecordNotFound),
		errors.Is(err, sql.ErrNoRows):

		serviceError = common.NewServiceError(http.StatusNotFound, response.Fail(response.CodeNotFound).WithMessage("数据不存在"))

	case os.IsPermission(err):
		serviceError = common.NewServiceError(http.StatusForbidden, response.Fail(response.CodeForbidden).WithMessage("无权操作"))
	//case errors.Is(err, auth.ErrBadToken):
	//	serviceError = common.NewServiceError(http.StatusUnauthorized, response.Fail(response.CodeUnauthorized).WithMessage("请先登录"))
	default:
		// 默认服务器异常
		slog.Error("[handler] 服务器异常",
			slog.String("err", err.Error()),
		)
		serviceError = common.NewServiceError(
			http.StatusInternalServerError,
			response.Fail(response.CodeServerError).WithMessage("服务器异常："+err.Error()),
		)
	}

	ctx.Status(serviceError.StatusCode())

	responseBody := serviceError.Response()
	if responseBody != nil {
		ctx.Header("Content-Type", "application/json; charset=utf-8")

		encoder := json.NewEncoder(ctx.Writer)
		//encoder.SetIndent("", "  ")
		//encoder.SetEscapeHTML(false)

		if err := encoder.Encode(responseBody); err != nil {
			slog.Error("[handler] JSON 响应异常", slog.String("err", err.Error()))
		}
	}
}
