package util

import (
	"io"
	"ispace/common/types"
	"maps"
	"net/url"
	"slices"
	"strconv"
)

func Require[T any](fn func() (T, error)) T {
	ret, err := fn()
	if err != nil {
		panic(err)
	}
	return ret
}

func If[T any](condition bool, trueVal, falseVal T) T {
	if condition {
		return trueVal
	}
	return falseVal
}

func SafeClose(closer io.Closer) {
	if closer == nil {
		return
	}
	_ = closer.Close()
}

func P[T any](value T) *T {
	return &value
}

// BoolQuery 解析 bool 类型的查询参数
func BoolQuery(query url.Values, name string) *bool {
	values, ok := query[name]
	if !ok {
		return nil // 没传
	}
	if len(values) == 0 {
		return nil // 传的空
	}

	var value = values[0]

	var boolVal bool

	if value == "" {
		boolVal = true
	} else {
		boolVal, _ = strconv.ParseBool(value)
	}
	return &boolVal
}

// Int64SliceQuery int64 查询参数，去重
func Int64SliceQuery(value []string) (types.Int64Slice, error) {
	m := make(map[int64]struct{}, len(value))
	for _, v := range value {
		int64Val, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return nil, err // 解码失败
		}
		m[int64Val] = struct{}{}
	}
	return slices.Collect(maps.Keys(m)), nil
}
