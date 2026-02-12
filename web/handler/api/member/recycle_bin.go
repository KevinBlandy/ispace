package member

import (
	"ispace/web/service"

	"github.com/gin-gonic/gin"
)

type RecycleBinApi struct {
	service *service.RecycleBinService
}

// List 回收站项目列表
func (r RecycleBinApi) List(c *gin.Context) (any, error) {
	return nil, nil
}

func NewRecycleBinApi(binService service.RecycleBinService) *RecycleBinApi {
	return &RecycleBinApi{service: &binService}
}
