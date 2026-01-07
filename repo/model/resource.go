package model

// DefaultResourceParentId 默认的顶层节点 ID
const DefaultResourceParentId int64 = 0

// DefaultResourceObjectId 默认的资源 ID
const DefaultResourceObjectId int64 = 0

// ResourcePathSeparator 资源 Path 分割符
const ResourcePathSeparator string = ","

// Resource 会员资源
type Resource struct {
	Id         int64  `gorm:"primaryKey"`              // 资源 ID
	MemberId   int64  `gorm:"index:idx_member_object"` // 会员 ID
	ObjectId   int64  `gorm:"index:idx_member_object"` // 物理资源 ID，如果是目录则为 0
	ParentId   int64  // 父级资源 ID
	Title      string // 资源标题
	Dir        bool   // 是否是目录
	Path       string `gorm:"index"` // ID 关系树
	Depth      uint64 // 树深度
	CreateTime int64  // 创建时间
	UpdateTime int64  // 更新时间
}

func (Resource) TableName() string {
	return "t_resource"
}
