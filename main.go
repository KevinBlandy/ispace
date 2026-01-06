package main

import (
	"context"
	"errors"
	"ispace/common/constant"
	"ispace/config"
	"ispace/db"
	"ispace/log"
	"ispace/rdb"
	"ispace/repo"
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

func init() {
	_ = config.Initialization(os.Args[1:])
	gin.SetMode(gin.ReleaseMode)
}

func main() {

	shutdown := log.Initialization()
	defer shutdown()

	// ===================================
	// 系统资源初始化
	// ===================================
	if err := db.Initialization(); err != nil {
		slog.Error("数据源初始化异常", slog.String("error", err.Error()))
		return
	}
	if err := repo.Initialization(); err != nil {
		slog.ErrorContext(context.Background(), "数据表初始化异常", slog.String("error", err.Error()))
		return
	}
	if err := rdb.Initialization(); err != nil {
		slog.ErrorContext(context.Background(), "Redis 初始化异常", slog.String("error", err.Error()))
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

	_ = db.Close()
	_ = rdb.Close()

	slog.Info("服务已停止")
}
