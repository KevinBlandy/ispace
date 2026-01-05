package config

import (
	"flag"
	"path/filepath"
)

// HttpPort Http 服务端口
var HttpPort = flag.Int("http.port", 8689, "Http server port")

// HttpHost Http 服务地址
var HttpHost = flag.String("http.host", "0.0.0.0", "Http server host")

// DB 数据库地址
var DB = flag.String("db", filepath.Join("database", "db"), "Sqlite 数据库文件")

// LogDir 日志输出目录
var LogDir = flag.String("log.dir", "logs", "日志输出目录")

// PublicDir 公共资源目录
var PublicDir = flag.String("public.dir", "public", "公共资源目录")
