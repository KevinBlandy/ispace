package service

import (
	"context"
	"errors"
	"io/fs"
	"ispace/common"
	"ispace/common/page"
	"ispace/common/response"
	"ispace/db"
	"ispace/repo/model"
	"ispace/store"
	"ispace/web/handler/api"
	"net/http"
	"strings"

	"gorm.io/gorm"
)

type ObjectService struct{}

func NewObjectService() *ObjectService {
	return &ObjectService{}
}

// List 分页检索数据
func (o *ObjectService) List(ctx context.Context, request *api.ObjectListRequest) (*page.Pagination[*api.ObjectListResponse], error) {
	query := strings.Builder{}
	query.WriteString("SELECT * FROM t_object WHERE 1=1")

	var args = make([]any, 0)
	// 状态过滤
	if request.Status != "" {
		query.WriteString(" AND status = ?")
		args = append(args, request.Status)
	}
	return db.PageQuery[api.ObjectListResponse](ctx, request.Pager, query.String(), args)
}

// Exists 根据 column = value 判断记录是否存在
// 如果 id = 0，则表示记录不存在
func (o *ObjectService) Exists(ctx context.Context, column string, value any) (id int64, err error) {
	err = db.Session(ctx).Table(model.Object{}.TableName()).Select("id").Where(column+" = ?", value).Scan(&id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, nil
	}
	return
}

// Update 更新资源记录
func (o *ObjectService) Update(ctx context.Context, request *api.ObjectUpdateRequest) error {
	// 更新参数
	var updateMap = make(map[string]any)
	if request.Status != "" {
		updateMap["status"] = request.Status
	}

	if len(updateMap) == 0 {
		return nil
	}
	result := db.Session(ctx).
		Table(model.Object{}.TableName()).
		Where("id = ?", request.Id).
		UpdateColumns(updateMap)

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return common.NewServiceError(http.StatusBadRequest, response.Fail(response.CodeBadRequest).WithMessage("资源信息更新失败"))
	}
	return nil
}

// Delete 删除资源
func (o *ObjectService) Delete(ctx context.Context, request *api.ObjectDeleteRequest) error {
	for _, v := range request.Id {
		if err := o.deleteById(ctx, v); err != nil {
			return err
		}
	}
	return nil
}

// deleteById 根据 id 删除记录
func (o *ObjectService) deleteById(ctx context.Context, id int64) error {
	rowAffected, err := gorm.G[model.Object](db.Session(ctx)).Where("id = ?", id).Delete(ctx)
	if err != nil {
		return err
	}
	if rowAffected != 1 {
		return common.NewServiceError(http.StatusBadRequest, response.Fail(response.CodeBadRequest).WithMessage("资源不存在"))
	}

	// 删除关联的数据
	_, err = gorm.G[model.Resource](db.Session(ctx)).Where("object_id = ?", id).Delete(ctx)
	if err != nil {
		return err
	}
	// TODO 其他业务逻辑
	return nil
}

// InvalidClean 清理无效的存储对象
func (o *ObjectService) InvalidClean(ctx context.Context) error {
	bucket := store.DefaultStore()
	err := fs.WalkDir(bucket.FS(), ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		// TODO 迭代每个对象，判断是否是孤儿对象
		return nil
	})
	return err
}

var DefaultObjectService = NewObjectService()
