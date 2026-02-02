package model

// ObjectCompression 对象的压缩方式，使用 Content-Encoding 值
type ObjectCompression string

// ContentEncoding 对应的编码类型
// https://developer.mozilla.org/en-US/docs/Web/HTTP/Reference/Headers/Content-Encoding
func (o ObjectCompression) ContentEncoding() string {
	switch o {
	case ObjectCompressionGzip:
		return "gzip"
	case ObjectCompressionBrotli:
		return "br"
	default:
		return ""
	}
}

var (
	// ObjectCompressionNone 未压缩
	ObjectCompressionNone ObjectCompression = "none"
	// ObjectCompressionGzip Gzip 压缩
	ObjectCompressionGzip ObjectCompression = "gzip"
	// ObjectCompressionBrotli Brotli 压缩
	ObjectCompressionBrotli ObjectCompression = "br"
	// ObjectCompressionUnknow 未知
	ObjectCompressionUnknow ObjectCompression = ""
)

type ObjectStatus string

var (
	// ObjectStatusOk 状态 OK
	ObjectStatusOk ObjectStatus = "OK"
	// ObjectStatusPendingReview 待审核
	ObjectStatusPendingReview ObjectStatus = "PENDING_REVIEW"
	// ObjectStatusDisabled 禁用
	ObjectStatusDisabled ObjectStatus = "DISABLED"
)

// Object 对象
type Object struct {
	Id          int64             `gorm:"primaryKey"`
	Path        string            `gorm:"uniqueIndex"` // 资源在本地的存储路径
	Compression ObjectCompression // 压缩算法
	Hash        string            `gorm:"index"` // Sha256 值
	Size        int64             // 原始文件大小
	FileSize    int64             // 实际文件大小
	RefCount    uint64            // 引用数量
	ContentType string            // 媒体类型
	Status      ObjectStatus      // 对象状态
	CreateTime  int64             // 创建时间
	UpdateTime  int64             // 更新时间
}

func (Object) TableName() string {
	return "t_object"
}
