package manager

import (
	"context"
	"ispace/common/response"
	"ispace/db"
	"ispace/web/handler/api"
	"ispace/web/service"

	"github.com/gin-gonic/gin"
)

type DashboardApi struct {
	service *service.DashboardService
}

func NewDashboardApi(dashboardService *service.DashboardService) *DashboardApi {
	return &DashboardApi{dashboardService}
}

func (d *DashboardApi) Stat(g *gin.Context) (any, error) {
	ret, err := db.Transaction(g.Request.Context(), func(ctx context.Context) (*api.DashboardStatResponse, error) {
		return d.service.Stat(ctx)
	})
	if err != nil {
		return nil, err
	}
	return response.Ok(ret), nil
}

var DefaultDashboardApi = NewDashboardApi(service.DefaultDashboardService)
