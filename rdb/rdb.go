package rdb

import (
	"context"
	"ispace/config"
	"time"

	"github.com/redis/go-redis/v9"
)

var client *redis.Client

func Initialization() error {

	option := &redis.Options{
		Network:      *config.RedisNetWork,
		Addr:         *config.RedisAddress,
		Password:     *config.RedisPassword,
		DB:           *config.RedisDB,
		DialTimeout:  *config.RedisConnectTimeout,
		ReadTimeout:  *config.RedisReadTimeout,
		WriteTimeout: *config.RedisWriteTimeout,
	}

	// 连接池
	option.PoolSize = *config.RedisPoolMaxOpenConn        // 连接池大小
	option.MinIdleConns = *config.RedisPoolMinIdleConn    // 最小空闲连接数，受PoolSize限制
	option.MaxIdleConns = *config.RedisPoolMaxIdleConn    // 最大空闲连接数，多余会被关闭
	option.ConnMaxIdleTime = *config.RedisPoolMaxIdleTime // 每个连接最大空闲时间，如果超过了这个时间会被关闭
	option.ConnMaxLifetime = *config.RedisPoolMaxLifetime // 连接最大生命周期
	option.PoolTimeout = *config.RedisPoolTimeout         // 从连接池获取连接超时时间（如果所有连接都繁忙，等待的时间）

	client = redis.NewClient(option)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	return client.Ping(ctx).Err()
}

// Execute 执行 Redis 任务
func Execute[T any](fn func(conn *redis.Conn) (T, error)) (T, error) {
	conn := client.Conn()
	defer func() {
		_ = conn.Close()
	}()
	return fn(conn)
}

func ExecuteClient[T any](client *redis.Client, fn func(conn *redis.Conn) (T, error)) (T, error) {
	conn := client.Conn()
	defer func() {
		_ = conn.Close()
	}()
	return fn(conn)
}

func Get() *redis.Client {
	return client
}

func Close() error {
	return client.Close()
}

func Stats() *redis.PoolStats {
	return client.PoolStats()
}
