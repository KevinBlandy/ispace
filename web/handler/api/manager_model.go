package api

import (
	"ispace/common/page"
	"ispace/common/types"
	"ispace/repo/model"
)

// ManagerSignInRequest 管理员登录
type ManagerSignInRequest struct {
	Account  string `json:"account"`
	Password string `json:"password"`
}

// MemberListRequest 会员列表查询
type MemberListRequest struct {
	Pager   *page.Pager `json:"-"`
	Account string      `json:"account"` // 账户
	Email   string      `json:"email"`   // 邮箱
}

// MemberListResponse 会员列表响应
type MemberListResponse struct {
	Id           int64  `json:"id,string"`
	NickName     string `json:"nickName"`
	Avatar       string `json:"avatar"`
	Account      string `json:"account"`
	Email        string `json:"email"`
	Enabled      bool   `json:"enabled"`
	CreateTime   int64  `json:"createTime,string"`
	UpdateTime   int64  `json:"updateTime,string"`
	Resources    int64  `json:"resources"`           // 资源数量
	ResourceSize int64  `json:"resourceSize,string"` // 资源大小
}

// MemberCreateRequest 创建会员
type MemberCreateRequest struct {
	NickName string `json:"nickName"` // 昵称
	Account  string `json:"account"`  // 账户
	Password string `json:"password"` // 密码
	Email    string `json:"email"`    // 邮箱
	Enabled  bool   `json:"enabled"`  // 状态
}

// MemberUpdateRequest 会员更新
type MemberUpdateRequest struct {
	Id int64 `json:"-"`
	// MemberCreateRequest
	Avatar   string `json:"avatar"`
	NickName string `json:"nickName"`
	Account  string `json:"account"`  // 账户
	Password string `json:"password"` // 密码
	Email    string `json:"email"`    // 邮箱
	Enabled  *bool  `json:"enabled"`  // 状态
}

// MemberDeleteRequest 会员删除
type MemberDeleteRequest struct {
	Id types.Int64Slice `json:"id"`
}

// ObjectListRequest 存储对象列表查询
type ObjectListRequest struct {
	Pager  *page.Pager `json:"-"`
	Status string      `json:"status"` // 状态
}

// ObjectListResponse 存储对象列表响应
type ObjectListResponse struct {
	Id          int64                   `json:"id,string"`
	Path        string                  `json:"path"`              // 资源在本地的存储路径
	Compression model.ObjectCompression `json:"compression"`       // 压缩算法
	Hash        string                  `json:"hash"`              // Sha256 值
	Size        int64                   `json:"size"`              // 原始文件大小
	FileSize    int64                   `json:"fileSize"`          // 实际文件大小
	RefCount    uint64                  `json:"refCount"`          // 引用数量
	ContentType string                  `json:"contentType"`       // 媒体类型
	Status      model.ObjectStatus      `json:"status"`            // 对象状态
	CreateTime  int64                   `json:"createTime,string"` // 创建时间
	UpdateTime  int64                   `json:"updateTime,string"` // 更新时间
}

// ObjectUpdateRequest 更新对象请求
type ObjectUpdateRequest struct {
	Id     int64              `json:"-"`
	Status model.ObjectStatus `json:"status"`
}

// ObjectDeleteRequest 对象删除请求
type ObjectDeleteRequest struct {
	Id types.Int64Slice `json:"id"`
}

// AdminPasswordUpdateRequest  密码修改
type AdminPasswordUpdateRequest struct {
	AdminId     int64  `json:"-"`
	OldPassword string `json:"oldPassword"` // 旧密码
	NewPassword string `json:"newPassword"` // 新密码
}

// AdminProfileResponse 个人信息
type AdminProfileResponse struct {
	Id       int64  `json:"id,string"`
	NickName string `json:"nickName"`
	Account  string `json:"account"`
	Email    string `json:"email"`
	Avatar   string `json:"avatar"`
}

// AdminProfileUpdateRequest 个人信息修改
type AdminProfileUpdateRequest struct {
	AdminId  int64  `json:"-"`
	NickName string `json:"nickName"`
	Account  string `json:"account"`
	Email    string `json:"email"`
	Avatar   string `json:"avatar"`
}

// ObjectStatResponse 资源统计
type ObjectStatResponse struct {
	Total    int64              `json:"total"`    // 总资源文件数量
	Size     uint64             `json:"size"`     // 总资源逻辑大小
	FileSize int64              `json:"fileSize"` // 实际占用空间大小
	Daily    []*ObjectDailyStat `json:"daily"`    // 每日统计
}

// ObjectDailyStat 每日统计
type ObjectDailyStat struct {
	Date     string `json:"date"`     // 日期
	Total    uint64 `json:"total"`    // 上传文件数量
	Size     uint64 `json:"size"`     // 上传文件大小
	FileSize uint64 `json:"fileSize"` // 实际文件大小
}
