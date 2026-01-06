package model

type SysConfigKey string

var (
	// SysConfigKeySessionJwtKey 会话签名密钥
	SysConfigKeySessionJwtKey = SysConfigKey("session_jwt_key")
)

type SysConfigValueType string

var (
	SysConfigValueTypeString   = SysConfigValueType("string")
	SysConfigValueTypeNumber   = SysConfigValueType("number")
	SysConfigValueTypeDatetime = SysConfigValueType("datetime")
	SysConfigValueTypeBoolean  = SysConfigValueType("boolean")
	SysConfigValueTypeJson     = SysConfigValueType("json")
)

type SysConfig struct {
	Id         int64        `gorm:"primaryKey;autoIncrement"`
	Key        SysConfigKey `gorm:"uniqueIndex"`
	Value      string
	ValueType  SysConfigValueType
	Remark     string
	CreateTime int64
	UpdateTime int64
}

func (s *SysConfig) TableName() string {
	return "t_sys_config"
}
