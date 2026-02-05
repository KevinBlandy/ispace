package service

import (
	"context"
	"errors"
	"ispace/common"
	"ispace/common/constant"
	"ispace/common/id"
	"ispace/common/page"
	"ispace/common/response"
	"ispace/common/util"
	"ispace/db"
	"ispace/repo/model"
	"ispace/web/handler/api"
	"net/http"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type MemberService struct {
	resourceService *ResourceService
}

func NewMemberService(resourceService *ResourceService) *MemberService {
	return &MemberService{resourceService: resourceService}
}

// Login 登录
func (m *MemberService) Login(ctx context.Context, request *api.MemberSignInRequest) (*model.Member, error) {

	member, err := gorm.G[model.Member](db.Session(ctx)).
		Select("id", "password", "enabled").
		Where("account = ?", request.Account).
		Take(ctx)

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, common.NewServiceError(http.StatusBadRequest, response.Fail(response.CodeBadRequest).WithMessage("用户名或密码错误"))
		}
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(member.Password), []byte(request.Password)); err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return nil, common.NewServiceError(http.StatusBadRequest, response.Fail(response.CodeBadRequest).WithMessage("用户名或密码错误"))
		}
		return nil, err
	}

	if !member.Enabled {
		return nil, common.NewServiceError(http.StatusForbidden, response.Fail(response.CodeForbidden).WithMessage("账户被封禁"))
	}

	return &member, nil
}

// List 分页检索会员列表
func (m *MemberService) List(ctx context.Context, request *api.MemberListRequest) (*page.Pagination[*api.MemberListResponse], error) {
	query := strings.Builder{}
	query.WriteString(`
				SELECT
					id,
					nick_name,
					avatar,
					account,
					email,
					enabled,
					create_time,
					update_time,
					-- 文件资源数量
					(
						SELECT COUNT(1) FROM t_resource t1 WHERE t1.member_id = t.id AND t1.dir = 0
					)  resources,
					-- 文件资源大小
					(
						SELECT 
							SUM(t2.size) 
						FROM 
							t_resource t1 
							INNER JOIN t_object t2 ON t2.id = t1.object_id
						WHERE 
							t1.member_id = t.id 
						AND 
							t1.dir = 0
					) resource_size
				FROM
					t_member t
				WHERE 1=1
				`)

	// 查询条件
	var condition = make([]any, 0)
	if request.Account != "" {
		query.WriteString(" AND account LIKE ?")
		condition = append(condition, "%"+request.Account+"%")
	}
	if request.Email != "" {
		query.WriteString(" AND email LIKE ?")
		condition = append(condition, "%"+request.Email+"%")
	}
	return db.PageQuery[api.MemberListResponse](ctx, request.Pager, query.String(), condition)
}

// Exists 根据 column = value 检索 ID
func (m *MemberService) Exists(ctx context.Context, column string, value any) (ret int64, err error) {
	err = db.Session(ctx).Table(model.Member{}.TableName()).Select("id").Where(column+" = ?", value).Scan(&ret).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		err = nil
	}
	return
}

