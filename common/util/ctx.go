package util

import "context"

// ContextValue 获取上下文中的数据
func ContextValue[T any](ctx context.Context, key string) T {
	t, _ := ctx.Value(key).(T)
	return t
}

// ContextValueDefault 获取上下文中的数据，可以设置默认值
func ContextValueDefault[T any](ctx context.Context, key string, defaultValue T) T {
	t, ok := ctx.Value(key).(T)
	if ok {
		return t
	}
	return defaultValue
}
