package id

import (
	"strings"
	"sync/atomic"
	"time"

	"github.com/google/uuid"

	"github.com/bwmarrin/snowflake"
)

var snowflakeNode *snowflake.Node

var atomicId = &atomic.Uint64{}

func init() {

	var err error

	// 纪元 2025-01-01 00:00:00.0
	snowflake.Epoch = time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC).UnixMilli()

	snowflakeNode, err = snowflake.NewNode(0) // 节点编号

	if err != nil {
		panic(err)
	}
}

// Next 生成下一个雪花ID
func Next() snowflake.ID {
	return snowflakeNode.Generate()
}

// Increment 全局递增 ID，从 0 开始
func Increment() uint64 {
	return atomicId.Add(1)
}

// UUID 生成无符号的 UUID
func UUID() string {
	return strings.ReplaceAll(uuid.New().String(), "-", "")
}
