package filter

import (
	"context"
	"ispace/common/constant"
	"time"

	"github.com/gin-gonic/gin"
)

// TimeZone 客户端时区
func TimeZone(defaultTimeZone *time.Location, header string) gin.HandlerFunc {
	return func(c *gin.Context) {

		var timeZone = defaultTimeZone

		value := c.GetHeader(header)

		if value != "" {
			// value 如果为 ""，则会返回 UTC 时区
			if loc, err := time.LoadLocation(value); err == nil {
				timeZone = loc
			}
		}

		c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), constant.CtxKeyTimezone, timeZone))
	}
}
