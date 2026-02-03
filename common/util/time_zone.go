package util

import (
	"fmt"
	"time"
)

// TimeZoneOffset 根据传入的 t 和 time.Location, 计算 loc 和 UTC 的偏移字符串
// 例如 +08:00
func TimeZoneOffset(t time.Time, loc *time.Location) string {
	if loc == nil {
		return "+00:00"
	}
	// 使用当前时间点在该时区的偏移量
	// 注意：如果涉及历史数据且有夏令时切换，建议传入具体的时间点进行计算
	_, offsetSeconds := t.In(loc).Zone()

	absSeconds := offsetSeconds
	sign := "+"
	if offsetSeconds < 0 {
		sign = "-"
		absSeconds = -offsetSeconds
	}

	hours := absSeconds / 3600
	minutes := (absSeconds % 3600) / 60

	// 格式必须为 [+-]HH:MM
	return fmt.Sprintf("%s%02d:%02d", sign, hours, minutes)
}
