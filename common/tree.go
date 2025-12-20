package common

type ObjectTree struct {
	Id         int64         // Id
	Title      string        // 标题
	Dir        bool          // 是否是目录
	CreateTime int64         // 创建时间
	Entry      []*ObjectTree // 子项目
}
