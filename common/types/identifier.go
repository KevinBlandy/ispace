package types

import (
	"regexp"
	"strconv"
)

// Identifier ID 标识
type Identifier string

// Numeric 是否为 Number 类型的 ID
func (i Identifier) Numeric() bool {
	return regexp.MustCompile("^[1-9]\\d*").MatchString(string(i))
}

// String 字符串类型的 ID
func (i Identifier) String() string {
	return string(i)
}

// Int64 int 类型的 ID
func (i Identifier) Int64() (ret int64) {
	ret, _ = strconv.ParseInt(string(i), 10, 64)
	return
}
