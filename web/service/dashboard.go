package service

import (
	"context"
	"ispace/common/constant"
	"ispace/common/util"
	"ispace/db"
	"ispace/web/handler/api"
	"time"
)

type DashboardService struct{}

func (s *DashboardService) Stat(ctx context.Context) (*api.DashboardStatResponse, error) {
	var ret = &api.DashboardStatResponse{}
	var err error

	// 对象
	ret.Object, err = s.ObjectStat(ctx)
	if err != nil {
		return nil, err
	}

	// 会员
	ret.Member, err = s.MemberStat(ctx)
	return ret, err
}

// MemberStat 会员统计
func (s *DashboardService) MemberStat(ctx context.Context) (*api.DashboardMemberStat, error) {

	var ret = &api.DashboardMemberStat{}

	session := db.Session(ctx)

	// 总统计
	if err := session.Raw("SELECT COUNT(id) FROM t_member").
		Row().
		Scan(&ret.Total); err != nil {
		return nil, err
	}

	// 上下文中的时区信息
	now := util.ContextValue[time.Time](ctx, constant.CtxKeyRequestTime)
	timeZone := util.ContextValue[*time.Location](ctx, constant.CtxKeyTimezone)

	// 当前时间在客户端的时区
	now = now.In(timeZone)

	// 统计最近30天的数据
	// TODO 替换为 group
	for i := range 30 {

		day := now.AddDate(0, 0, -i)

		var dailyStat = api.MemberDailyStat{
			Date: day.Format(time.DateOnly),
		}

		// 此日开始和结束
		dayStart := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, day.Location()).UnixMilli()
		dayEnd := time.Date(day.Year(), day.Month(), day.Day(), 23, 59, 59, 999999999, day.Location()).UnixMilli()

		// 此日的统计
		err := session.
			Raw("SELECT COUNT(id) FROM t_member WHERE create_time BETWEEN ? AND ?", dayStart, dayEnd).
			Row().
			Scan(&dailyStat.Total)
		if err != nil {
			return nil, err
		}
		ret.Daily = append(ret.Daily, &dailyStat)
	}

	return ret, nil
}

// ObjectStat 对象统计
func (s *DashboardService) ObjectStat(ctx context.Context) (*api.DashboardObjectStat, error) {

	var ret = &api.DashboardObjectStat{}

	session := db.Session(ctx)

	// 总统计
	if err := session.Raw("SELECT IFNULL(SUM(size), 0), IFNULL(SUM(file_size), 0), COUNT(id) FROM t_object").
		Row().
		Scan(&ret.Size, &ret.FileSize, &ret.Total); err != nil {
		return nil, err
	}

	// 上下文中的时区信息
	now := util.ContextValue[time.Time](ctx, constant.CtxKeyRequestTime)
	timeZone := util.ContextValue[*time.Location](ctx, constant.CtxKeyTimezone)

	// 当前时间在客户端的时区
	now = now.In(timeZone)

	// 统计最近30天的数据
	// TODO 替换为 group
	for i := range 30 {

		day := now.AddDate(0, 0, -i)

		var dailyStat = api.ObjectDailyStat{
			Date: day.Format(time.DateOnly),
		}

		// 此日开始和结束
		dayStart := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, day.Location()).UnixMilli()
		dayEnd := time.Date(day.Year(), day.Month(), day.Day(), 23, 59, 59, 999999999, day.Location()).UnixMilli()

		// 此日的统计
		err := session.
			Raw("SELECT IFNULL(SUM(size), 0), IFNULL(SUM(file_size), 0), COUNT(id) FROM t_object WHERE create_time BETWEEN ? AND ?",
				dayStart, dayEnd,
			).
			Row().
			Scan(&dailyStat.Size, &dailyStat.FileSize, &dailyStat.Total)
		if err != nil {
			return nil, err
		}
		ret.Daily = append(ret.Daily, &dailyStat)
	}

	return ret, nil
}

func NewDashboardService() *DashboardService {
	return &DashboardService{}
}

var DefaultDashboardService = NewDashboardService()
