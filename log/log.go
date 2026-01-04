package log

import (
	"fmt"
	"ispace/config"
	"log/slog"
	"os"
	"path/filepath"

	"gopkg.in/natefinch/lumberjack.v2"
)

// Level 全局日志级别
var Level = &slog.LevelVar{}

// Initialization 初始化日志级别
func Initialization() func() {

	Level.Set(slog.LevelDebug) // 默认 Debug

	var handlers []slog.Handler

	// Std 控制台
	handlers = append(handlers, slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		AddSource:   false,
		ReplaceAttr: dateTimeFormatter,
	}))

	// 文件日志
	logFile := newLogFile(filepath.Join(*config.LogDir, "app.log"))
	handlers = append(handlers, slog.NewJSONHandler(logFile, &slog.HandlerOptions{
		AddSource:   false,
		ReplaceAttr: dateTimeFormatter,
	}))

	// 异常日志
	errorFile := newLogFile(filepath.Join(*config.LogDir, "error.log"))

	// 全局日志
	slog.SetDefault(slog.New(newAppLogHandler(Level,
		slog.NewJSONHandler(errorFile, &slog.HandlerOptions{
			AddSource:   false,
			ReplaceAttr: dateTimeFormatter,
		}),
		handlers...),
	))

	return func() {
		if logFile != nil {
			if err := logFile.Close(); err != nil {
				_, _ = fmt.Fprintf(os.Stderr, "close log file error: %s", err.Error())
			}
		}
		if err := errorFile.Close(); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "close error log file error: %s", err.Error())
		}
	}
}

func newLogFile(filename string) *lumberjack.Logger {
	logger := &lumberjack.Logger{
		Filename:   filename, // 日志文件
		MaxSize:    3,        // Mb
		MaxAge:     10,       // 最多保留天数
		MaxBackups: 300,      // 最多保留备份数量
		LocalTime:  true,     // 使用本地时间
		Compress:   false,    // 压缩文件
	}
	return logger
}

var formatter = "2006-01-02 15:04:05.000"

// dateTimeFormatter 日期格式化，精确到毫秒
func dateTimeFormatter(_ []string, a slog.Attr) slog.Attr {
	if a.Value.Kind() == slog.KindTime {
		return slog.String(a.Key, a.Value.Time().Format(formatter))
	}
	return a
}
