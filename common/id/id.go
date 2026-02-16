package id

import (
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/bwmarrin/snowflake"
)

var snowflakeNode *snowflake.Node

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

// UUID 生成无符号的 UUID
func UUID() string {
	return strings.ReplaceAll(uuid.New().String(), "-", "")
}

const (
	xorKey      uint64 = 0x5A3FE791B2C4D068
	base62Chars        = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
)

// PathOfId 基于 ID 生成 ~11 个字符的路径，唯一、不可穷举
// Gen By Claude
func PathOfId(i int64) string {
	return base62Encode(scramble(uint64(i)))
}

func scramble(n uint64) uint64 {
	n ^= xorKey
	n = (n >> 32) | (n << 32)
	n = ((n & 0xFFFF0000FFFF0000) >> 16) | ((n & 0x0000FFFF0000FFFF) << 16)
	n = ((n & 0xFF00FF00FF00FF00) >> 8) | ((n & 0x00FF00FF00FF00FF) << 8)
	return n
}

func base62Encode(n uint64) string {
	if n == 0 {
		return string(base62Chars[0])
	}
	buf := make([]byte, 0, 11)
	for n > 0 {
		buf = append(buf, base62Chars[n%62])
		n /= 62
	}
	for i, j := 0, len(buf)-1; i < j; i, j = i+1, j-1 {
		buf[i], buf[j] = buf[j], buf[i]
	}
	return string(buf)
}
