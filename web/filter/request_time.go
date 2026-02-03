package filter

import (
	"context"
	"ispace/common/constant"
	"time"

	"github.com/gin-gonic/gin"
)

// RequestTime 请求时间，整个业务中保持统一
func RequestTime(g *gin.Context) {
	g.Request = g.Request.WithContext(context.WithValue(g.Request.Context(), constant.CtxKeyRequestTime, time.Now()))
}
