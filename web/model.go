package web

import "ispace/common/types"

// SignInApiRequest 登录
type SignInApiRequest struct {
	Account  string `json:"account"`
	Password string `json:"password"`
}

type ResourceListApiRequest struct {
	MemberId int64
	ParentId int64
	Dir      *bool
}

// ResourceTreeApiResponse 完整的资源树
type ResourceTreeApiResponse struct {
	ResourceListApiResponse
	Entries []*ResourceTreeApiResponse `json:"entries"` // 子项目
}

// ResourceListApiResponse 资源列表
type ResourceListApiResponse struct {
	Id         int64  `json:"id,string"`
	ParentId   int64  `json:"parentId,string"`   // 父级资源 ID
	Title      string `json:"title"`             // 资源标题
	Dir        bool   `json:"dir"`               // 是否是目录
	Size       int64  `json:"size,string"`       // 文件大小
	CreateTime int64  `json:"createTime,string"` // 创建时间
	UpdateTime int64  `json:"updateTime,string"` // 更新时间
}

// ResourceMkdirRequest 创建文件夹
type ResourceMkdirRequest struct {
	MemberId int64
	ParentId int64  `json:"parentId,string"` // 父级目录
	Title    string `json:"title"`           // 文件夹名称
}

// ResourceRenameRequest 资源重命名请求
type ResourceRenameRequest struct {
	Id       int64
	MemberId int64
	Title    string `json:"title"` // 新的名称
}

// ResourceDeleteRequest 资源删除请求
type ResourceDeleteRequest struct {
	MemberId int64
	Id       types.Int64Slice `json:"id,string"` // 要删除的资源列表
}

// ResourceMoveRequest 移动资源
type ResourceMoveRequest struct {
	MemberId int64
	Id       types.Int64Slice `json:"id,string"`       // From 资源 Id
	ParentId int64            `json:"parentId,string"` // 目标 ID
}
