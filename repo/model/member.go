package model

// Member 会员
type Member struct {
	Id         int64 `gorm:"primaryKey"`
	NickName   string
	Avatar     string
	Account    string `gorm:"uniqueIndex"`
	Password   string
	Email      string `gorm:"uniqueIndex"`
	Enabled    bool
	CreateTime int64
	UpdateTime int64
	//DeleteTime int64 // 逻辑删除
	//Version    int64 // 版本控制
}

func (Member) TableName() string {
	return "t_member"
}

// MemberDeletedQueue 会员删除队列
type MemberDeletedQueue struct {
	Id          int64 `gorm:"primaryKey"`
	MemberId    int64 `gorm:"uniqueIndex"`
	Avatar      string
	Account     string
	Password    string
	Email       string
	Enabled     bool
	CreateTime  int64
	UpdateTime  int64
	DeletedTime int64 // 删除时间
}

func (MemberDeletedQueue) TableName() string {
	return "t_member_deleted_queue"
}
