package api

import (
	"context"
	"errors"
	"ispace/common/constant"
	"ispace/common/response"
	"ispace/common/util"
	"ispace/db"
	"ispace/repo/model"
	"ispace/web"
	"ispace/web/service"
	"strconv"
	"sync"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ResourceApi struct {
}

func NewResourceApi() *ResourceApi {
	return &ResourceApi{}
}

// List 资源列表
func (r ResourceApi) List(ctx *gin.Context) (any, error) {
	memberId := ctx.GetInt64(constant.CtxKeySubject)
	parentId, err := strconv.ParseInt(ctx.Query("parentId"), 10, 64)
	if err != nil || parentId < 0 {
		parentId = model.DefaultResourceParentId
	}

	result, err := db.Transaction(ctx.Request.Context(), func(ctx context.Context) ([]*web.ResourceListApiResponse, error) {

		var ret = make([]*web.ResourceListApiResponse, 0)

		session := db.Session(ctx)
		rows, err := session.Raw(`
			SELECT
				id,
				parent_id,
				title,
				dir,
				create_time,
				update_time,
				(
					CASE dir 
						WHEN 1 THEN 0
						WHEN 0 THEN (SELECT size from t_object WHERE id = object_id)
					END
				) size
			FROM
				t_resource
			WHERE
				member_id = ? AND parent_id = ?
			ORDER BY dir DESC, title ASC
		`, memberId, parentId).Rows()

		if err != nil {
			return nil, err
		}

		defer util.SafeClose(rows)

		for rows.Next() {
			resource := &web.ResourceListApiResponse{}
			if err := session.ScanRows(rows, resource); err != nil {
				return nil, err
			}
			ret = append(ret, resource)
		}

		return ret, nil
	}, db.TxReadOnly)

	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	return response.Ok(result), nil
}

// Upload 上传文件
func (r ResourceApi) Upload(ctx *gin.Context) (any, error) {

	defer util.SafeClose(ctx.Request.Body)

	multipartForm, err := ctx.MultipartForm()
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = multipartForm.RemoveAll()
	}()

	// 上传目录
	var parentId = model.DefaultResourceParentId
	parentIds := multipartForm.Value["parentId"]
	if len(parentIds) > 0 {
		parentId, err = strconv.ParseInt(parentIds[0], 10, 64)
		if err != nil || parentId < 0 {
			parentId = model.DefaultResourceParentId
		}
	}

	// 会员 ID
	var memberId = ctx.GetInt64(constant.CtxKeySubject)

	for _, files := range multipartForm.File {
		for _, file := range files {
			if err := service.DefaultResourceService.Upload(ctx.Request.Context(), memberId, parentId, file); err != nil {
				return nil, err
			}
		}
	}
	return nil, nil
}

var DefaultResourceApi = sync.OnceValue(func() *ResourceApi {
	return NewResourceApi()
})
