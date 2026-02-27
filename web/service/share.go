package service

import (
	"context"
	"errors"
	"ispace/common"
	"ispace/common/page"
	"ispace/common/response"
	"ispace/common/types"
	"ispace/common/util"
	"ispace/db"
	"ispace/repo/model"
	"ispace/store"
	"ispace/web/handler/api"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"
)

type ShareService struct{}

// List 分享列表
func (s ShareService) List(ctx context.Context, request *api.ShareListRequest) (*page.Pagination[*api.ShareListResponse], error) {
	query := &strings.Builder{}
	query.WriteString(`
	SELECT
		t.id,
		t.path,
		t.enabled,
		t.password,
		t.views,
		t.create_time,
		t.expire_time,
		-- 检索 root 节点的资源名称，拼接起来作为标题
		(
			SELECT GROUP_CONCAT(t2.resource_title, ',') FROM t_share_resource t2 WHERE t2.share_id = t.id AND t2.root = 1
		) title,
		(
			SELECT COUNT(1) FROM t_share_resource WHERE share_id = t.id AND resource_dir = 0
		) resources
	FROM
		t_share t
	WHERE
		t.member_id = ?
`)
	params := []any{request.MemberId}

	// 根据标题检索
	if request.Title != "" {
		query.WriteString(" AND EXISTS(SELECT 1 FROM t_share_resource t1 WHERE t1.share_id = t.id AND t1.resource_title LIKE ?)")
		params = append(params, "%"+request.Title+"%")
	}

	return db.PageQuery[api.ShareListResponse](ctx, request.Pager, query.String(), params)
}

// Update 更新
func (s ShareService) Update(ctx context.Context, request *api.ShareUpdateRequest) error {

	var params = make(map[string]any)

	if request.Password != "" {
		params["password"] = request.Password
	}
	if request.Enabled != nil {
		params["enabled"] = *request.Enabled
	}

	if len(params) == 0 {
		return nil
	}

	result := db.Session(ctx).Table(model.Share{}.TableName()).Where("id = ?", request.Id).UpdateColumns(params)

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return common.NewServiceError(http.StatusBadRequest, response.Fail(response.CodeBadRequest).WithMessage("更新失败"))
	}
	return nil
}

// Delete 资源删除
func (s ShareService) Delete(ctx context.Context, request *api.ShareDeleteRequest) error {
	session := db.Session(ctx)
	affected, err := gorm.G[model.Share](session).Where("id IN ? AND member_id = ?", request.Id, request.MemberId).Delete(ctx)
	if err != nil {
		return err
	}
	// 必须保证所有记录都删除成功
	if affected != len(request.Id) {
		return common.NewServiceError(http.StatusBadRequest, response.Fail(response.CodeBadRequest).WithMessage("资源删除失败"))
	}

	// 删除关联的子记录
	_, err = gorm.G[model.ShareResource](session).Where("share_id IN ?", request.Id).Delete(ctx)
	return err
}

// GetByIdentifier 根据检索记录
func (s ShareService) GetByIdentifier(ctx context.Context, identifier types.Identifier, columns ...string) (*model.Share, error) {
	numberId := identifier.Numeric()
	if len(columns) == 0 {
		return gorm.G[*model.Share](db.Session(ctx)).
			Where(
				util.If(numberId, "id = ?", "path = ?"),
				util.If[any](numberId, identifier.Int64(), identifier.String())).
			Take(ctx)
	}
	return gorm.G[*model.Share](db.Session(ctx)).
		Select(columns[0], columns[1:]).
		Where(
			util.If(numberId, "id = ?", "path = ?"),
			util.If[any](numberId, identifier.Int64(), identifier.String())).
		Take(ctx)
}

// ResourceList 资源列表
func (s ShareService) ResourceList(ctx context.Context, request *api.ShareResourceListRequest) ([]*api.ShareResourceListResponse, error) {
	share, err := s.GetByIdentifier(ctx, request.Identifier)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if share == nil || share.Id == 0 {
		return nil, common.NewServiceError(http.StatusBadRequest, response.Fail(response.CodeBadRequest).WithMessage("访问的资源不存在"))
	}
	if !share.Enabled {
		return nil, common.NewServiceError(http.StatusBadRequest, response.Fail(response.CodeBadRequest).WithMessage("资源被屏蔽了"))
	}

	// 构建查询
	query := &strings.Builder{}
	query.WriteString(`
SELECT
	t1.id,
	t1.resource_title title,
	t1.resource_dir dir,
	t1.resource_content_type contentType,
	t1.resource_create_time create_time,
	t2.size,
	t2.status
FROM
	t_share t
	INNER JOIN t_share_resource t1 ON t1.share_id = t.id
	LEFT JOIN t_object t2 ON t2.id = t1.resource_object_id AND t1.resource_dir = ?
WHERE
	t.id = ?
`)

	params := []any{false, share.Id}

	if request.ParentId == model.DefaultResourceParentId {
		// 查询 root 记录
		query.WriteString(" AND t1.root = ?")
		params = append(params, true)
	} else {
		// 检索子记录
		query.WriteString(` AND t1.resource_parent_id = (
			SELECT resource_id FROM t_share_resource WHERE id = ? AND share_id = ?
		)`)
		params = append(params, request.ParentId, share.Id)
	}

	if request.Title != "" {
		query.WriteString(" AND t1.resource_title LIKE ?")
		params = append(params, "%"+request.Title+"%")
	}

	return db.List[api.ShareResourceListResponse](ctx, query.String(), params...)
}

