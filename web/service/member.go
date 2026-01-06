package service

import (
	"context"
	"errors"
	"ispace/common"
	"ispace/common/response"
	"ispace/db"
	"ispace/repo/model"
	"ispace/web"
	"net/http"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type MemberService struct {
}

func NewMemberService() *MemberService {
	return &MemberService{}
}

// Login 登录
func (m MemberService) Login(ctx context.Context, request *web.SignInApiRequest) (*model.Member, error) {

	member, err := gorm.G[*model.Member](db.Session(ctx)).Where("account = ?", request.Account).Take(ctx)

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
	return member, nil
}

var DefaultMemberService = NewMemberService()
