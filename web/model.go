package web

// SignInApiRequest 登录
type SignInApiRequest struct {
	Account  string `json:"account"`
	Password string `json:"password"`
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