// Share 分享资源的详情
func (s ShareService) Share(ctx context.Context, identifier types.Identifier) (*api.ShareResponse, error) {
	share, err := s.GetByIdentifier(ctx, identifier, "id", "path", "create_time", "enabled", "member_id", "views")
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if share == nil || share.Id == 0 {
		return nil, common.NewServiceError(http.StatusNotFound, response.Fail(response.CodeNotFound).WithMessage("访问的资源不存在"))
	}
	if !share.Enabled {
		return nil, common.NewServiceError(http.StatusForbidden, response.Fail(response.CodeForbidden).WithMessage("资源被屏蔽了"))
	}

	// 检索会员信息
	member, err := gorm.G[model.Member](db.Session(ctx)).Select("id", "nick_name", "avatar").Where("id = ?", share.MemberId).Take(ctx)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if member.Id == 0 {
		return nil, common.NewServiceError(http.StatusBadRequest, response.Fail(response.CodeBadRequest).WithMessage("会员信息异常"))
	}
	return &api.ShareResponse{
		Id:    share.Id,
		Path:  share.Path,
		Views: share.Views,
		Member: struct {
			Id       int64  `json:"id,string"`
			NickName string `json:"nickName"`
			Avatar   string `json:"avatar"`
		}{member.Id, member.NickName, member.Avatar},
		CreateTime: share.CreateTime,
	}, nil
}

// GetResource 检索资源
func (s ShareService) GetResource(ctx context.Context, identifier types.Identifier, shareResourceId int64) (ret struct {
	Id          int64
	Title       string
	Compression model.ObjectCompression
	ContentType string
	Status      model.ObjectStatus
	Path        string
}, err error) {

	query := &strings.Builder{}
	query.WriteString(`
		SELECT
			t1.id,
			t1.resource_title title,
			t2.compression,
			t1.resource_content_type contentType,
			t2.status,
			t2.path
		FROM
			t_share t
			INNER JOIN t_share_resource t1 ON t1.share_id = t.id
			INNER JOIN t_object t2 ON t2.id = t1.resource_object_id
		WHERE`,
	)

	params := make([]any, 0)

	if identifier.Numeric() {
		query.WriteString(" t.id = ?")
		params = append(params, identifier.Int64())
	} else {
		query.WriteString(" t.path = ?")
		params = append(params, identifier.String())
	}

	query.WriteString(` AND t1.id = ? AND t1.resource_dir = ?`)
	params = append(params, shareResourceId, false)

	err = db.Session(ctx).Raw(query.String(), params...).Row().Scan(
		&ret.Id,
		&ret.Title,
		&ret.Compression,
		&ret.ContentType,
		&ret.Status,
		&ret.Path,
	)
	return
}

