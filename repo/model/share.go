package model

// Share 分享设置
type Share struct {
	Id         int64  `gorm:"primaryKey"`  // ID
	MemberId   int64  `gorm:"index"`       // 会员 ID
	Path       string `gorm:"uniqueIndex"` // 唯一 URL
	Password   string // 访问密码
	Views      int64  // 访问次数
	CreateTime int64  // 分享日期
	ExpireTime int64  // 过期时间
}

func (Share) TableName() string {
	return "t_share"
}

type ShareResource struct {
	Id               int64  `gorm:"primaryKey"` // 记录 ID
	ShareId          int64  `gorm:"index"`      // 分享 ID
	ResourceId       int64  `gorm:"index"`      // 资源 ID
	ResourceParentId int64  // 资源父级 ID
	ResourcePath     string // 资源 ID 关系树
	ResourceTitle    string // 资源标题
	ResourceDir      bool   // 资源是否是目录
	ResourceDepth    uint64 // 资源树深度
	CreateTime       int64  // 创建时间
	UpdateTime       int64  // 更新时间
}

func (ShareResource) TableName() string {
	return "t_share_resource"
}
