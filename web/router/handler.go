package router

import (
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
		} else if result != nil {

		}
		c.Next()
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
		slog.Warn("[handler] 未定义的 Handler 返回类型", slog.String("type", reflect.TypeOf(resp).Name()))
	}
}

func errorHandle(err error, ctx *gin.Context) {

	var serviceError common.ServiceError

	if !errors.As(err, &serviceError) {
		// 非业务异常
		if errors.Is(err, gorm.ErrRecordNotFound) {
			serviceError = common.NewServiceError(http.StatusNotFound, response.Fail(response.CodeNotFound).WithMessage("数据不存在"))
		} else if errors.Is(err, os.ErrNotExist) {
			serviceError = common.NewServiceError(http.StatusNotFound, response.Fail(response.CodeNotFound).WithMessage("文件不存在"))
		} else {
			serviceError = common.NewServiceError(http.StatusInternalServerError, response.Fail(response.CodeNotFound).WithMessage("服务器异常"))
		}
	}
	ctx.JSON(
		serviceError.StatusCode(),
		serviceError.Response(),
	)
}
