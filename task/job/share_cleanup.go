package job

import (
	"context"
	"ispace/db"
	"ispace/web/service"
	"log/slog"
	"runtime/debug"
)

type ShareCleaner struct {
	service *service.ShareService
}

func NewShareCleaner(binService *service.ShareService) *ShareCleaner {
	return &ShareCleaner{service: binService}
}

func (s *ShareCleaner) Run() {
	defer func() {
		if err := recover(); err != nil {
			slog.Error("分享清理执行崩溃",
				slog.Any("panic", err),
				slog.String("stack", string(debug.Stack())),
			)
		}
	}()
	c, err := db.Transaction(context.Background(), func(ctx context.Context) (int64, error) {
		return s.service.Clean(ctx)
	})
	if err != nil {
		slog.Error("分享清理执行异常", slog.String("err", err.Error()))
		return
	}
	if c > 0 {
		slog.Info("分享清理完毕", slog.Int64("清理数量", c))
	}
}