// Create 创建新的会员
func (m *MemberService) Create(ctx context.Context, request *api.MemberCreateRequest) error {
	// 账户是否重复
	existsId, err := m.Exists(ctx, "account", request.Account)
	if err != nil {
		return err
	}
	if existsId > 0 {
		return common.NewServiceError(http.StatusBadRequest, response.Fail(response.CodeBadRequest).WithMessage("账户已经存在"))
	}
	// 邮箱
	existsId, err = m.Exists(ctx, "email", request.Email)
	if err != nil {
		return err
	}
	if existsId > 0 {
		return common.NewServiceError(http.StatusBadRequest, response.Fail(response.CodeBadRequest).WithMessage("邮箱已经存在"))
	}

	// 保持新的账户
	password, err := bcrypt.GenerateFromPassword([]byte(request.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	now := time.Now().UnixMilli()

	return db.Session(ctx).Create(&model.Member{
		Id:         id.Next().Int64(),
		NickName:   request.NickName,
		Avatar:     "", // TODO 随机头像
		Account:    request.Account,
		Password:   string(password),
		Email:      request.Email,
		Enabled:    request.Enabled,
		CreateTime: now,
		UpdateTime: now,
	}).Error
}

// Update 更新
func (m *MemberService) Update(ctx context.Context, request *api.MemberUpdateRequest) error {

	// 账户重复
	if request.Account != "" {
		existsId, err := m.Exists(ctx, "account", request.Account)
		if err != nil {
			return err
		}
		if existsId > 0 && existsId != request.Id {
			return common.NewServiceError(http.StatusBadRequest, response.Fail(response.CodeBadRequest).WithMessage("账户已经存在"))
		}
	}

	// 邮箱重复
	if request.Email != "" {
		existsId, err := m.Exists(ctx, "email", request.Email)
		if err != nil {
			return err
		}
		if existsId > 0 && existsId != request.Id {
			return common.NewServiceError(http.StatusBadRequest, response.Fail(response.CodeBadRequest).WithMessage("邮箱已经存在"))
		}
	}

	// 更新参数
	var updateMap = make(map[string]any)
	if request.NickName != "" {
		updateMap["nick_name"] = request.NickName
	}
	if request.Account != "" {
		updateMap["account"] = request.Account
	}
	if request.Email != "" {
		updateMap["email"] = request.Email
	}
	if request.Password != "" {
		password, err := bcrypt.GenerateFromPassword([]byte(request.Password), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		updateMap["password"] = password
	}
	if request.Enabled != nil {
		updateMap["enabled"] = *request.Enabled
	}
	if request.Avatar != "" {
		updateMap["avatar"] = request.Avatar
	}

	if len(updateMap) == 0 {
		return nil
	}

	// 执行更新
	result := db.Session(ctx).
		Table(model.Member{}.TableName()).
		Where("id = ?", request.Id).
		UpdateColumns(updateMap)

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return common.NewServiceError(http.StatusBadRequest, response.Fail(response.CodeBadRequest).WithMessage("会员信息更新失败"))
	}
	return nil
}

// Delete 批量删除会员
func (m *MemberService) Delete(ctx context.Context, request *api.MemberDeleteRequest) error {
	for _, v := range request.Id {
		if err := m.deleteById(ctx, v); err != nil {
			return err
		}
	}
	return nil
}

// deleteById 根据 ID 删除会员
func (m *MemberService) deleteById(ctx context.Context, memberId int64) error {

	// 查询完整会员信息
	member, err := gorm.G[model.Member](db.Session(ctx)).Where("id = ?", memberId).Take(ctx)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return common.NewServiceError(http.StatusBadRequest, response.Fail(response.CodeBadRequest).WithMessage("会员记录已删除"))
	}

	// 删除记录
	affectedRows, err := gorm.G[model.Member](db.Session(ctx)).Where("id = ?", memberId).Delete(ctx)
	if err != nil {
		return err
	}
	if affectedRows == 0 {
		return common.NewServiceError(http.StatusBadRequest, response.Fail(response.CodeBadRequest).WithMessage("会员记录删除失败"))
	}
	// 添加删除会员的队列
	return gorm.G[model.MemberDeletedQueue](db.Session(ctx)).Create(ctx, &model.MemberDeletedQueue{
		Id:          id.Next().Int64(),
		MemberId:    member.Id,
		Avatar:      member.Avatar,
		Account:     member.Account,
		Password:    member.Password,
		Email:       member.Email,
		Enabled:     member.Enabled,
		CreateTime:  member.CreateTime,
		UpdateTime:  member.UpdateTime,
		DeletedTime: time.Now().UnixMilli(),
	})
}

// UpdatePassword 修改密码
func (m *MemberService) UpdatePassword(ctx context.Context, request *api.MemberPasswordUpdateRequest) error {

	session := db.Session(ctx)

	var oldPassword string
	if err := session.Raw("SELECT password FROM t_member WHERE id = ?", request.MemberId).Scan(&oldPassword).Error; err != nil {
		return err
	}
	if oldPassword == "" {
		return common.NewServiceError(http.StatusBadRequest, response.Fail(response.CodeBadRequest).WithMessage("用户信息错误"))
	}

	if err := bcrypt.CompareHashAndPassword([]byte(oldPassword), []byte(request.OldPassword)); err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return common.NewServiceError(http.StatusBadRequest, response.Fail(response.CodeBadRequest).WithMessage("旧密码错误"))
		}
		return err
	}
	password, err := bcrypt.GenerateFromPassword([]byte(request.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	r := session.Exec("UPDATE t_member SET password = ?, update_time = ? WHERE id = ?",
		string(password),
		util.ContextValueDefault(ctx, constant.CtxKeyRequestTime, time.Now().UnixMilli()),
		request.MemberId,
	)
	if r.Error != nil {
		return r.Error
	}
	if r.RowsAffected < 1 {
		return common.NewServiceError(http.StatusBadRequest, response.Fail(response.CodeBadRequest).WithMessage("密码更新失败"))
	}
	return nil
}

// Profile 检索会员的基本信息
func (m *MemberService) Profile(ctx context.Context, memberId int64) (*api.MemberProfileResponse, error) {
	var ret api.MemberProfileResponse
	return &ret, db.Session(ctx).Raw("SELECT id, nick_name, avatar, account, email FROM t_member WHERE id = ?", memberId).
		Row().Scan(&ret.Id, &ret.NickName, &ret.Avatar, &ret.Account, &ret.Email)
}

// UpdateProfile 更新会员的基本信息
func (m *MemberService) UpdateProfile(ctx context.Context, request *api.MemberProfileUpdateRequest) error {
	return m.Update(ctx, &api.MemberUpdateRequest{
		Id:       request.MemberId,
		Avatar:   request.Avatar,
		NickName: request.NickName,
		Account:  request.Account,
		Password: "",
		Email:    request.Email,
		Enabled:  nil,
	})
}

var DefaultMemberService = NewMemberService(DefaultResourceService)
