package member

import (
	"context"
	"errors"
	"ispace/common"
	"ispace/common/rege"
	"ispace/common/response"
	"ispace/db"
	"ispace/repo/model"
	"ispace/web/handler/api"
	"ispace/web/service"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ObjectApi struct {
	os *service.ObjectService
}

func NewObjectApi(os *service.ObjectService) *ObjectApi {
	return &ObjectApi{os: os}
}

// Hash 根据资源的 Hash 查询记录是否存在
func (o *ObjectApi) Hash(c *gin.Context) (any, error) {
	h := strings.ToLower(c.Param("hash"))
	if !rege.Sha256Hex.MatchString(h) {
		return nil, common.NewServiceError(http.StatusBadRequest, response.Fail(response.CodeBadRequest).WithMessage("非法请求"))
	}
	objectId, err := db.Transaction(c.Request.Context(), func(ctx context.Context) (int64, error) {
		var result int64
		return result, db.Session(ctx).Table(model.Object{}.TableName()).Select("id").Where("hash = ?", h).Scan(&result).Error
	})
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	return response.Ok(&api.ObjectHashResponse{Hit: objectId > 0}), nil
}

var DefaultObjectApi = NewObjectApi(service.DefaultObjectService)
