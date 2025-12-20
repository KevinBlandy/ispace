package common

// ObjectTree 通用的对象树
type ObjectTree struct {
	Id         int64         `json:"id"`          // Id
	ParentId   int64         `json:"parentId"`    // 父级 ID
	Title      string        `json:"title"`       // 标题
	Dir        bool          `json:"dir"`         // 是否是目录
	Size       uint64        `json:"size"`        // 大小
	CreateTime int64         `json:"create_time"` // 创建时间
	Entry      []*ObjectTree `json:"entry"`       // 子项目
}

// NewObjectTrees 解析为树形结构
func NewObjectTrees(entries ...*ObjectTree) []*ObjectTree {

	if len(entries) == 0 {
		return make([]*ObjectTree, 0)
	}

	// 转换为 id map
	var objectMap = make(map[int64]*ObjectTree)
	for _, entry := range entries {
		objectMap[entry.Id] = entry
	}

	var rootNodes = make([]*ObjectTree, 0)

	// 先找出 root 节点，即 parentId 不存在于列表中
	for _, obj := range objectMap {
		_, parentNode := objectMap[obj.ParentId]
		if !parentNode {
			rootNodes = append(rootNodes, obj)
		}
	}

	// 在 subNodes 中找出 node 的直接子节点
	var fn func(*ObjectTree, map[int64]*ObjectTree)
	fn = func(node *ObjectTree, subNodes map[int64]*ObjectTree) {
		for _, obj := range subNodes {
			if obj.ParentId == node.Id {
				node.Entry = append(node.Entry, obj)
			}
		}
		if len(node.Entry) > 0 {
			// 递归
			for _, obj := range node.Entry {
				fn(obj, subNodes)
			}
		}
	}

	// 存在子节点，组装完整的树结构
	for _, obj := range rootNodes {
		fn(obj, objectMap)
	}
	return rootNodes
}
