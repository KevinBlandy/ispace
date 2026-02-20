package service

import (
	"context"
	"errors"
	"io/fs"
	"ispace/common"
	"ispace/common/constant"
	"ispace/common/page"
	"ispace/common/response"
	"ispace/common/util"
	"ispace/db"
	"ispace/repo/model"
	"ispace/store"
	"ispace/web/handler/api"
	"log/slog"
	"net/http"
	"path/filepath"
	"strings"
	"time"

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

	// 删除资源数据
	_, err = gorm.G[model.Resource](db.Session(ctx)).Where("object_id = ?", id).Delete(ctx)
	if err != nil {
		return err
	}
	// 删除回收站数据
	_, err = gorm.G[model.RecycleBin](db.Session(ctx)).Where("recource_object_id = ?", id).Delete(ctx)
	// TODO 其他业务逻辑
	return err
}

// InvalidClean 清理无效的存储对象
// 已经落盘存储，但是没入库的磁盘资源
func (o *ObjectService) InvalidClean(ctx context.Context) error {

	// 7 天前
	weekAgo := time.Now().AddDate(0, 0, -7)

	bucket := store.DefaultStore()
	err := fs.WalkDir(bucket.FS(), ".", func(f string, d fs.DirEntry, err error) error {

		// 忽略文件夹
		if d.IsDir() {
			return nil
		}

		// 文件信息
		stat, err := d.Info()
		if err != nil {
			return err
		}

		// 文件最后修改时间为 7 天前
		if stat.ModTime().After(weekAgo) {
			return nil
		}

		//// 相对路径
		//relPath, err := filepath.Rel(bucket.Name(), f)
		//if err != nil {
		//	return err
		//}

		localFilePath := filepath.ToSlash(f)

		// 检索文件是否存在
		objectId, err := db.Transaction(ctx, func(ctx context.Context) (int64, error) {
			var objectId int64
			return objectId, db.Session(ctx).Raw("SELECT id FROM t_object WHERE path = ?", localFilePath).Scan(&objectId).Error
		})
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		// 文件不存在，则删除无效资源
		if objectId == 0 {
			slog.InfoContext(ctx, "删除无效资源",
				slog.String("path", localFilePath),
				slog.Time("modTime", stat.ModTime()),
			)
			if err := bucket.Remove(f); err != nil {
				slog.ErrorContext(ctx, "删除无效文件异常",
					slog.String("err", err.Error()),
					slog.String("path", localFilePath),
				)
				return err
			}
		}
		return nil
	})
	return err
}

// UpdateRefCount 更新 Ref 引用
func (o *ObjectService) UpdateRefCount(ctx context.Context, id int64, count int64) error {
	// 更新引用
	result := db.Session(ctx).
		Table(model.Object{}.TableName()).
		Where("id = ?", id).UpdateColumns(map[string]any{
		"update_time": util.ContextValueDefault(ctx, constant.CtxKeyRequestTime, time.Now()).UnixMilli(),
		"ref_count":   gorm.Expr("ref_count + ?", count),
	})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return common.NewServiceError(http.StatusBadRequest, response.Fail(response.CodeBadRequest).WithMessage("存储引用更新失败"))
	}
	return nil
}

var DefaultObjectService = NewObjectService()
