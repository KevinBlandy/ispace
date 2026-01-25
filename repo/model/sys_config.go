package model

import (
	"encoding/json"
	"strconv"
	"time"
)

type SysConfigKey string

var (
	// SysConfigKeySessionSecret 会话签名密钥
	SysConfigKeySessionSecret = SysConfigKey("session_secret")
	// SysConfigKeySessionExpire 回话过期时间
	SysConfigKeySessionExpire = SysConfigKey("session_expire")
)

// SysConfigValueType 配置值类型
type SysConfigValueType string

var (
	SysConfigValueTypeString   = SysConfigValueType("string")   // 字符串
	SysConfigValueTypeNumber   = SysConfigValueType("number")   // 数值
	SysConfigValueTypeDatetime = SysConfigValueType("datetime") // 日期
	SysConfigValueTypeDuration = SysConfigValueType("duration") // 时长
	SysConfigValueTypeSwitch   = SysConfigValueType("switch")   // 开关
	SysConfigValueTypeJson     = SysConfigValueType("json")     // json 配置
)

// SysConfig 系统配置
type SysConfig struct {
	Id         int64              `gorm:"primaryKey;autoIncrement"`
	Key        SysConfigKey       `gorm:"uniqueIndex"` // 配置 Key
	Value      string             // 配置 Value
	ValueType  SysConfigValueType // 配置值类型
	Remark     string             // 备注
	CreateTime int64              // 创建时间
	UpdateTime int64              // 更新时间
}

func (s *SysConfig) BoolValue() (ret bool) {
	ret, _ = strconv.ParseBool(s.Value)
	return
}
func (s *SysConfig) StringValue() (ret string) {
	ret = s.Value
	return
}
func (s *SysConfig) DurationValue() (ret time.Duration) {
	ret, _ = time.ParseDuration(s.Value)
	return
}

func (s *SysConfig) Int64Value() (ret int64) {
	ret, _ = strconv.ParseInt(s.Value, 10, 64)
	return
}

func (s *SysConfig) Uint64Value() (ret uint64) {
	ret, _ = strconv.ParseUint(s.Value, 10, 64)
	return
}

func (s *SysConfig) JsonValue(dest any) (err error) {
	return json.Unmarshal([]byte(s.Value), dest)
}

func (s *SysConfig) TableName() string {
	return "t_sys_config"
}
