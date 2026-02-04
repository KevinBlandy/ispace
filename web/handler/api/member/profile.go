package member

import (
	"ispace/web/service"

	"github.com/gin-gonic/gin"
)

type ProfileApi struct {
	m *service.MemberService
}

func NewProfileApi(m *service.MemberService) *ProfileApi {
	return &ProfileApi{m: m}
}

func (p *ProfileApi) Profile(g *gin.Context) (any, error) {
	return nil, nil
}

var DefaultProfileApi = NewProfileApi(service.DefaultMemberService)