// Download 资源下载树
func (s ShareService) Download(ctx context.Context, request *api.ShareResourceDownloadRequest) ([]*store.DownloadTree, error) {

	session := db.Session(ctx)

	var shareId int64

	// 统一根据 id 进行检索
	if request.Identifier.Numeric() {
		shareId = request.Identifier.Int64()
	} else {
		share, err := s.GetByIdentifier(ctx, request.Identifier, "id")
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		shareId = share.Id
	}

	if shareId < 1 {
		return nil, common.NewServiceError(http.StatusBadRequest, response.Fail(response.CodeBadRequest).WithMessage("资源不存在"))
	}

	// 检索资源列表
	rows, err := session.Raw(`
			SELECT
				t1.resource_id id,
				t1.resource_parent_id parent_id,
				t1.resource_title title,
				t1.resource_dir dir,
				ifnull(t2.path, '') path,
				ifnull(t2.compression, '') compression
			FROM
				t_share t
				INNER JOIN t_share_resource t1 ON t1.share_id = t.id
				LEFT JOIN t_object t2 ON t2.id = t1.resource_object_id AND t1.resource_dir = ?
			WHERE
				t.id = ?
			AND
				t1.id IN ?`, false, shareId, request.Id).Rows()

	if err != nil {
		return nil, err
	}

	defer util.SafeClose(rows)

	resources := make([]*store.DownloadTree, 0)

	for rows.Next() {
		resource := new(store.DownloadTree)
		if err := rows.Scan(&resource.Id, &resource.ParentId, &resource.Title, &resource.Dir, &resource.Path, &resource.Compression); err != nil {
			return nil, err
		}
		resources = append(resources, resource)
	}

	// 构建完整的订单树
	var subTree = func(r *store.DownloadTree) error {
		// 检索树下的所有记录
		rows, err := session.Raw(`
			SELECT
				t1.resource_id id,
				t1.resource_parent_id parent_id,
				t1.resource_title title,
				t1.resource_dir dir,
				ifnull(t2.path, '') path,
				ifnull(t2.compression, '') compression
			FROM
				t_share t
				INNER JOIN t_share_resource t1 ON t1.share_id = t.id
				LEFT JOIN t_object t2 ON t2.id = t1.resource_object_id AND t1.resource_dir = ?
			WHERE
				t.id = ?
			AND
				t1.resource_path LIKE CONCAT(
					(SELECT resource_path FROM t_share_resource WHERE id = ?),
					'%'
				)
			AND
				t1.id <> ?
		`, false, shareId, r.Id, r.Id).Rows()

		if err != nil {
			return err
		}

		defer util.SafeClose(rows)

		// Map 结构存储
		var resourceMap = make(map[int64]*store.DownloadTree)
		for rows.Next() {
			resource := new(store.DownloadTree)
			if err := rows.Scan(&resource.Id, &resource.ParentId, &resource.Title, &resource.Dir, &resource.Path, &resource.Compression); err != nil {
				return err
			}
			resourceMap[resource.Id] = resource
		}

		if len(resourceMap) == 0 {
			return nil
		}

		// 顶级记录
		for _, resource := range resourceMap {
			if resource.ParentId == r.Id {
				r.Entries = append(r.Entries, resource)
				delete(resourceMap, resource.Id)
			}
		}

		var subEntry func(*store.DownloadTree, map[int64]*store.DownloadTree)

		subEntry = func(r *store.DownloadTree, m map[int64]*store.DownloadTree) {
			r.Entries = make([]*store.DownloadTree, 0)
			for _, resource := range m {
				if resource.ParentId == r.Id {
					r.Entries = append(r.Entries, resource)
					delete(m, resource.Id)
				}
			}
			if len(r.Entries) > 0 {
				for _, resource := range r.Entries {
					subEntry(resource, m)
				}
			}
		}

		// 递归构建所有的子记录
		for _, entry := range r.Entries {
			subEntry(entry, resourceMap)
		}

		return nil
	}

	waitGroup := new(sync.WaitGroup)

	for _, resource := range resources {
		// 目录的话，构建完整的文件树
		if resource.Dir {
			waitGroup.Go(func() {
				if err := subTree(resource); err != nil {
					slog.ErrorContext(ctx, "检索资源树异常", slog.String("err", err.Error()))
				}
			})
		}
	}

	waitGroup.Wait()

	return resources, nil
}

// IncrViews 递增访问次数
func (s ShareService) IncrViews(ctx context.Context, id int64, views int) error {

	result := db.Session(ctx).Table(model.Share{}.TableName()).
		Where("id = ?", id).
		UpdateColumns(map[string]any{
			"views": gorm.Expr("views + ?", views),
		})

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return common.NewServiceError(http.StatusBadRequest, response.Fail(response.CodeBadRequest).WithMessage("更新失败"))
	}
	return nil
}

// Clean 清理过期的分享记录
func (s ShareService) Clean(ctx context.Context) (int64, error) {
	var counter int64

	now := time.Now().UnixMilli()

	rows, err := db.Session(ctx).Raw("SELECT id, member_id FROM t_share WHERE expire_time > 0 AND expire_time < ?", now).Rows()
	if err != nil {
		return counter, err
	}
	defer util.SafeClose(rows)

	for rows.Next() {
		var shareId, memberId int64
		if err := rows.Scan(&shareId, &memberId); err != nil {
			return counter, err
		}
		// 删除
		err := s.Delete(ctx, &api.ShareDeleteRequest{
			MemberId: memberId,
			Id:       types.Int64Slice{shareId},
		})
		if err != nil {
			return counter, err
		}

		counter++
	}

	return counter, nil
}

func NewShareService() *ShareService {
	return &ShareService{}
}

var DefaultShareService = NewShareService()
