package model

// Manager 系统管理员
type Manager struct {
	Id         int64  `gorm:"primaryKey;autoIncrement"`
	Account    string // 账户
	Password   string // 密码
	CreateTime int64  // 创建时间
	UpdateTime int64  // 更新时间
}

func (Manager) TableName() string {
	return "t_manager"
}
