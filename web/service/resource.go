package service

import (
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"ispace/common"
	"ispace/common/id"
	"ispace/common/response"
	"ispace/common/types"
	"ispace/common/util"
	"ispace/config"
	"ispace/db"
	"ispace/repo"
	"ispace/repo/model"
	"ispace/web"
	"log/slog"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// sniffLen 最多读取字节数，来判断 contentType
const sniffLen = 512

// compressionThreshold 压缩阈值
var compressionThreshold = int64(types.KB)

type ResourceService struct{}

func NewResourceService() *ResourceService {
	return &ResourceService{}
}

// List 查询资源列表
func (s *ResourceService) List(ctx context.Context, request *web.ResourceListApiRequest) ([]*web.ResourceListApiResponse, error) {

	var ret = make([]*web.ResourceListApiResponse, 0)

	params := []any{request.MemberId, request.ParentId}

	statement := &strings.Builder{}
	_, _ = statement.WriteString(`SELECT
				id,
				parent_id,
				title,
				dir,
				create_time,
				update_time,
				(
					CASE dir 
						WHEN 1 THEN 0
						WHEN 0 THEN (SELECT size from t_object WHERE id = object_id)
					END
				) size
			FROM
				t_resource
			WHERE
				member_id = ? 
			AND 
				parent_id = ?`)

	// 条件
	if request.Dir != nil {
		_, _ = statement.WriteString(" AND dir = ?")
		params = append(params, request.Dir)
	}

	// 排序
	_, _ = statement.WriteString(" ORDER BY dir DESC, title ASC")

	session := db.Session(ctx)
	rows, err := session.Raw(statement.String(), params...).Rows()
	if err != nil {
		return nil, err
	}

	defer util.SafeClose(rows)

	for rows.Next() {
		resource := &web.ResourceListApiResponse{}
		if err := session.ScanRows(rows, resource); err != nil {
			return nil, err
		}
		ret = append(ret, resource)
	}

	return ret, nil
}

// Get 获取资源信息
func (s *ResourceService) Get(ctx context.Context, memberId, resourceId int64) (ret struct {
	Title       string                  // 文件标题
	Compression model.ObjectCompression // 压缩方式
	ContentType string                  // 文件类型
	Path        string                  // 相对路径
}, err error) {
	row := db.Session(ctx).Raw(`
			SELECT
				t.title,
				t1.compression,
				t1.content_type,
				t1.path
			FROM
				t_resource t
				INNER JOIN t_object t1 ON t1.id = t.object_id
			WHERE
				t.id = ?
			AND
				t.member_id = ?
			AND
				t.dir = ?
		`, resourceId, memberId, false).Row()
	err = row.Scan(&ret.Title, &ret.Compression, &ret.ContentType, &ret.Path)
	return
}

// NewObjectRef 创建新的资源引用
func (s *ResourceService) NewObjectRef(ctx context.Context, memberId, parentId, objectId int64, title string) error {

	var resourceId = id.Next().Int64()

	var (
		newPath         = strconv.FormatInt(resourceId, 10) + model.ResourcePathSeparator
		newDepth uint64 = 0
	)

	if parentId != model.DefaultResourceParentId {
		dir, err := gorm.G[*model.Resource](db.Session(ctx).Clauses(clause.Locking{Strength: "UPDATE"})). // for update
															Select("id", "dir", "path", "depth").
															Where("id = ? AND member_id = ?", parentId, memberId).
															Take(ctx)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				err = common.NewServiceError(http.StatusBadRequest, response.Fail(response.CodeBadRequest).WithMessage("上传目录不存在"))
			}
			return err
		}

		if !dir.Dir {
			return common.NewServiceError(http.StatusBadRequest, response.Fail(response.CodeBadRequest).WithMessage("上传目录不存在"))
		}

		// 父目录存在，则拼接
		newPath = dir.Path + newPath
		newDepth = dir.Depth + 1
	}

	// 重名处理
	ext := filepath.Ext(title)
	fileName := strings.TrimSuffix(title, ext)

	var counter = 1
	for {
		var existsId int64
		err := gorm.G[model.Resource](db.Session(ctx)).Select("id").
			Where("member_id = ? AND parent_id = ? AND title = ?", memberId, parentId, title).
			Scan(ctx, &existsId)

		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if existsId == 0 {
			break // Ok 没重复
		}

		// 递增序号
		title = fmt.Sprintf("%s(%d)%s", fileName, counter, ext)

		counter++
	}

	now := time.Now().UnixMilli()

	err := gorm.G[model.Resource](db.Session(ctx)).Create(ctx, &model.Resource{
		Id:         resourceId,
		MemberId:   memberId,
		ObjectId:   objectId,
		ParentId:   parentId,
		Title:      title,
		Dir:        false, // 文件
		Path:       newPath,
		Depth:      newDepth,
		CreateTime: now,
		UpdateTime: now,
	})

	if err != nil {
		return err
	}

	// 更新引用
	result := db.Session(ctx).
		Table(model.Object{}.TableName()).
		Where("id = ?", objectId).UpdateColumns(map[string]any{
		"update_time": now,
		"ref_count":   gorm.Expr("ref_count + ?", 1),
	})

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected != 1 {
		return common.NewServiceError(http.StatusBadRequest, response.Fail(response.CodeBadRequest).WithMessage("存储引用更新失败"))
	}
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
	objectId, err := repo.DefaultObjectRepo.GetIdByHash(ctx, hash)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	if objectId > 0 {
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

	// 目录打散 & 随机文件名称
	dir := s.RandDir()
	fileName := id.UUID()

	// 逻辑路径
	absPath := path.Join(path.Join(dir...), fileName)

	// 本地存储的完整路径
	newFilePath := filepath.Join(*config.StoreDir, filepath.FromSlash(absPath))

	// 先尝试创建完整的目录
	if err := os.MkdirAll(filepath.Dir(newFilePath), os.ModePerm); err != nil && !os.IsExist(err) {
		return err
	}

	// 创建本地文件
	newFile, err := os.OpenFile(newFilePath, os.O_CREATE|os.O_EXCL|os.O_RDWR, os.ModePerm)
	if err != nil {
		return err
	}
	defer util.SafeClose(newFile)

	var newFileWriter io.WriteCloser = newFile

	// 压缩判断
	var compress = fileHeader.Size > compressionThreshold
	if compress {
		newFileWriter = gzip.NewWriter(newFile)
		defer util.SafeClose(newFileWriter)
	}

	// IO 落盘
	if _, err = io.Copy(newFileWriter, reader); err != nil {
		return err
	}

	slog.InfoContext(ctx, "文件上传",
		slog.String("path", absPath),
		slog.String("name", fileHeader.Filename),
		slog.Int64("size", fileHeader.Size),
		slog.String("hash", hash),
	)

	// 持久化数据
	now := time.Now().UnixMilli()

	object := &model.Object{
		Id:          id.Next().Int64(),
		Path:        absPath,
		Compression: util.If(compress, model.ObjectCompressionGzip, model.ObjectCompressionNone),
		Hash:        hash,
		Size:        fileHeader.Size,
		RefCount:    0,
		ContentType: contentType,
		CreateTime:  now,
		UpdateTime:  now,
	}
	if err := gorm.G[model.Object](db.Session(ctx)).Create(ctx, object); err != nil {
		return err
	}

	return s.NewObjectRef(ctx, memberId, parentId, object.Id, fileHeader.Filename)
}

