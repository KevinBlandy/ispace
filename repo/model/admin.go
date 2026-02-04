package model

// Admin 系统管理员
type Admin struct {
	Id         int64 `gorm:"primaryKey"`
	NickName   string
	Avatar     string
	Account    string `gorm:"uniqueIndex"`
	Password   string
	Email      string `gorm:"uniqueIndex"`
	Enabled    bool
	CreateTime int64
	UpdateTime int64
}

func (Admin) TableName() string {
	return "t_admin"
}
