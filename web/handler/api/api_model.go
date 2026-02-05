package api

import (
	"ispace/common/page"
	"ispace/common/types"
	"ispace/repo/model"
	"time"
)

// MemberSignInRequest 登录
type MemberSignInRequest struct {
	Account  string `json:"account"`
	Password string `json:"password"`
}

// ResourceListRequest 资源列表
type ResourceListRequest struct {
	MemberId int64
	ParentId int64
	//Keywords string
	//Dir      *bool
}

// ResourceTreeResponse 完整的资源树
type ResourceTreeResponse struct {
	ResourceListResponse
	Entries []*ResourceTreeResponse `json:"entries"` // 子项目
}

// ResourceListResponse 资源列表
type ResourceListResponse struct {
	Id          int64              `json:"id,string"`
	ParentId    int64              `json:"parentId,string"`   // 父级资源 ID
	Title       string             `json:"title"`             // 资源标题
	ContentType string             `json:"contentType"`       // 媒体类型
	Dir         bool               `json:"dir"`               // 是否是目录
	Size        int64              `json:"size,string"`       // 文件大小
	Status      model.ObjectStatus `json:"status"`            // 文件状态
	CreateTime  int64              `json:"createTime,string"` // 创建时间
	UpdateTime  int64              `json:"updateTime,string"` // 更新时间
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

// ObjectHashResponse 对象的 Hash 查询结果
type ObjectHashResponse struct {
	Hit bool `json:"hit"` // 是否命中
}

// ResourceFlashUploadRequest ⚡️传请求
type ResourceFlashUploadRequest struct {
	Title string `json:"title"` // 资源标题
	Hash  string `json:"hash"`  // 资源 Hash
}

// ResourceSearchRequest 资源搜索请求
type ResourceSearchRequest struct {
	Pager    *page.Pager
	MemberId int64
	Keywords string
}

// ResourceSearchResponse 资源搜索结果
type ResourceSearchResponse struct {
	Id          int64              `json:"id,string"`
	Title       string             `json:"title"`             // 资源标题
	ContentType string             `json:"contentType"`       // 类型
	Size        int64              `json:"size,string"`       // 文件大小
	Status      model.ObjectStatus `json:"status"`            // 文件状态
	CreateTime  int64              `json:"createTime,string"` // 创建时间
	UpdateTime  int64              `json:"updateTime,string"` // 更新时间
}

// ResourceRecentRequest 最近资源
type ResourceRecentRequest struct {
	Pager       *page.Pager
	MemberId    int64
	ContentType string // 文件类型 eg image/video/audio/text
}

// ResourceRecentResponse 最近资源
type ResourceRecentResponse struct {
	Id          int64              `json:"id,string"`
	Title       string             `json:"title"`             // 资源标题
	ContentType string             `json:"contentType"`       // 文件类型
	Size        int64              `json:"size,string"`       // 文件大小
	Status      model.ObjectStatus `json:"status"`            // 文件状态
	CreateTime  int64              `json:"createTime,string"` // 创建时间
	UpdateTime  int64              `json:"updateTime,string"` // 更新时间
}

// ResourceGroupRequest 资源分组请求
type ResourceGroupRequest struct {
	Pager       *page.Pager
	MemberId    int64
	Group       string         // 分组类型 day/week/month/year
	ContentType string         // 资源类型
	TimeZone    *time.Location // 客户端时区
}

// ResourceGroupItem 资源项目
type ResourceGroupItem struct {
	Id          int64              `json:"id,string"`
	Title       string             `json:"title"`             // 资源标题
	ContentType string             `json:"contentType"`       // 文件类型
	Size        int64              `json:"size,string"`       // 文件大小
	Status      model.ObjectStatus `json:"status"`            // 文件状态
	CreateTime  int64              `json:"createTime,string"` // 创建时间
	UpdateTime  int64              `json:"updateTime,string"` // 更新时间
}

// ResourceGroupResponse 资源分组响应
type ResourceGroupResponse struct {
	Group string               `json:"group"` // 组别
	Items []*ResourceGroupItem `json:"items"` // 项目列表
}

// MemberPasswordUpdateRequest  密码修改
type MemberPasswordUpdateRequest struct {
	MemberId    int64  `json:"-"`
	OldPassword string `json:"oldPassword"` // 旧密码
	NewPassword string `json:"newPassword"` // 新密码
}

// MemberProfileResponse 个人信息
type MemberProfileResponse struct {
	Id       int64  `json:"id,string"`
	NickName string `json:"nickName"`
	Account  string `json:"account"`
	Email    string `json:"email"`
	Avatar   string `json:"avatar"`
}

// MemberProfileUpdateRequest 个人信息修改
type MemberProfileUpdateRequest struct {
	MemberId int64  `json:"-"`
	NickName string `json:"nickName"`
	Account  string `json:"account"`
	Email    string `json:"email"`
	Avatar   string `json:"avatar"`
}
