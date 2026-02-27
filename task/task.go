package task

import (
	"ispace/task/job"
	"ispace/web/service"

	"github.com/robfig/cron/v3"
)

// Initialization 初始化定时任务
func Initialization() (func(), error) {

	scheduler := cron.New(
		cron.WithSeconds(), // 解析秒
		cron.WithChain(cron.SkipIfStillRunning(cron.DiscardLogger)), // 当前任务执行时间超过了间隔时间，当前任务执行完毕后，等待间隔时间后，执行下次任务。
	)

	// 每小时执行一次失效文件清理
	if _, err := scheduler.AddJob("0 1/1 * * * ? ", job.NewInvalidObjectCleaner(service.DefaultObjectService)); err != nil {
		return func() {}, nil
	}
	//  每分钟执行一次回收站过期文件清理
	if _, err := scheduler.AddJob("0 1/1 * * * ? ", job.NewRecycleBinCleaner(service.DefaultRecycleBinService)); err != nil {
		return func() {}, nil
	}
	//  每分钟执行一次分享过期文件清理
	if _, err := scheduler.AddJob("0 1/1 * * * ? ", job.NewShareCleaner(service.DefaultShareService)); err != nil {
		return func() {}, nil
	}

	scheduler.Start()
	return func() {
		<-scheduler.Stop().Done()
	}, nil
}
