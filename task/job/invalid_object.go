package job

import (
	"context"
	"ispace/web/service"
	"log/slog"
	"runtime/debug"
)

// InvalidObjectCleaner 的的
type InvalidObjectCleaner struct {
	service *service.ObjectService
}

func (i InvalidObjectCleaner) Run() {
	defer func() {
		if err := recover(); err != nil {
			slog.Error("失效对象清理执行崩溃",
				slog.Any("panic", err),
				slog.String("stack", string(debug.Stack())),
			)
		}
	}()
	if err := i.service.InvalidClean(context.Background()); err != nil {
		slog.Error("失效对象清理执行异常", slog.String("err", err.Error()))
	}
}

func NewInvalidObjectCleaner(service *service.ObjectService) *InvalidObjectCleaner {
	return &InvalidObjectCleaner{service: service}
}
