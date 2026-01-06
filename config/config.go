package config

import (
	"flag"
	"path/filepath"
	"time"
)

// LogDir 日志输出目录
var LogDir = flag.String("log.dir", "logs", "日志输出目录")

// PublicDir 公共资源目录
var PublicDir = flag.String("public.dir", "public", "公共资源目录")

// HTTP 服务配置
var (
	HttpPort = flag.Int("http.port", 8689, "Http server port")
	HttpHost = flag.String("http.host", "0.0.0.0", "Http server host")
)

// DB 数据库
var (
	DB                = flag.String("db", filepath.Join("database", "db"), "Sqlite 数据库文件")
	DBPoolMaxIdleConn = flag.Int("db.pool.max-idle-conn", 5, "DB 连接池最多空闲连接")
	DBPoolMaxOpenConn = flag.Int("db.pool.max-open-conn", 20, "DB 连接池最多打开连接")
	DBPoolMaxIdleTime = flag.Duration("db.pool.max-idle-time", time.Minute*5, "DB 连接的最大空闲时间")
	DBPoolMaxLifetime = flag.Duration("db.pool.max-life-time", time.Hour, "DB 连接的最大存活时间")
)

// Redis 配置
var (
	RedisNetWork        = flag.String("redis.network", "tcp", "Redis 连接网络")
	RedisAddress        = flag.String("redis.address", "127.0.0.1:6379", "Redis 主机")
	RedisPassword       = flag.String("redis.password", "", "Redis 密码")
	RedisDB             = flag.Int("redis.db", 0, "Redis 数据库")
	RedisConnectTimeout = flag.Duration("redis.connect-timeout", time.Second, "Redis 连接超时时间")
	RedisReadTimeout    = flag.Duration("redis.read-timeout", time.Second*2, "Redis 读超时时间")
	RedisWriteTimeout   = flag.Duration("redis.write-timeout", time.Second*2, "Redis 写超时时间")

	RedisPoolMaxOpenConn = flag.Int("redis.pool.max-open-conn", 10, "Redis 连接池最多连接数量")
	RedisPoolMinIdleConn = flag.Int("redis.pool.min-idle-conn", 5, "Redis 连接池最小活跃连接数量")
	RedisPoolMaxIdleConn = flag.Int("redis.pool.max-idle-conn", 2, "Redis 连接池最多空闲连接数量")
	RedisPoolMaxIdleTime = flag.Duration("redis.pool.max-idle-time", time.Minute*5, "Redis 连接最大空闲时间")
	RedisPoolMaxLifetime = flag.Duration("redis.pool.max-life-time", time.Minute*5, "Redis 连接最大存活时间")
	RedisPoolTimeout     = flag.Duration("redis.pool.timeout", time.Second*5, "Redis 获取连接超时时间")
)
