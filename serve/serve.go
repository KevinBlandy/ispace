package serve

import (
	"context"
	"errors"
	"ispace/common/constant"
	"ispace/config"
	"ispace/db"
	"ispace/log"
	"ispace/rdb"
	"ispace/task/job"
	"ispace/web/server"
	"ispace/web/service"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	_ "ispace/config"

	"github.com/gin-gonic/gin"
	"github.com/robfig/cron/v3"
)

func Serve() {

	_ = config.Initialization(os.Args[1:])
	gin.SetMode(gin.ReleaseMode) //

	// 日志初始化
	shutdown := log.Initialization()
	defer shutdown()

	// ===================================
	// 系统资源初始化
	// ===================================
	if err := initialization(); err != nil {
		slog.Error("系统初始化异常",
			slog.String("err", err.Error()),
		)
		return
	}

	// ===================================
	// 定时任务调度
	// ===================================

	scheduler := cron.New(
		cron.WithSeconds(), // 解析秒
		cron.WithChain(cron.SkipIfStillRunning(cron.DiscardLogger)), // 当前任务执行时间超过了间隔时间，当前任务执行完毕后，等待间隔时间后，执行下次任务。
	)

	// 每小时执行一次失效文件清理
	if _, err := scheduler.AddJob("0 0 1/1 * * ? ", job.NewInvalidObjectCleaner(service.DefaultObjectService)); err != nil {
		slog.Error("Schedule 调度任务初始化异常",
			slog.String("err", err.Error()),
		)
		return
	}

	scheduler.Start()

	// ===================================
	// HTTP 服务
	// ===================================
	httpServer := server.New()

	go func() {
		ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
		defer cancel()
		<-ctx.Done()

		slog.Info("正在停止服务...")

		if err := httpServer.Shutdown(context.Background()); err != nil {
			slog.Error("服务停止异常", slog.String("error", err.Error()))
		}
	}()

	slog.Info("服务启动",
		slog.String("http", httpServer.Addr),
		slog.String("os", runtime.GOOS+"/"+runtime.GOARCH),
		slog.String("go", runtime.Version()),
		slog.String("workDir", constant.WorkDir),
	)

	if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("服务启动异常", slog.String("error", err.Error()))
		return
	}

	ctx := scheduler.Stop()

	_ = db.Close()
	_ = rdb.Close()

	<-ctx.Done()

	slog.Info("服务已停止")
}
