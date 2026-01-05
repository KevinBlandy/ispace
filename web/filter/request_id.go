package filter

import (
	"context"
	"ispace/common/constant"
	"strconv"
	"sync/atomic"

	"github.com/gin-gonic/gin"
)

func NewRequestIdFilter() gin.HandlerFunc {

	var idGenerator = atomic.Uint64{}

	return func(c *gin.Context) {

		requestId := idGenerator.Add(1)

		c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), constant.CtxKeyRequestId, requestId))

		c.Header(constant.HttpHeaderRequestId, strconv.FormatUint(requestId, 10))
		c.Next()
	}
}
