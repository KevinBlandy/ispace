package types

import (
	"encoding/json"
	"fmt"
	"strconv"
)

// Int64Slice []int64 类型，处理 json 的序列化和反序列化，一律用字符串代替 int64，避免 js 精度丢失
type Int64Slice []int64

func (s Int64Slice) Len() int {
	return len(s)
}

func (s *Int64Slice) UnmarshalJSON(data []byte) error {
	var raw []any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	res := make([]int64, len(raw))
	for i, v := range raw {
		switch x := v.(type) {
		case string:
			n, err := strconv.ParseInt(x, 10, 64)
			if err != nil {
				return err
			}
			res[i] = n
		case float64: // JSON number 默认是 float64
			res[i] = int64(x)
		default:
			return fmt.Errorf("invalid type %T", v)
		}
	}

	*s = res
	return nil
}

func (s Int64Slice) MarshalJSON() ([]byte, error) {
	tmp := make([]string, len(s))
	for i, v := range s {
		tmp[i] = strconv.FormatInt(v, 10)
	}
	return json.Marshal(tmp)
}
