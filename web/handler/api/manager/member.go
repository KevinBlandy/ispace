package manager

import (
	"context"
	"ispace/common"
	"ispace/common/page"
	"ispace/common/response"
	"ispace/db"
	"ispace/web/handler/api"
	"ispace/web/service"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type MemberApi struct {
	memberService *service.MemberService
}

func NewMemberApi(memberService *service.MemberService) *MemberApi {
	return &MemberApi{memberService: memberService}
}

// List 列表
func (m *MemberApi) List(g *gin.Context) (any, error) {

	var request = new(api.MemberListRequest)
	request.Pager = page.NewPagerFromQuery(g.Request.URL.Query())
	request.Account = g.Query("account")
	request.Email = g.Query("email")

	// 排序
	request.Pager.Sort = []page.Sort{{Field: "create_time", Order: "DESC"}}

	result, err := db.Transaction(g.Request.Context(), func(ctx context.Context) (*page.Pagination[*api.MemberListResponse], error) {
		return m.memberService.List(ctx, request)
	}, db.TxReadOnly)
	if err != nil {
		return nil, err
	}
	return response.Ok(result), nil
}

// Create 创建
func (m *MemberApi) Create(g *gin.Context) (any, error) {
	var request = new(api.MemberCreateRequest)
	if err := g.ShouldBindJSON(request); err != nil {
		return nil, err
	}
	err := db.TransactionWithOutResult(g.Request.Context(), func(ctx context.Context) error {
		return m.memberService.Create(ctx, request)
	})
	if err != nil {
		return nil, err
	}
	g.AbortWithStatusJSON(http.StatusCreated, response.Ok(nil))
	return nil, nil
}

// Update 更新
func (m *MemberApi) Update(g *gin.Context) (any, error) {
	var request = new(api.MemberUpdateRequest)
	if err := g.ShouldBindJSON(request); err != nil {
		return nil, err
	}
	if request.Id, _ = strconv.ParseInt(g.Param("id"), 10, 64); request.Id < 1 {
		return nil, common.NewServiceError(http.StatusBadRequest, response.Fail(response.CodeBadRequest).WithMessage("非法请求"))
	}

	err := db.TransactionWithOutResult(g.Request.Context(), func(ctx context.Context) error {
		return m.memberService.Update(ctx, request)
	})
	if err != nil {
		return nil, err
	}
	return response.Ok(nil), nil
}

// Delete 删除
func (m *MemberApi) Delete(g *gin.Context) (any, error) {
	var request = new(api.MemberDeleteRequest)
	if err := g.ShouldBindJSON(request); err != nil {
		return nil, err
	}
	err := db.TransactionWithOutResult(g.Request.Context(), func(ctx context.Context) error {
		return m.memberService.Delete(ctx, request)
	})
	if err != nil {
		return nil, err
	}
	return response.Ok(nil), nil
}

var DefaultMemberApi = NewMemberApi(service.DefaultMemberService)
