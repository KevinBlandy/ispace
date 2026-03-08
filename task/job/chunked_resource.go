package job

import (
	"context"
	"ispace/web/service"
	"log/slog"
	"runtime/debug"
)

type ChunkedResourceCleaner struct {
	service *service.ResourceChunkService
}

func NewChunkedResourceCleaner(service *service.ResourceChunkService) *ChunkedResourceCleaner {
	return &ChunkedResourceCleaner{
		service: service,
	}
}

func (c *ChunkedResourceCleaner) Run() {
	defer func() {
		if err := recover(); err != nil {
			slog.Error("无效分的片文件清理执行崩溃",
				slog.Any("panic", err),
				slog.String("stack", string(debug.Stack())),
			)
		}
	}()
	if err := c.service.InvalidClean(context.Background()); err != nil {
		slog.Error("无效分的片文件清理执行异常", slog.String("err", err.Error()))
	}
}
