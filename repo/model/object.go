package model

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
	Id          int64             `gorm:"primaryKey"`
	Path        string            `gorm:"uniqueIndex"` // 资源在本地的存储路径
	Compression ObjectCompression // 压缩算法
	Hash        string            `gorm:"index"` // Sha256 值
	Size        int64             // 原始文件大小
	RefCount    uint64            // 引用数量
	ContentType string            // 媒体类型
	CreateTime  int64             // 创建时间
	UpdateTime  int64             // 更新时间
}

func (Object) TableName() string {
	return "t_object"
}
