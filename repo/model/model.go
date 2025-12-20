package model

type UserRole string

var (
	// UserRoleAdmin 管理员
	UserRoleAdmin UserRole = "admin"
	// UserRoleMember 会员
	UserRoleMember UserRole = "member"
)

// User 管理员
type User struct {
	Id         int64    `gorm:"primaryKey;autoIncrement"`
	Account    string   // 账户
	Password   string   // 密码
	Email      string   // 邮箱
	Role       UserRole // 角色
	Enabled    bool     // 是否启用账户
	CreateTime int64    // 创建时间
	UpdateTime int64    // 更新时间
}

func (User) TableName() string {
	return "t_user"
}

type ObjectCompression string

var (
	// ObjectCompressionNone 未压缩
	ObjectCompressionNone ObjectCompression = "none"
	// ObjectCompressionGzip Gzip 压缩
	ObjectCompressionGzip ObjectCompression = "gzip"
	// ObjectCompressionBrotli brotli 压缩
	ObjectCompressionBrotli ObjectCompression = "brotli"
)

// Object 对象
type Object struct {
	Id          int64             `gorm:"primaryKey;autoIncrement"`
	Path        string            // 资源在本地的存储路径
	Compression ObjectCompression // 压缩算法
	Hash        string            // Sha256 值
	Size        uint64            // 原始文件大小
	RefCount    uint64            // 引用数量
	ContentType string            // 媒体类型
	CreateTime  int64             // 创建时间
	UpdateTime  int64             // 更新时间
}

func (Object) TableName() string {
	return "t_object"
}

// UserObject 会员资源
type UserObject struct {
	Id         int64  `gorm:"primaryKey;autoIncrement"` // 资源 ID
	UserId     int64  // 会员 ID
	ObjectId   int64  // 物理资源 ID，如果是目录则为 0
	ParentId   int64  // 父级ID
	Title      string // 资源标题
	Dir        bool   // 是否是目录
	Path       string // ID 关系树
	Depth      uint64 // 树深度
	DeleteTime int64  // 删除时间
	CreateTime int64  // 创建时间
	UpdateTime int64  // 更新时间
}

func (UserObject) TableName() string {
	return "t_user_object"
}

type RecycleBin struct {
	Id           int64 `gorm:"primaryKey;autoIncrement"`
	UserObjectId int64 // 会员资源 ID
	CreateTime   int64 // 创建时间
}
