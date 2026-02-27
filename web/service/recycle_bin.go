package service

import (
	"container/list"
	"context"
	"errors"
	"ispace/common"
	"ispace/common/constant"
	"ispace/common/page"
	"ispace/common/response"
	"ispace/common/util"
	"ispace/db"
	"ispace/repo/model"
	"ispace/web/handler/api"
	"net/http"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
)

type RecycleBinService struct {
	objectService   *ObjectService
	memberService   *MemberService
	resourceService *ResourceService
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
	ORDER BY t.create_time DESC
`)

	var condition = []any{false, request.MemberId, true}
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

	query.WriteString("member_id = ?")
	params = append(params, request.MemberId)

	if len(request.Id) > 0 {
		// 删除指定的记录
		query.WriteString(" AND id IN ?")
		params = append(params, request.Id)
	} else {
		// 如果没传 ID，则表示删除所有，直接检索所有的 root 记录
		query.WriteString(" AND root = ?")
		params = append(params, true)
	}

	session := db.Session(ctx)

	// TODO 数据量过大的情况下，应该流式迭代
	results, err := gorm.G[*model.RecycleBin](session).
		Select("id", "root", "resource_id", "resource_object_id", "resource_dir", "member_id").
		Where(query.String(), params...).
		Order("id DESC"). // 根据 ID 逆序，后删除的先执行
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
// id
// ResourceId
// ResourceDir
// ResourceObjectId
func (s RecycleBinService) Remove(ctx context.Context, m *model.RecycleBin) error {

	// TODO 聚合，批量执行

	session := db.Session(ctx)
	if m.ResourceDir {
		// 递归删除不包含 root 的子项目
		// 回收站中，同一个资源不可能被删除两次
		results, err := gorm.G[*model.RecycleBin](session).
			Select("id", "root", "resource_id", "resource_object_id", "resource_dir", "member_id").
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
	// 递减用户的空间使用量
	var size int64
	if err := session.Raw("SELECT size FROM t_object where id = ?", m.ResourceObjectId).Row().Scan(&size); err != nil {
		return err
	}
	if err := s.memberService.AddUsedStorageSpace(ctx, m.MemberId, -size); err != nil {
		return err
	}
	// 递减对应的引用计数
	return s.objectService.UpdateRefCount(ctx, m.ResourceObjectId, -1)
}

// Restore 恢复文件
// 只能恢复 root 项目
// 如果是恢复所有，则按照删除时间逆序进行恢复
func (s RecycleBinService) Restore(ctx context.Context, request *api.RecycleBinRestoreRequest) error {

	var query = &strings.Builder{}
	var params []any

	// 如果没传 ID，则表示删除所有
	if len(request.Id) > 0 {
		query.WriteString("id IN ? AND ")
		params = append(params, request.Id)
	}

	query.WriteString("member_id = ? AND root = ?") // 只能恢复 root 项目
	params = append(params, request.MemberId, true)

	session := db.Session(ctx)
	// TODO 考虑流式
	results, err := gorm.G[*model.RecycleBin](session).
		Where(query.String(), params...).
		Order("create_time DESC").
		Find(ctx)

	if err != nil {
		return err
	}

	for _, result := range results {

		// 子项目
		var entries []*model.RecycleBin

		// 是目录的话，检索其所有的非 root 级子项目
		if result.ResourceDir {

			queue := list.New()

			subEntries, err := gorm.G[*model.RecycleBin](session).
				Where("resource_parent_id = ? AND root = ?", result.ResourceId, false).
				Find(ctx)
			if err != nil {
				return err
			}

			for _, entry := range subEntries {
				entries = append(entries, entry)
				if entry.ResourceDir {
					queue.PushBack(entry)
				}
			}
			for queue.Len() > 0 {
				item := queue.Remove(queue.Front()).(*model.RecycleBin)
				subEntries, err = gorm.G[*model.RecycleBin](session).
					Where("resource_parent_id = ? AND root = ?", item.ResourceId, false).
					Find(ctx)
				if err != nil {
					return err
				}

				for _, entry := range subEntries {
					entries = append(entries, entry)
					if entry.ResourceDir {
						queue.PushBack(entry)
					}
				}
			}
		}

		// 执行恢复
		if err := s.restore(ctx, result, entries); err != nil {
			return err
		}

		// 删除回收站资源
		var ids = []int64{result.Id}
		for _, entry := range entries {
			ids = append(ids, entry.Id)
		}
		if _, err := gorm.G[*model.RecycleBin](session).Where("id IN ?", ids).Delete(ctx); err != nil {
			return err
		}
	}
	return nil
}

// restore 恢复文件
func (s RecycleBinService) restore(ctx context.Context, root *model.RecycleBin, entries []*model.RecycleBin) error {

	now := util.ContextValueDefault(ctx, constant.CtxKeyRequestTime, time.Now()).UnixMilli()

	var noChange bool

	session := db.Session(ctx)

	// 检索删除前的父资源
	if root.ResourceParentId != model.DefaultResourceParentId {
		// 检索父目录
		parent, err := gorm.G[model.Resource](session).
			Select("id", "path").
			Where("id = ? AND member_id = ?", root.ResourceParentId, root.MemberId).
			Take(ctx)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		// 父目录存在，且没移动过位置，则直接恢复记录
		if parent.Id > 0 && strings.HasPrefix(root.ResourcePath, parent.Path) {
			noChange = true
		}
	} else {
		// 本身就是从根目录删除的资源
		noChange = true
	}

	// 父目录移动过位置，则恢复到根目录
	if !noChange {

		// 结构树的共同前缀
		prefix := strings.TrimSuffix(root.ResourcePath, strconv.FormatInt(root.ResourceId, 10)+model.ResourcePathSeparator)

		// root 资源移动到根目录
		root.ResourceParentId = model.DefaultResourceParentId

		// 修改所有子结构
		for _, entry := range append(entries, root) {
			entry.ResourcePath = strings.ReplaceAll(entry.ResourcePath, prefix, "")
			entry.ResourceDepth = entry.ResourceDepth - root.ResourceDepth
		}
	}

	// 重名处理，可能在删除后又新建了同名资源
	var err error
	root.ResourceTitle, err = s.resourceService.UniqueTitle(ctx,
		root.ResourceDir,
		root.ResourceTitle,
		root.ResourceId,
		root.MemberId,
		root.ResourceParentId,
	)
	if err != nil {
		return err
	}

	// 恢复资源到目录
	var resources []*model.Resource
	for _, entry := range append(entries, root) {
		resources = append(resources, &model.Resource{
			Id:          entry.ResourceId,
			ObjectId:    entry.ResourceObjectId,
			ParentId:    entry.ResourceParentId,
			Title:       entry.ResourceTitle,
			ContentType: entry.ResourceContentType,
			Dir:         entry.ResourceDir,
			Path:        entry.ResourcePath,
			Depth:       entry.ResourceDepth,
			CreateTime:  entry.ResourceCreateTime,

			MemberId:   entry.MemberId,
			UpdateTime: now,
		})
	}
	return gorm.G[*model.Resource](session).CreateInBatches(ctx, &resources, 100)
}

// Entries 项目列表
func (s RecycleBinService) Entries(ctx context.Context, memberId int64, id int64) ([]*api.RecycleBinEntryResponse, error) {
	return db.List[api.RecycleBinEntryResponse](ctx, `SELECT 
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
		t.resource_parent_id = (
			SELECT t2.resource_id FROM t_recycle_bin t2 WHERE t2.id = ?
		)
	AND
		t.root = ?
`, true, memberId, id, false)
}

// GetObject 查询回收站项目对象
func (s RecycleBinService) GetObject(ctx context.Context, memberId int64, id int64) (ret struct {
	Id          int64
	Title       string
	Status      model.ObjectStatus
	Path        string
	Compression model.ObjectCompression
}, err error) {

	err = db.Session(ctx).Raw(`SELECT
		t.id,
		t.resource_title title,
		t1.status,
		t1.compression,
		t1.path
	FROM
		t_recycle_bin t
	INNER JOIN t_object t1 ON t1.id = t.resource_object_id
	WHERE
		t.id = ?
	AND
		t.member_id = ?
	AND
		t.resource_dir = ?`, id, memberId, false).Row().Scan(&ret.Id, &ret.Title, &ret.Status, &ret.Compression, &ret.Path)
	return
}

// Clean 清理过期的回收站内容
func (s RecycleBinService) Clean(ctx context.Context) (int64, error) {
	t := time.Now().AddDate(0, 0, -30).UnixMilli() // 小于30 天前的都删除

	//t := time.Now().Add(-time.Minute).UnixMilli()

	session := db.Session(ctx)

	rows, err := session.
		Raw("SELECT id, root, resource_id, resource_object_id, resource_dir, member_id FROM t_recycle_bin WHERE create_time <= ?  AND root = ?", t, true).
		Rows()

	if err != nil {
		return 0, err
	}
	defer util.SafeClose(rows)

	var total int64

	for rows.Next() {
		var m model.RecycleBin
		if err := session.ScanRows(rows, &m); err != nil {
			return total, err
		}
		// 删除
		if err := s.Remove(ctx, &m); err != nil {
			return total, err
		}
		total += 1
	}

	return total, nil
}

var DefaultRecycleBinService = &RecycleBinService{
	memberService:   DefaultMemberService,
	objectService:   DefaultObjectService,
	resourceService: DefaultResourceService,
}
