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

// MemberObject 会员资源
type MemberObject struct {
	Id         int64  `gorm:"primaryKey;autoIncrement"` // 资源 ID
	MemberId   int64  // 会员 ID
	ObjectId   int64  // 物理资源 ID，如果是目录则为 0
	ParentId   int64  // 父级ID
	Title      string // 资源标题
	Dir        bool   // 是否是目录
	Path       string // ID 关系树
	Depth      uint64 // 树深度
	DeleteTime int64  // 删除时间
	CreateTime int64  // 创建时间
	UpdateTime int64  // 更新时间
}

func (MemberObject) TableName() string {
	return "t_member_object"
}

type MemberConfig struct {
	MemberId     int64  `gorm:"primaryKey"` // 会员id
	MaxStorage   uint64 // 最大存储空间，字节
	MaxTraffic   uint64 // 最多流量
	MaxBandwidth uint64 // 最高带宽
}
