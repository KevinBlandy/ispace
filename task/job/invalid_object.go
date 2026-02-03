package job

import (
	"context"
	"ispace/web/service"
	"log/slog"
)

// InvalidObjectCleaner 的的
type InvalidObjectCleaner struct {
	service *service.ObjectService
}

func (i InvalidObjectCleaner) Run() {
	if err := i.service.InvalidClean(context.Background()); err != nil {
		slog.Error("失效对象清理执行异常", slog.String("err", err.Error()))
	}
}

func NewInvalidObjectCleaner(service *service.ObjectService) *InvalidObjectCleaner {
	return &InvalidObjectCleaner{service: service}
}
