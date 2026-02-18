package model

// Share 分享设置
type Share struct {
	Id         int64  `gorm:"primaryKey"`  // ID
	MemberId   int64  `gorm:"index"`       // 会员 ID
	Path       string `gorm:"uniqueIndex"` // 唯一 URL
	Enabled    bool   // 是否启用
	Password   string // 访问密码
	Views      int64  // 访问次数，每次打开算一次
	CreateTime int64  // 分享日期
	UpdateTime int64  // 更新日期
	ExpireTime int64  // 分享过期时间，0 表示永不过期
}

func (Share) TableName() string {
	return "t_share"
}

// ShareResource 分享的资源详情
// 直接复制完整的资源树
type ShareResource struct {
	Id      int64 `gorm:"primaryKey"` // 记录 ID
	ShareId int64 `gorm:"index"`      // 分享 ID
	Root    bool  // 是否是根路径

	// 资源快照
	ResourceId          int64  `gorm:"index"` // 资源 ID
	ResourceParentId    int64  `gorm:"index"` // 资源父级 ID
	ResourceObjectId    int64  // 资源引用对象 ID
	ResourcePath        string // 资源 ID 关系树
	ResourceTitle       string // 资源标题
	ResourceDir         bool   // 资源是否是目录
	ResourceContentType string // 资源类型
	ResourceDepth       uint64 // 资源树深度
}

func (ShareResource) TableName() string {
	return "t_share_resource"
}
