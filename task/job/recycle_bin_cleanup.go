package job

import (
	"context"
	"ispace/db"
	"ispace/web/service"
	"log/slog"
	"runtime/debug"
)

type RecycleBinCleaner struct {
	service *service.RecycleBinService
}

func NewRecycleBinCleaner(binService *service.RecycleBinService) *RecycleBinCleaner {
	return &RecycleBinCleaner{binService}
}

func (r *RecycleBinCleaner) Run() {
	defer func() {
		if err := recover(); err != nil {
			slog.Error("回收站清理执行崩溃",
				slog.Any("panic", err),
				slog.String("stack", string(debug.Stack())),
			)
		}
	}()
	s, err := db.Transaction(context.Background(), func(ctx context.Context) (int64, error) {
		return r.service.Clean(ctx)
	})
	if err != nil {
		slog.Error("回收站清理执行异常", slog.String("err", err.Error()))
		return
	}

	if s > 0 {
		slog.Info("回收站清理完毕", slog.Int64("清理数量", s))
	}
}
