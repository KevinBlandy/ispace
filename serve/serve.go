package serve

import (
	"context"
	"errors"
	"ispace/common/constant"
	"ispace/config"
	"ispace/db"
	"ispace/log"
	"ispace/task"
	"ispace/web/server"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	_ "ispace/config"

	"github.com/gin-gonic/gin"
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
	taskStop, err := task.Initialization()
	if err != nil {
		return
	}

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

	taskStop()

	_ = db.Close()
	//_ = rdb.Close()

	slog.Info("服务已停止")
}
