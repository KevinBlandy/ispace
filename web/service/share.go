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
	"ispace/web/handler/api"
	"net/http"
	"strings"

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
		) title
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
	t2.size,
	t2.status,
	-- 资源的实际创建时间
	t3.create_time
FROM
	t_share t
	INNER JOIN t_share_resource t1 ON t1.share_id = t.id
	LEFT JOIN t_object t2 ON t2.id = t1.resource_object_id AND t1.resource_dir = ?
	INNER JOIN t_resource t3 ON t3.id = t1.resource_id
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
	share, err := s.GetByIdentifier(ctx, identifier, "id", "path", "create_time", "enabled", "member_id")
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if share == nil || share.Id == 0 {
		return nil, common.NewServiceError(http.StatusBadRequest, response.Fail(response.CodeBadRequest).WithMessage("访问的资源不存在"))
	}
	if !share.Enabled {
		return nil, common.NewServiceError(http.StatusBadRequest, response.Fail(response.CodeBadRequest).WithMessage("资源被屏蔽了"))
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
		Id:   share.Id,
		Path: share.Path,
		Member: struct {
			Id       int64  `json:"id,string"`
			NickName string `json:"nickName"`
			Avatar   string `json:"avatar"`
		}{member.Id, member.NickName, member.Avatar},
		CreateTime: share.CreateTime,
	}, nil
}

func NewShareService() *ShareService {
	return &ShareService{}
}

var DefaultShareService = NewShareService()
