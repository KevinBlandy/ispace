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

// Redis Redis 链接地址
var Redis = flag.String("redis", "redis://@localhost:6379/0", "Redis 连接地址")

/*
type Redis struct {
	Network        string        `yaml:"network"`
	Address        string        `yaml:"address"`
	Password       string        `yaml:"password"`
	Db             int           `yaml:"db"`
	ConnectTimeout time.Duration `yaml:"connect-timeout"`
	ReadTimeout    time.Duration `yaml:"read-timeout"`
	WriteTimeout   time.Duration `yaml:"write-timeout"`
	Pool           *RedisPool    `yaml:"pool"`
}

type RedisPool struct {
	MaxOpenConn int           `yaml:"max-open-conn"`
	MinIdleConn int           `yaml:"min-idle-conn"`
	MaxIdleConn int           `yaml:"max-idle-conn"`
	MaxIdleTime time.Duration `yaml:"max-idle-time"`
	MaxLifetime time.Duration `yaml:"max-life-time"`
	Timeout     time.Duration `yaml:"timeout"`
}

*/
