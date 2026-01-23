package service

import (
	"context"
	"errors"
	"ispace/common"
	"ispace/common/response"
	"ispace/db"
	"ispace/repo/model"
	"ispace/web/handler/api"
	"net/http"

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

func NewAdminService() *AdminService {
	return &AdminService{}
}

var DefaultAdminService = NewAdminService()
