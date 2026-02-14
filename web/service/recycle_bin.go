package service

import (
	"context"
	"ispace/common"
	"ispace/common/page"
	"ispace/common/response"
	"ispace/db"
	"ispace/repo/model"
	"ispace/web/handler/api"
	"net/http"
	"strings"

	"gorm.io/gorm"
)

type RecycleBinService struct {
	objectService *ObjectService
}

/*
CREATE TABLE `t_recycle_bin` (`id` integer PRIMARY KEY AUTOINCREMENT,`member_id` integer,`root` numeric,`create_time` integer,`resource_id` integer,`resource_object_id` integer,`resource_parent_id` integer,`resource_title` text,`resource_content_type` text,`resource_dir` numeric,`resource_path` text,`resource_create_time` integer);

CREATE INDEX `idx_t_recycle_bin_member_id` ON `t_recycle_bin`(`member_id`);

*/

// List 分页检索
func (s RecycleBinService) List(ctx context.Context, request *api.RecycleBinListRequest) (*page.Pagination[*api.RecycleBinListResponse], error) {
	var query = &strings.Builder{}
	query.WriteString(`SELECT 
		t.id,
		t.resource_title title,
		t.resource_content_type content_type,
		t.resource_dir dir,
		t.create_time,
		t1.size,
		t1.status
	FROM
		t_recycle_bin t
		LEFT JOIN t_object t1 ON t1.id = t.resource_object_id AND t.resource_dir = ?
	WHERE
		t.member_id = ?
	AND
		t.root = ?
`)

	var condition = []any{true, request.MemberId, true}
	if request.Title != "" {
		query.WriteString(" AND resource_title like ?")
		condition = append(condition, "%"+request.Title+"%")
	}
	return db.PageQuery[api.RecycleBinListResponse](ctx, request.Pager, query.String(), condition)
}

// Delete 删除用户回收站内容
func (s RecycleBinService) Delete(ctx context.Context, request *api.RecycleBinDeleteRequest) error {

	var query = &strings.Builder{}
	var params []any

	// 如果没传 ID，则表示删除所有
	if len(request.Id) > 0 {
		query.WriteString("id IN ? AND ")
		params = append(params, request.Id)
	}

	query.WriteString("member_id = ?")
	params = append(params, request.MemberId)

	session := db.Session(ctx)
	results, err := gorm.G[*model.RecycleBin](session).
		Select("id", "root", "resource_id", "resource_object_id", "resource_dir").
		Where(query.String(), params...).
		Order("create_time DESC").
		Find(ctx)

	if err != nil {
		return err
	}
	for _, result := range results {
		// 执行删除
		if err := s.Remove(ctx, result); err != nil {
			return err
		}
	}
	return nil
}

// Remove 删除回收站的内容
func (s RecycleBinService) Remove(ctx context.Context, m *model.RecycleBin) error {

	// TODO 聚合，批量执行

	session := db.Session(ctx)
	if m.ResourceDir {
		// 递归删除不包含 root 的子项目
		results, err := gorm.G[*model.RecycleBin](session).
			Select("id", "root", "resource_id", "resource_object_id", "resource_dir").
			Where("resource_parent_id = ? AND root = ?", m.ResourceId, false).
			Find(ctx)

		if err != nil {
			return err
		}

		for _, result := range results {
			// 递归删除元素
			if err := s.Remove(ctx, result); err != nil {
				return err
			}
		}
	}

	return s.delete(ctx, m)
}

// delete 彻底删除资源
func (s RecycleBinService) delete(ctx context.Context, m *model.RecycleBin) error {
	// 删除目录
	session := db.Session(ctx)
	affected, err := gorm.G[model.RecycleBin](session).Where("id = ?", m.Id).Delete(ctx)
	if err != nil {
		return err
	}
	if affected == 0 {
		return common.NewServiceError(http.StatusBadRequest, response.Fail(response.CodeBadRequest).WithMessage("资源删除失败"))
	}
	if m.ResourceDir {
		return nil
	}

	// 如果是文件的话，还要递减对应的引用计数
	return s.objectService.UpdateRefCount(ctx, m.ResourceObjectId, -1)
}

var DefaultRecycleBinService = &RecycleBinService{objectService: DefaultObjectService}
