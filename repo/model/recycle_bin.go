package model

// RecycleBin 回收站
type RecycleBin struct {
	Id         int64 `gorm:"primaryKey"` // 回收站记录 ID
	MemberId   int64 `gorm:"index"`      // 会员 ID
	Root       bool  // 是否是根记录
	CreateTime int64 // 进入回收站的时间

	// 资源快照
	ResourceId          int64  `gorm:"index"` // 资源对象 ID
	ResourceObjectId    int64  // 资源引用对象 ID
	ResourceParentId    int64  `gorm:"index"` // 父级资源 ID
	ResourceTitle       string // 资源标题
	ResourceContentType string // 媒体类型
	ResourceDir         bool   // 是否是目录
	ResourcePath        string // 路径
	ResourceDepth       uint64 // 深度
	ResourceCreateTime  int64  // 资源创建时间
}

func (RecycleBin) TableName() string {
	return "t_recycle_bin"
}

/*

## 删除
	* 如果删除的是目录，则把整个目录都加入到回收站中
	* 把根目录设置为 root

## 回收
	* 根据 ResourceParentId 检索父级记录是否存在，以及父级目录是否移动过位置（path 是否匹配）
	* 父目录不存在，则直接移动到根目录，资源 ID 保持不变（mac 会提示 xxx 目录已经不存在）
	* 如果是目录，则按照关系层次结构，移动整个树结构

## 查询
	* 只展示 root 记录，可以查询子项目。但是不能只恢复目录中的子项目（mac 好像是这样的设计）




*/
