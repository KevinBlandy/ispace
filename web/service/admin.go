package service

import (
	"context"
	"errors"
	"ispace/common"
	"ispace/common/constant"
	"ispace/common/response"
	"ispace/common/util"
	"ispace/db"
	"ispace/repo/model"
	"ispace/web/handler/api"
	"net/http"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AdminService struct {
}

// Login 登录
func (a AdminService) Login(ctx context.Context, request *api.ManagerSignInRequest) (*model.Admin, error) {
	admin, err := gorm.G[model.Admin](db.Session(ctx)).
		Select("id", "password", "enabled").
		Where("account = ?", request.Account).
		Take(ctx)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, common.NewServiceError(http.StatusBadRequest, response.Fail(response.CodeBadRequest).WithMessage("用户名或密码错误"))
		}
		return nil, err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(admin.Password), []byte(request.Password)); err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return nil, common.NewServiceError(http.StatusBadRequest, response.Fail(response.CodeBadRequest).WithMessage("用户名或密码错误"))
		}
		return nil, err
	}
	if !admin.Enabled {
		return nil, common.NewServiceError(http.StatusForbidden, response.Fail(response.CodeForbidden).WithMessage("账户被封禁"))
	}
	return &admin, nil
}

// Exists 根据 column = value 检索 ID
func (a AdminService) Exists(ctx context.Context, column string, value any) (ret int64, err error) {
	err = db.Session(ctx).Table(model.Admin{}.TableName()).Select("id").Where(column+" = ?", value).Scan(&ret).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		err = nil
	}
	return
}

func (a AdminService) Profile(ctx context.Context, adminId int64) (*api.AdminProfileResponse, error) {
	var ret api.AdminProfileResponse
	return &ret, db.Session(ctx).Raw("SELECT id, nick_name, avatar, account, email FROM t_admin WHERE id = ?", adminId).
		Row().Scan(&ret.Id, &ret.NickName, &ret.Avatar, &ret.Account, &ret.Email)
}

func (a AdminService) UpdateProfile(ctx context.Context, request *api.AdminProfileUpdateRequest) error {
	// 账户重复
	if request.Account != "" {
		existsId, err := a.Exists(ctx, "account", request.Account)
		if err != nil {
			return err
		}
		if existsId > 0 && existsId != request.AdminId {
			return common.NewServiceError(http.StatusBadRequest, response.Fail(response.CodeBadRequest).WithMessage("账户已经存在"))
		}
	}

	// 邮箱重复
	if request.Email != "" {
		existsId, err := a.Exists(ctx, "email", request.Email)
		if err != nil {
			return err
		}
		if existsId > 0 && existsId != request.AdminId {
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
	if request.Avatar != "" {
		updateMap["avatar"] = request.Avatar
	}

	if len(updateMap) == 0 {
		return nil
	}

	// 执行更新
	result := db.Session(ctx).
		Table(model.Admin{}.TableName()).
		Where("id = ?", request.AdminId).
		UpdateColumns(updateMap)

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return common.NewServiceError(http.StatusBadRequest, response.Fail(response.CodeBadRequest).WithMessage("会员信息更新失败"))
	}
	return nil
}

func (a AdminService) UpdatePassword(ctx context.Context, request *api.AdminPasswordUpdateRequest) error {
	session := db.Session(ctx)

	var oldPassword string
	if err := session.Raw("SELECT password FROM t_admin WHERE id = ?", request.AdminId).Scan(&oldPassword).Error; err != nil {
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

	r := session.Exec("UPDATE t_admin SET password = ?, update_time = ? WHERE id = ?",
		string(password),
		util.ContextValueDefault(ctx, constant.CtxKeyRequestTime, time.Now().UnixMilli()),
		request.AdminId,
	)
	if r.Error != nil {
		return r.Error
	}
	if r.RowsAffected < 1 {
		return common.NewServiceError(http.StatusBadRequest, response.Fail(response.CodeBadRequest).WithMessage("密码更新失败"))
	}
	return nil
}

func NewAdminService() *AdminService {
	return &AdminService{}
}

var DefaultAdminService = NewAdminService()
