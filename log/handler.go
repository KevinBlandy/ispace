package log

import (
	"context"
	"errors"
	"ispace/common/constant"
	"log/slog"
)

// appLogHandler 组合 Handler
// 非线程安全
type appLogHandler struct {
	handlers    []slog.Handler // 整合的所有 handler
	errorHandle slog.Handler   // 异常日志的 handler
	level       slog.Leveler   // 日志级别
}

func newAppLogHandler(level slog.Leveler, errorHandler slog.Handler, handlers ...slog.Handler) *appLogHandler {
	return &appLogHandler{handlers: handlers, errorHandle: errorHandler, level: level}
}

func (c *appLogHandler) Enabled(_ context.Context, level slog.Level) bool {
	return c.level.Level() <= level.Level()

}

func (c *appLogHandler) Handle(ctx context.Context, record slog.Record) error {

	var errSlice []error

	// 请求 id
	requestId := ctx.Value(constant.CtxKeyRequestId)
	if requestId != nil {
		record.AddAttrs(slog.Any("request", requestId))
	}

	// logger 名称
	loggerName := ctx.Value(constant.CtxKeyLoggerName)
	if loggerName != nil {
		record.AddAttrs(slog.Any("logger", loggerName))
	}

	for _, v := range c.handlers {
		if err := v.Handle(ctx, record); err != nil {
			errSlice = append(errSlice, err)
		}
	}

	/*
		异常日志需要单独处理
	*/
	if record.Level == slog.LevelError {
		if err := c.errorHandle.Handle(ctx, record); err != nil {
			errSlice = append(errSlice, err)
		}
	}

	if len(errSlice) > 0 {
		return errors.Join(errSlice...)
	}
	return nil
}
func (c *appLogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	var handlers []slog.Handler
	for _, v := range c.handlers {
		handlers = append(handlers, v.WithAttrs(attrs))
	}
	r := newAppLogHandler(c.level, c.errorHandle.WithAttrs(attrs), handlers...)
	*c = *r
	return c
}

func (c *appLogHandler) WithGroup(name string) slog.Handler {
	var handlers []slog.Handler
	for _, v := range c.handlers {
		handlers = append(handlers, v.WithGroup(name))
	}

	r := newAppLogHandler(c.level, c.errorHandle.WithGroup(name), handlers...)
	*c = *r
	return c
}
