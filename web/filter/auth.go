package filter

import (
	"ispace/common/constant"

	"github.com/gin-gonic/gin"
)

func AuthFilter(c *gin.Context) (any, error) {
	// TODO 认证
	c.Set(constant.CtxKeySubject, int64(1000))
	return nil, nil
}
