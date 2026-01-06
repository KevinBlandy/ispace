package config

import (
	"flag"
	"path/filepath"
	"time"
)

// 全局配置

var LogDir *string
var PublicDir *string
var StoreDir *string

var HttpPort *int
var HttpHost *string

var DB *string
var DBPoolMaxIdleConn *int
var DBPoolMaxOpenConn *int
var DBPoolMaxIdleTime *time.Duration
var DBPoolMaxLifetime *time.Duration

var RedisNetWork *string
var RedisAddress *string
var RedisPassword *string
var RedisDB *int
var RedisConnectTimeout *time.Duration
var RedisReadTimeout *time.Duration
var RedisWriteTimeout *time.Duration

var RedisPoolMaxOpenConn *int
var RedisPoolMinIdleConn *int
var RedisPoolMaxIdleConn *int
var RedisPoolMaxIdleTime *time.Duration
var RedisPoolMaxLifetime *time.Duration
var RedisPoolTimeout *time.Duration

func Initialization(args []string) error {

	var flagSet = flag.NewFlagSet("Server 命令行参数", flag.PanicOnError)

	// ==========================
	// 本地目录
	// ==========================
	LogDir = flagSet.String("log.dir", "logs", "日志输出目录")
	PublicDir = flagSet.String("public.dir", "public", "公共资源目录")
	StoreDir = flagSet.String("store.dir", "store", "文件资源存储目录")

	// ==========================
	// HTTP
	// ==========================
	HttpPort = flagSet.Int("http.port", 8689, "Http server port")
	HttpHost = flagSet.String("http.host", "0.0.0.0", "Http server host")

	// ==========================
	// 数据源
	// ==========================
	DB = flagSet.String("db", filepath.Join("database", "db"), "DB 数据库文件")
	DBPoolMaxIdleConn = flagSet.Int("db.pool.max-idle-conn", 5, "DB 连接池最多空闲连接")
	DBPoolMaxOpenConn = flagSet.Int("db.pool.max-open-conn", 20, "DB 连接池最多打开连接")
	DBPoolMaxIdleTime = flagSet.Duration("db.pool.max-idle-time", time.Minute*5, "DB 连接的最大空闲时间")
	DBPoolMaxLifetime = flagSet.Duration("db.pool.max-life-time", time.Hour, "DB 连接的最大存活时间")

	// ==========================
	// Redis
	// ==========================
	RedisNetWork = flagSet.String("redis.network", "tcp", "Redis 连接网络")
	RedisAddress = flagSet.String("redis.address", "127.0.0.1:6379", "Redis 主机")
	RedisPassword = flagSet.String("redis.password", "", "Redis 密码")
	RedisDB = flagSet.Int("redis.db", 0, "Redis 数据库")
	RedisConnectTimeout = flagSet.Duration("redis.connect-timeout", time.Second, "Redis 连接超时时间")
	RedisReadTimeout = flagSet.Duration("redis.read-timeout", time.Second*2, "Redis 读超时时间")
	RedisWriteTimeout = flagSet.Duration("redis.write-timeout", time.Second*2, "Redis 写超时时间")

	RedisPoolMaxOpenConn = flagSet.Int("redis.pool.max-open-conn", 10, "Redis 连接池最多连接数量")
	RedisPoolMinIdleConn = flagSet.Int("redis.pool.min-idle-conn", 5, "Redis 连接池最小活跃连接数量")
	RedisPoolMaxIdleConn = flagSet.Int("redis.pool.max-idle-conn", 2, "Redis 连接池最多空闲连接数量")
	RedisPoolMaxIdleTime = flagSet.Duration("redis.pool.max-idle-time", time.Minute*5, "Redis 连接最大空闲时间")
	RedisPoolMaxLifetime = flagSet.Duration("redis.pool.max-life-time", time.Minute*5, "Redis 连接最大存活时间")
	RedisPoolTimeout = flagSet.Duration("redis.pool.timeout", time.Second*5, "Redis 获取连接超时时间")

	return flagSet.Parse(args)
}
