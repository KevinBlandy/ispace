package model

// Member 会员
type Member struct {
	Id         int64  `gorm:"primaryKey;autoIncrement"`
	Avatar     string // 头像
	Account    string // 账户
	Password   string // 密码
	Email      string // 邮箱
	Enabled    bool   // 是否启用账户
	CreateTime int64  // 创建时间
	UpdateTime int64  // 更新时间
}

func (Member) TableName() string {
	return "t_member"
}

//type MemberConfig struct {
//	MemberId     int64  `gorm:"primaryKey"` // 会员id
//	MaxStorage   uint64 // 最大存储空间，字节
//	MaxTraffic   uint64 // 最多流量
//	MaxBandwidth uint64 // 最高带宽
//}
