package service

import (
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"ispace/common"
	"ispace/common/response"
	"ispace/common/types"
	"ispace/common/util"
	"ispace/db"
	"ispace/repo/model"
	"mime"
	"mime/multipart"
	"net/http"
	"path/filepath"

	"gorm.io/gorm"
)

// sniffLen 最多读取字节数，来判断 contentType
const sniffLen = 512

// compressionThreshold 压缩阈值
var compressionThreshold = int64(types.KB)

type ResourceService struct{}

func NewResourceService() *ResourceService {
	return &ResourceService{}
}

// NewObjectRef 创建新的资源引用
func (s *ResourceService) NewObjectRef(ctx context.Context, memberId, parentId, objectId int64, title string) error {
	return nil
}

// Upload 上传文件到磁盘
func (s *ResourceService) Upload(ctx context.Context, memberId int64, parentId int64, fileHeader *multipart.FileHeader) error {
	if fileHeader.Size == 0 {
		return common.NewServiceError(http.StatusBadRequest, response.Fail(response.CodeBadRequest).WithMessage("不能上传空文件"))
	}
	file, err := fileHeader.Open()
	if err != nil {
		return err
	}
	defer util.SafeClose(file)

	// 计算文件的 Sha256
	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return err
	}
	hash := hex.EncodeToString(hasher.Sum(nil))

	// 查询 Hash 是否存在
	objectId, err := db.Transaction(ctx, func(ctx context.Context) (int64, error) {
		var id int64
		return id, db.Session(ctx).Table(model.Object{}.TableName()).Select("id").Where("sha256 = ?", hash).Scan(&id).Error
	})
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		// 已存在了文件，复制引用即可
		return s.NewObjectRef(ctx, memberId, parentId, objectId, fileHeader.Filename)
	}

	// 重置指针
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return err
	}

	// 查询媒体类型
	contentType := mime.TypeByExtension(filepath.Ext(fileHeader.Filename))
	if contentType == "" {
		// 没扩展名，则尝试用魔术值
		var buf [sniffLen]byte
		n, _ := io.ReadFull(file, buf[:])
		contentType = http.DetectContentType(buf[:n])
		// 重置指针
		if _, err := file.Seek(0, io.SeekStart); err != nil {
			return err
		}
	}

	var reader io.ReadCloser = file

	// 压缩判断
	var compress = fileHeader.Size > compressionThreshold
	if compress {
		var err error
		reader, err = gzip.NewReader(reader)
		if err != nil {
			return err
		}
		defer util.SafeClose(reader)
	}

	// TODO 原子式 IO 到磁盘

	return nil
}

var DefaultResourceService = NewResourceService()