// RandDir 目录打散
func (s *ResourceService) RandDir() []string {

	var ret []string

	now := time.Now()

	ret = append(ret, fmt.Sprintf("%d", now.Year()))
	ret = append(ret, fmt.Sprintf("%02d", now.Month()))
	ret = append(ret, fmt.Sprintf("%02d", now.Day()))

	return ret
}

// Mkdir 创建文件夹
func (s *ResourceService) Mkdir(ctx context.Context, request *web.ResourceMkdirRequest) error {

	var resourceId = id.Next().Int64()

	var (
		newPath         = strconv.FormatInt(resourceId, 10) + model.ResourcePathSeparator
		newDepth uint64 = 0
	)

	if request.ParentId != model.DefaultResourceParentId {
		// 确定父目录存在
		dir, err := gorm.G[*model.Resource](db.Session(ctx).Clauses(clause.Locking{Strength: "UPDATE"})). // for update
															Select("id", "dir", "path", "depth").
															Where("id = ? AND member_id = ?", request.ParentId, request.MemberId).
															Take(ctx)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		// 父目录存在，则拼接
		newPath = dir.Path + newPath
		newDepth = dir.Depth + 1
	}

	// 重名处理

	title := request.Title

	var counter = 1
	for {
		var existsId int64
		err := gorm.G[model.Resource](db.Session(ctx)).Select("id").
			Where("member_id = ? AND parent_id = ? AND title = ?", request.MemberId, request.ParentId, request.Title).
			Scan(ctx, &existsId)

		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if existsId == 0 {
			break // Ok 没重复
		}
		request.Title = fmt.Sprintf("%s(%d)", title, counter)
		counter++
	}

	// 保存
	now := time.Now().UnixMilli()

	return gorm.G[model.Resource](db.Session(ctx)).Create(ctx, &model.Resource{
		Id:         resourceId,
		MemberId:   request.MemberId,
		ObjectId:   model.DefaultResourceObjectId,
		ParentId:   request.ParentId,
		Title:      request.Title,
		Dir:        true, // 目录
		Path:       newPath,
		Depth:      newDepth,
		CreateTime: now,
		UpdateTime: now,
	})
}

// Rename 重命名文件
func (s *ResourceService) Rename(ctx context.Context, request *web.ResourceRenameRequest) error {

	// 查询资源所在目录
	var parentId uint64
	err := gorm.G[model.Resource](db.Session(ctx)).
		Select("parent_id").
		Where("id = ? AND member_id = ?", request.Id, request.MemberId).Scan(ctx, &parentId)
	if err != nil {
		return err
	}

	title := request.Title

	var counter = 1
	for {
		var existsId int64
		err := gorm.G[model.Resource](db.Session(ctx)).Select("id").
			Where("member_id = ? AND parent_id = ? AND title = ?", request.MemberId, parentId, request.Title).
			Scan(ctx, &existsId)

		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if existsId == 0 {
			break // Ok 没重复
		}
		request.Title = fmt.Sprintf("%s(%d)", title, counter)
		counter++
	}

	// 更新资源名称
	result := db.Session(ctx).
		Table(model.Resource{}.TableName()).
		Where("id = ?", request.Id).UpdateColumns(map[string]any{
		"update_time": time.Now().UnixMilli(),
		"title":       request.Title,
	})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return common.NewServiceError(http.StatusBadRequest, response.Fail(response.CodeBadRequest).WithMessage("更新失败"))
	}
	return nil
}

var DefaultResourceService = NewResourceService()
