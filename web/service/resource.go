package service

import (
	"compress/gzip"
	"container/list"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"ispace/common"
	"ispace/common/constant"
	"ispace/common/id"
	"ispace/common/page"
	"ispace/common/response"
	"ispace/common/types"
	"ispace/common/util"
	"ispace/db"
	"ispace/repo/model"
	"ispace/store"
	"ispace/web/handler/api"
	"log/slog"
	"maps"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"
)

const sniffLen = 512

type ResourceService struct {
	objectService        *ObjectService
	memberService        *MemberService
	compressionThreshold int64    // 压缩阈值
	unCompressionType    []string // 不进行压缩的媒体类型
}

func NewResourceService(objectService *ObjectService, memberService *MemberService, compressionThreshold int64, unCompressionType []string) *ResourceService {
	return &ResourceService{
		objectService:        objectService,
		memberService:        memberService,
		compressionThreshold: compressionThreshold,
		unCompressionType:    unCompressionType,
	}
}

// Compression 是否满足压缩条件
func (s *ResourceService) Compression(contentType string, size int64) bool {
	if size < s.compressionThreshold {
		return false
	}
	for _, t := range s.unCompressionType {
		if strings.HasPrefix(contentType, t) {
			return false
		}
	}
	return true
}

// List 查询资源列表
func (s *ResourceService) List(ctx context.Context, request *api.ResourceListRequest) ([]*api.ResourceListResponse, error) {

	//var ret = make([]*api.ResourceListResponse, 0)

	params := []any{request.MemberId}

	statement := &strings.Builder{}
	_, _ = statement.WriteString(`SELECT
				t.id,
				t.parent_id,
				t.title,
				t.content_type,
				t.dir,
				t.create_time,
				t.update_time,
				-- 文件大小
				t1.size size,
				-- 文件状态
				t1.status status
			FROM
				t_resource t
				LEFT JOIN t_object t1 ON t1.id = t.object_id AND t.dir = 0
			WHERE
				t.member_id = ?`)

	// 条件
	if request.ParentId >= 0 {
		statement.WriteString(" AND t.parent_id = ?")
		params = append(params, request.ParentId)
	}
	//if request.Dir != nil {
	//	statement.WriteString(" AND t.dir = ?")
	//	params = append(params, request.Dir)
	//}
	//if request.Keywords != "" {
	//	statement.WriteString(" AND t.title = ?")
	//	params = append(params, "%"+request.Keywords+"%")
	//}

	return db.List[api.ResourceListResponse](ctx, statement.String(), params...)
}

// Get 获取资源信息
func (s *ResourceService) Get(ctx context.Context, memberId, resourceId int64) (ret struct {
	Id          int64
	Title       string                  // 文件标题
	Compression model.ObjectCompression // 压缩方式
	ContentType string                  // 文件类型
	Status      model.ObjectStatus      // 文件状态
	Path        string                  // 相对路径
}, err error) {
	row := db.Session(ctx).Raw(`
			SELECT
				t.id,
				t.title,
				t.content_type,
				t1.compression,
				t1.path,
				t1.status
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
	err = row.Scan(&ret.Id, &ret.Title, &ret.ContentType, &ret.Compression, &ret.Path, &ret.Status)
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
		dir, err := gorm.G[*model.Resource](db.Session(ctx)). // for update
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

	var err error
	title, err = s.UniqueTitle(ctx, false, title, resourceId, memberId, parentId)
	if err != nil {
		return err
	}

	// 根据文件名称计算资源的 ContentType
	contentType := mime.TypeByExtension(filepath.Ext(title))
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	now := util.ContextValueDefault(ctx, constant.CtxKeyRequestTime, time.Now()).UnixMilli()

	err = gorm.G[model.Resource](db.Session(ctx)).Create(ctx, &model.Resource{
		Id:          resourceId,
		MemberId:    memberId,
		ObjectId:    objectId,
		ParentId:    parentId,
		Title:       title,
		ContentType: contentType,
		Dir:         false, // 文件
		Path:        newPath,
		Depth:       newDepth,
		CreateTime:  now,
		UpdateTime:  now,
	})

	if err != nil {
		return err
	}

	return s.objectService.UpdateRefCount(ctx, objectId, 1)
}

// Resource 资源描述
type Resource interface {
	io.Reader
	io.ReaderAt
	io.Seeker
	io.Closer
}

// UploadResource 资源接口
type UploadResource interface {
	Filename() string        // 文件名称
	Size() int64             // 文件大小
	Open() (Resource, error) // 打开 Reader
}

type LocalFileResource struct {
	*os.File
	size     int64
	filename string
}

func NewLocalFileResource(
	size int64,
	filename string,
	file *os.File) *LocalFileResource {
	return &LocalFileResource{File: file, size: size, filename: filename}
}

func (r *LocalFileResource) Open() (Resource, error) {
	return r.File, nil
}

func (r *LocalFileResource) Size() int64 {
	return r.size
}
func (r *LocalFileResource) Filename() string {
	return r.filename
}

// Upload 资源上传
func (s *ResourceService) Upload(ctx context.Context, memberId int64, parentId int64, fileHeader UploadResource) error {
	if fileHeader.Size() == 0 {
		return common.NewServiceError(http.StatusBadRequest, response.Fail(response.CodeBadRequest).WithMessage("不能上传空文件"))
	}
	if strings.TrimSpace(fileHeader.Filename()) == "" {
		return common.NewServiceError(http.StatusBadRequest, response.Fail(response.CodeBadRequest).WithMessage("文件名称不能为空"))
	}

	// 先累计空间使用量
	if err := s.memberService.AddUsedStorageSpace(ctx, memberId, fileHeader.Size()); err != nil {
		return err
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
	hash := strings.ToLower(hex.EncodeToString(hasher.Sum(nil))) // 统一 hash 为小写

	// 查询 Hash 是否存在
	objectId, err := s.objectService.Exists(ctx, "hash", hash)
	if err != nil {
		return err
	}
	if objectId > 0 {
		// 已存在了文件，复制引用即可
		return s.NewObjectRef(ctx, memberId, parentId, objectId, fileHeader.Filename())
	}

	// 重置指针
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return err
	}

	// 查询媒体类型
	contentType := mime.TypeByExtension(filepath.Ext(fileHeader.Filename()))
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

	contentType = strings.ToLower(contentType)

	// 目录打散 & 随机文件名称
	newFilePath := path.Join(path.Join(s.RandDir(ctx)...), id.UUID())

	// 创建文件
	newFile, err := store.DefaultStore().OpenFile(newFilePath, os.O_CREATE|os.O_EXCL|os.O_RDWR, os.ModePerm)
	if err != nil {
		return err
	}
	defer util.SafeClose(newFile)

	var writer io.WriteCloser = newFile

	// 压缩判断
	var compress = s.Compression(contentType, fileHeader.Size())

	if compress {
		writer = gzip.NewWriter(newFile)
		defer util.SafeClose(writer)
	}

	// 写入
	written, err := io.Copy(writer, file)
	if err != nil {
		return err
	}

	slog.InfoContext(ctx, "新文件",
		slog.String("name", fileHeader.Filename()),
		slog.Int64("size", fileHeader.Size()),
		slog.String("path", newFilePath),
		slog.Int64("written", written),
		slog.String("hash", hash),
	)

	// 刷出缓存 flush
	if compress {
		if err := writer.Close(); err != nil {
			return err
		}
	}

	if err := newFile.Sync(); err != nil {
		return err
	}

	// 查询文件状态
	stat, err := newFile.Stat()
	if err != nil {
		return err
	}

	// 持久化数据
	now := util.ContextValueDefault(ctx, constant.CtxKeyRequestTime, time.Now()).UnixMilli()

	object := &model.Object{
		Id:          id.Next().Int64(),
		Path:        newFilePath,
		Compression: util.If(compress, model.ObjectCompressionGzip, model.ObjectCompressionNone),
		Hash:        hash,
		Size:        fileHeader.Size(), // 逻辑大小
		FileSize:    stat.Size(),       // 实际大小
		RefCount:    0,
		ContentType: contentType,
		Status:      model.ObjectStatusPendingReview, // 默认待审核状态
		CreateTime:  now,
		UpdateTime:  now,
	}
	if err := gorm.G[model.Object](db.Session(ctx)).Create(ctx, object); err != nil {
		return err
	}

	// 创建对象引用
	return s.NewObjectRef(ctx, memberId, parentId, object.Id, fileHeader.Filename())
}

//// AddUsedStorageSpace 累加会员的资源用量，如果超出最大限制则返回异常
//func (s *ResourceService) AddUsedStorageSpace(ctx context.Context, memberId int64, size int64) error {
//
//	if size == 0 {
//		return nil
//	}
//
//	var where = "id = ? "
//	var params = []any{memberId}
//
//	if size > 0 {
//		// 累加的时候，要注意不能超出用户最大的限制
//		where += " AND max_storage_space >= used_storage_space + ?"
//		params = append(params, size)
//	}
//
//	result := db.Session(ctx).
//		Table(model.Member{}.TableName()).
//		Where(where, params...).
//		UpdateColumns(map[string]any{
//			"used_storage_space": gorm.Expr("used_storage_space + ?", size),
//		})
//
//	if result.Error != nil {
//		return result.Error
//	}
//	if result.RowsAffected == 0 {
//		return common.NewServiceError(http.StatusBadRequest, response.Fail(response.CodeBadRequest).WithMessage("可用存储空间不足"))
//	}
//	return nil
//}

// MultipartPartResource Multipart 上传
type MultipartPartResource struct {
	*multipart.FileHeader
}

func (m *MultipartPartResource) Filename() string {
	return m.FileHeader.Filename
}
func (m *MultipartPartResource) Size() int64 {
	return m.FileHeader.Size
}
func (m *MultipartPartResource) Open() (Resource, error) {
	return m.FileHeader.Open()
}

func NewMultipartPartResource(header *multipart.FileHeader) *MultipartPartResource {
	return &MultipartPartResource{FileHeader: header}
}

// UploadMultipart 上传文件到磁盘
func (s *ResourceService) UploadMultipart(ctx context.Context, memberId int64, parentId int64, fileHeader *multipart.FileHeader) error {
	return s.Upload(ctx, memberId, parentId, NewMultipartPartResource(fileHeader))
}

// RandDir 目录打散
func (s *ResourceService) RandDir(ctx context.Context) []string {

	var ret []string

	now := util.ContextValueDefault(ctx, constant.CtxKeyRequestTime, time.Now())

	ret = append(ret, fmt.Sprintf("%d", now.Year()))
	ret = append(ret, fmt.Sprintf("%02d", now.Month()))
	ret = append(ret, fmt.Sprintf("%02d", now.Day()))

	return ret
}

// Mkdir 创建文件夹
func (s *ResourceService) Mkdir(ctx context.Context, request *api.ResourceMkdirRequest) error {
	_, err := s.mkdir(ctx, request)
	return err
}

// mkdir 创建文件夹
func (s *ResourceService) mkdir(ctx context.Context, request *api.ResourceMkdirRequest) (*model.Resource, error) {

	var resourceId = id.Next().Int64()

	var (
		newPath         = strconv.FormatInt(resourceId, 10) + model.ResourcePathSeparator
		newDepth uint64 = 0
	)

	if request.ParentId != model.DefaultResourceParentId {
		// 确定父目录存在
		dir, err := gorm.G[*model.Resource](db.Session(ctx)). // for update
									Select("id", "dir", "path", "depth").
									Where("id = ? AND member_id = ?", request.ParentId, request.MemberId).
									Take(ctx)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, common.NewServiceError(http.StatusBadRequest, response.Fail(response.CodeBadRequest).WithMessage("父目录不存在"))
		}

		// 父目录存在，则拼接
		newPath = dir.Path + newPath
		newDepth = dir.Depth + 1
	}

	// 重名处理
	var err error
	request.Title, err = s.UniqueTitle(ctx, true, request.Title, resourceId, request.MemberId, request.ParentId)
	if err != nil {
		return nil, err
	}

	// 保存
	now := util.ContextValueDefault(ctx, constant.CtxKeyRequestTime, time.Now()).UnixMilli()

	r := &model.Resource{
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
	}
	return r, gorm.G[model.Resource](db.Session(ctx)).Create(ctx, r)
}

// Rename 重命名文件
func (s *ResourceService) Rename(ctx context.Context, request *api.ResourceRenameRequest) error {

	// 查询资源所在目录
	//var parentId uint64
	resource, err := gorm.G[model.Resource](db.Session(ctx)).
		Select("id", "parent_id", "dir").
		Where("id = ? AND member_id = ?", request.Id, request.MemberId).Take(ctx)
	if err != nil {
		return err
	}

	title, err := s.UniqueTitle(ctx, resource.Dir, request.Title, resource.Id, request.MemberId, resource.ParentId)
	if err != nil {
		return err
	}

	// 更新
	params := map[string]any{
		"update_time": util.ContextValueDefault(ctx, constant.CtxKeyRequestTime, time.Now()).UnixMilli(),
		"title":       title,
	}

	// 文件的话，根据名称重新计算 ContentType
	if !resource.Dir {
		contentType := mime.TypeByExtension(filepath.Ext(title))
		if contentType == "" {
			contentType = "application/octet-stream"
		}
		params["content_type"] = contentType
	}

	result := db.Session(ctx).
		Table(model.Resource{}.TableName()).
		Where("id = ?", request.Id).
		UpdateColumns(params)

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return common.NewServiceError(http.StatusBadRequest, response.Fail(response.CodeBadRequest).WithMessage("更新失败"))
	}
	return nil
}

//// Delete 物理删除资源
//func (s *ResourceService) Delete(ctx context.Context, request *api.ResourceDeleteRequest) error {
//
//	session := db.Session(ctx)
//
//	// 查询要删除的资源
//	for _, resourceId := range request.Id {
//		resource, err := gorm.G[*model.Resource](session).
//			Select("id", "path", "object_id", "dir").
//			Where("id = ? AND member_id = ?", resourceId, request.MemberId).Take(ctx)
//
//		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
//			return err
//		}
//		if errors.Is(err, gorm.ErrRecordNotFound) {
//			continue
//		}
//
//		if resource.Dir {
//			err := func() error {
//				// 删除的是目录，查询所有子级资源，包括自己
//				rows, err := session.Table(model.Resource{}.TableName()).
//					Select("id", "path", "object_id", "dir").
//					Where("member_id = ? AND path LIKE ?", request.MemberId, resource.Path+"%").Rows()
//
//				if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
//					return err
//				}
//				defer util.SafeClose(rows)
//
//				for rows.Next() {
//					var subResource = &model.Resource{}
//					if err := session.ScanRows(rows, subResource); err != nil {
//						return err
//					}
//					if err := s.delete(ctx, subResource); err != nil {
//						return err
//					}
//				}
//				return nil
//			}()
//
//			if err != nil {
//				return err
//			}
//		} else {
//			// 删除文件，
//			if err := s.delete(ctx, resource); err != nil {
//				return err
//			}
//		}
//	}
//	return nil
//}
//
//func (s *ResourceService) delete(ctx context.Context, resource *model.Resource) error {
//	affected, err := gorm.G[model.Resource](db.Session(ctx)).Where("id = ?", resource.Id).Delete(ctx)
//	if err != nil {
//		return err
//	}
//	if affected == 0 {
//		return nil
//	}
//
//	if !resource.Dir {
//		return s.objectService.UpdateRefCount(ctx, resource.ObjectId, -1)
//	}
//
//	// TODO 关联的业务数据处理
//
//	return nil
//}

// Move 移动资源
func (s *ResourceService) Move(ctx context.Context, request *api.ResourceMoveRequest) error {

	session := db.Session(ctx)

	var (
		parentPath        = ""
		parentDepth int64 = -1 // 根目录子项目的 depth 值是 0，这里为了后面计算的方便，设置 “根” 目录的 depth 为 -1
	)

	if request.ParentId != model.DefaultResourceParentId {
		// 查询父目录
		parent, err := gorm.G[*model.Resource](session).
			Select("id", "path", "dir", "depth").
			Where("id = ? AND member_id = ?", request.ParentId, request.MemberId).Take(ctx)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if errors.Is(err, gorm.ErrRecordNotFound) || !parent.Dir {
			return common.NewServiceError(http.StatusBadRequest, response.Fail(response.CodeBadRequest).WithMessage("目标目录不存在"))
		}
		parentPath = parent.Path
		parentDepth = int64(parent.Depth)
	}

	// 要移动的资源列表
	resources, err := gorm.G[*model.Resource](session).
		Select("id", "path", "dir", "depth", "title").
		Where("id IN ? AND member_id = ?", request.Id, request.MemberId).Find(ctx)

	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	// 嵌套检查
	nestedRes, ok := s.Nested(resources)
	if ok {
		// 存在嵌套
		return common.NewServiceError(http.StatusBadRequest,
			response.Fail(response.CodeBadRequest).
				WithMessage("嵌套的资源："+fmt.Sprintf("%s -> %s", nestedRes[0].Title, nestedRes[1].Title)))
	}

	//// 按路径长度排序（短的在前，父节点一定比子节点短）
	//sort.SliceStable(resources, func(i, j int) bool {
	//	return len(resources[i].Path) < len(resources[j].Path)
	//})
	//
	//for i, resource := range resources {
	//	if !resource.Dir {
	//		continue // 忽略文件
	//	}
	//	for j := i + 1; j < len(resources); j++ {
	//		child := resources[j]
	//		if strings.HasPrefix(child.Path, resource.Path) {
	//			return common.NewServiceError(http.StatusBadRequest,
	//				response.Fail(response.CodeBadRequest).
	//					WithMessage("嵌套的资源："+fmt.Sprintf("%s -> %s", resource.Title, child.Title)))
	//		}
	//	}
	//}

	// 目标目录不能是源资源的子目录
	if parentPath != "" {
		for _, resource := range resources {
			if !resource.Dir {
				continue // 忽略文件
			}
			if strings.HasPrefix(parentPath, resource.Path) {
				return common.NewServiceError(http.StatusBadRequest,
					response.Fail(response.CodeBadRequest).
						WithMessage("不允许移动资源到子目录中"))
			}
		}
	}

	// 更新子父级关系
	var now = util.ContextValueDefault(ctx, constant.CtxKeyRequestTime, time.Now()).UnixMilli()

	for _, resource := range resources {
		// 共同的前缀
		commonPrefix := strings.TrimSuffix(resource.Path, strconv.FormatInt(resource.Id, 10)+model.ResourcePathSeparator)
		// depth 差值
		diffDepth := int64(resource.Depth) - parentDepth

		result := session.
			Table(model.Resource{}.TableName()).
			Where("member_id = ? AND path LIKE ?", request.MemberId, resource.Path+"%").
			UpdateColumns(map[string]any{
				"update_time": now,
				//"path":        gorm.Expr("? || replace(path, ?, '')", parentPath, commonPrefix),
				"path":  gorm.Expr("CONCAT(?, replace(path, ?, ''))", parentPath, commonPrefix),
				"depth": gorm.Expr("depth - ? + 1", diffDepth),
			})

		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return common.NewServiceError(http.StatusBadRequest, response.Fail(response.CodeBadRequest).WithMessage("资源移动失败"))
		}

		// 重名处理
		resource.Title, err = s.UniqueTitle(ctx, resource.Dir, resource.Title, resource.Id, request.MemberId, request.ParentId)
		if err != nil {
			return err
		}

		/*
			更新当前记录的 parentId
		*/
		result = session.
			Table(model.Resource{}.TableName()).
			Where("id = ?", resource.Id).
			UpdateColumns(map[string]any{
				"title":     resource.Title,
				"parent_id": request.ParentId,
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return common.NewServiceError(http.StatusBadRequest, response.Fail(response.CodeBadRequest).WithMessage("资源移动失败"))
		}
	}
	return nil
}

// ResourceTree 资源树
type ResourceTree struct {
	Id          int64              `json:"id,string"`
	ParentId    int64              `json:"parentId,string"`   // 父级资源 ID
	Title       string             `json:"title"`             // 资源标题
	ContentType string             `json:"contentType"`       // 媒体类型
	Dir         bool               `json:"dir"`               // 是否是目录
	Size        int64              `json:"size,string"`       // 文件大小
	Status      model.ObjectStatus `json:"status"`            // 文件状态
	CreateTime  int64              `json:"createTime,string"` // 创建时间
	UpdateTime  int64              `json:"updateTime,string"` // 更新时间
	Entries     []*ResourceTree    `json:"entries"`           // 子项目
}

// entries 查询某个资源下的所有资源树，使用队列检索
func (s *ResourceService) entries(ctx context.Context, resourceId int64) ([]*ResourceTree, error) {
	// 查询直接子记录
	var subEntities = func(ctx context.Context, parentId int64) ([]*ResourceTree, error) {
		return db.List[ResourceTree](ctx, `
			SELECT
				t.id,
				t.parent_id,
				t.title,
				t.content_type,
				t.dir,
				t.create_time,
				t.update_time,
				t1.size size,
				t1.status status
			FROM
				t_resource t
				LEFT JOIN t_object t1 ON t1.id = t.object_id AND t.dir = 0
			WHERE
				t.parent_id = ?
		`, parentId)
	}

	root, err := subEntities(ctx, resourceId)
	if err != nil {
		return nil, err
	}

	// 添加到队列
	queue := list.New()
	for _, item := range root {
		if item.Dir {
			queue.PushBack(item)
		}
	}

	for queue.Len() > 0 {
		item := queue.Remove(queue.Front()).(*ResourceTree)
		// 查询子项目
		sub, err := subEntities(ctx, item.Id)
		if err != nil {
			return nil, err
		}
		item.Entries = sub
		// 存在子项目，则进行迭代
		if len(item.Entries) > 0 {
			for _, entry := range item.Entries {
				if entry.Dir {
					queue.PushBack(entry)
				}
			}
		}
	}
	return root, nil
}

// Tree 查询用户完整的资源树
func (s *ResourceService) Tree(ctx context.Context, memberId int64) ([]*api.ResourceTreeResponse, error) {

	session := db.Session(ctx)
	rows, err := session.Raw(`
			SELECT
				t.id,
				t.parent_id,
				t.title,
				t.content_type,
				t.dir,
				t.create_time,
				t.update_time,
				t1.size size,
				-- 文件状态
				t1.status status
			FROM
				t_resource t
				LEFT JOIN t_object t1 ON t1.id = t.object_id AND t.dir = 0
			WHERE
				t.member_id = ?
			ORDER BY t.dir DESC, t.title ASC, t.create_time DESC`, memberId).Rows()
	if err != nil {
		return nil, err
	}
	defer util.SafeClose(rows)

	var resources = make(map[int64]*api.ResourceTreeResponse) // 前端对结果排序

	for rows.Next() {
		resource := &api.ResourceListResponse{}
		if err := session.ScanRows(rows, resource); err != nil {
			return nil, err
		}
		resources[resource.Id] = &api.ResourceTreeResponse{ResourceListResponse: *resource}
	}

	// 先查询根元素
	var root = make([]*api.ResourceTreeResponse, 0)
	for _, resource := range resources {
		if resource.ParentId == model.DefaultResourceParentId {
			root = append(root, resource)
			delete(resources, resource.Id)
		}
	}

	// 构建树结构
	var subEntry func(*api.ResourceTreeResponse, map[int64]*api.ResourceTreeResponse)

	subEntry = func(resource *api.ResourceTreeResponse, resourceMap map[int64]*api.ResourceTreeResponse) {
		for _, entry := range resourceMap {
			if entry.ParentId == resource.Id {
				resource.Entries = append(resource.Entries, entry)
				delete(resourceMap, resource.Id)
			}
		}
		if len(resource.Entries) > 0 {
			for _, entry := range resource.Entries {
				subEntry(entry, resourceMap)
			}
		}
	}
	for _, resource := range root {
		subEntry(resource, resources)
	}
	return root, nil
}

// UniqueTitle 计算指定目录下的唯一文件名称
func (s *ResourceService) UniqueTitle(ctx context.Context, dir bool, title string, id, memberId, parentId int64) (string, error) {

	// 目录的话，忽略扩展名称
	ext := util.If(dir, "", filepath.Ext(title))

	// 文件的话，截掉扩展
	fileName := util.If(dir, title, strings.TrimSuffix(title, ext))

	var counter = 1

	for {
		var existsId int64
		err := gorm.G[model.Resource](db.Session(ctx)).Select("id").
			Where("member_id = ? AND parent_id = ? AND title = ? AND id <> ?", memberId, parentId, title, id).
			Scan(ctx, &existsId)

		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return "", err
		}
		if existsId == 0 {
			break // Ok 没重复
		}

		// 递增序号
		title = fmt.Sprintf("%s(%d)%s", fileName, counter, ext)

		counter++
	}

	return title, nil
}

// UploadDir 上传文件夹
func (s *ResourceService) UploadDir(ctx context.Context, memberId int64, parentId int64, dirs map[string][]*multipart.FileHeader) error {
	for k, v := range dirs {
		if err := s.uploadDir(ctx, memberId, parentId, k, v); err != nil {
			return err
		}
	}
	return nil
}

// uploadDir 上传文件
func (s *ResourceService) uploadDir(ctx context.Context, memberId int64, parentId int64, dirTitle string, files []*multipart.FileHeader) error {

	// 创建根目录
	var err error
	root, err := s.mkdir(ctx, &api.ResourceMkdirRequest{
		MemberId: memberId,
		ParentId: parentId,
		Title:    dirTitle,
	})
	if err != nil {
		return err
	}

	session := db.Session(ctx)

	// 创建目录 & 文件
	for _, file := range files {

		// 父级目录
		var parent = root

		// 拆分完整路径
		sections := strings.Split(file.Filename, constant.Slash)

		for i, section := range sections {
			if i == len(sections)-1 {
				// 文件
				file.Filename = section // 重置文件名称
				if err := s.UploadMultipart(ctx, memberId, parent.Id, file); err != nil {
					return err
				}
			} else {
				// 目录
				existsParent, err := gorm.G[model.Resource](session).Select("id").
					Where("member_id = ? AND parent_id = ? AND title = ?", memberId, parent.Id, section).
					Take(ctx)

				if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
					return err
				}
				if errors.Is(err, gorm.ErrRecordNotFound) {
					// 父不存在，创建
					parent, err = s.mkdir(ctx, &api.ResourceMkdirRequest{
						MemberId: memberId,
						ParentId: parent.Id,
						Title:    section,
					})
					if err != nil {
						return err
					}
				} else {
					// 父目录存在
					parent = &existsParent
				}
			}
		}
	}

	return nil
}

// FlashUpload 闪电传
func (s *ResourceService) FlashUpload(ctx context.Context, request *api.ResourceFlashUploadRequest, memberId, parentId int64) error {
	// 根据 hash 检索对象信息
	object, err := gorm.G[model.Object](db.Session(ctx)).Select("id", "size").Where("hash = ?", request.Hash).Take(ctx)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	if object.Id < 1 {
		return common.NewServiceError(http.StatusBadRequest, response.Fail(response.CodeBadRequest).WithMessage("资源不存在"))
	}

	// 先累计空间使用量
	if err := s.memberService.AddUsedStorageSpace(ctx, memberId, object.Size); err != nil {
		return err
	}

	// 创建新的资源
	return s.NewObjectRef(ctx, memberId, parentId, object.Id, request.Title)
}

// Download 下载文件
func (s *ResourceService) Download(ctx context.Context, memberId int64, resourceIds types.Int64Slice) ([]*store.DownloadTree, error) {

	session := db.Session(ctx)

	// 检索资源列表
	rows, err := session.Raw(`SELECT
				t.id,
				t.parent_id,
				t.title,
				t.dir,
				ifnull(t1.path, ''),
				ifnull(t1.compression, '')
			FROM
				t_resource t
				LEFT JOIN t_object t1 ON t1.id = t.object_id AND t.dir = 0
			WHERE
				t.member_id = ?
			AND
				t.id IN ?`, memberId, resourceIds).Rows()
	if err != nil {
		return nil, err
	}

	defer util.SafeClose(rows)

	resources := make([]*store.DownloadTree, 0)

	for rows.Next() {
		resource := new(store.DownloadTree)
		if err := rows.Scan(&resource.Id, &resource.ParentId, &resource.Title, &resource.Dir, &resource.Path, &resource.Compression); err != nil {
			return nil, err
		}
		resources = append(resources, resource)
	}

	// 构建完整的订单树
	var subTree = func(r *store.DownloadTree) error {
		// 检索树下的所有记录
		rows, err := session.Raw(`
			SELECT
				t.id,
				t.parent_id,
				t.title,
				t.dir,
				ifnull(t1.path, ''),
				ifnull(t1.compression, '')
			FROM
				t_resource t
				LEFT JOIN t_object t1 ON t1.id = t.object_id AND t.dir = 0
			WHERE
				t.member_id = ?
			AND
				t.path LIKE CONCAT(
					(SELECT path FROM t_resource WHERE id = ?),
					'%'
				)
			AND
				t.id <> ?
		`, memberId, r.Id, r.Id).Rows()

		if err != nil {
			return err
		}

		defer util.SafeClose(rows)

		// Map 结构存储
		var resourceMap = make(map[int64]*store.DownloadTree)
		for rows.Next() {
			resource := new(store.DownloadTree)
			if err := rows.Scan(&resource.Id, &resource.ParentId, &resource.Title, &resource.Dir, &resource.Path, &resource.Compression); err != nil {
				return err
			}
			resourceMap[resource.Id] = resource
		}

		if len(resourceMap) == 0 {
			return nil
		}

		// 顶级记录
		for _, resource := range resourceMap {
			if resource.ParentId == r.Id {
				r.Entries = append(r.Entries, resource)
				delete(resourceMap, resource.Id)
			}
		}

		var subEntry func(*store.DownloadTree, map[int64]*store.DownloadTree)

		subEntry = func(r *store.DownloadTree, m map[int64]*store.DownloadTree) {
			r.Entries = make([]*store.DownloadTree, 0)
			for _, resource := range m {
				if resource.ParentId == r.Id {
					r.Entries = append(r.Entries, resource)
					delete(m, resource.Id)
				}
			}
			if len(r.Entries) > 0 {
				for _, resource := range r.Entries {
					subEntry(resource, m)
				}
			}
		}

		// 递归构建所有的子记录
		for _, entry := range r.Entries {
			subEntry(entry, resourceMap)
		}

		return nil
	}

	waitGroup := new(sync.WaitGroup)

	for _, resource := range resources {
		// 目录的话，构建完整的文件树
		if resource.Dir {
			waitGroup.Go(func() {
				if err := subTree(resource); err != nil {
					slog.ErrorContext(ctx, "检索资源树异常", slog.String("err", err.Error()))
				}
			})
		}
	}

	waitGroup.Wait()

	return resources, nil
}

// Search 资源搜索
func (s *ResourceService) Search(ctx context.Context, request *api.ResourceSearchRequest) (*page.Pagination[*api.ResourceSearchResponse], error) {
	return db.PageQuery[api.ResourceSearchResponse](ctx, request.Pager, `
			SELECT
				t.id,
				t.title,
				t.content_type,
				t.create_time,
				t.update_time,
				-- 文件大小
				t1.size size,
				-- 文件状态
				t1.status status
			FROM
				t_resource t
				LEFT JOIN t_object t1 ON t1.id = t.object_id
			WHERE
				t.member_id = ?
			AND
				t.dir = ?
			AND
				t.title LIKE ?
`, []any{request.MemberId, false, "%" + request.Keywords + "%"})
}

func (s *ResourceService) Recent(ctx context.Context, request *api.ResourceRecentRequest) (*page.Pagination[*api.ResourceRecentResponse], error) {
	statement := `
			SELECT
				t.id,
				t.title,
				t.content_type,
				t.create_time,
				t.update_time,
				-- 文件大小
				t1.size size,
				-- 文件状态
				t1.status status
			FROM
				t_resource t
				LEFT JOIN t_object t1 ON t1.id = t.object_id
			WHERE
				t.member_id = ?
			AND
				t.dir = ?
`
	params := []any{request.MemberId, false}

	if request.ContentType != "" {
		statement += " AND t.content_type LIKE ?"
		params = append(params, "%"+request.ContentType+"%")
	}

	statement += " ORDER BY t.create_time DESC"

	return db.PageQuery[api.ResourceRecentResponse](ctx, request.Pager, statement, params)
}

// Group 分组查询
func (s *ResourceService) Group(ctx context.Context, request *api.ResourceGroupRequest) (*page.Pagination[*api.ResourceGroupResponse], error) {

	// 客户端时区偏移量
	offset := util.TimeZoneOffset(
		util.ContextValue[time.Time](ctx, constant.CtxKeyRequestTime),
		util.ContextValueDefault(ctx, constant.CtxKeyTimezone, constant.Location),
	)

	// 分组字段
	var groupField string

	// TODO 此处用到了数据库方言
	switch request.Group {
	case "day": // 2026-01-28
		groupField = "date(t.create_time / 1000, 'unixepoch', '" + offset + "')"
	case "week": // 2026-04  TODO 周的计算是从 0 开始的
		groupField = "strftime('%Y-%W', t.create_time / 1000, 'unixepoch', '" + offset + "')"
	case "month": // 2026-01
		groupField = "strftime('%Y-%m', t.create_time / 1000, 'unixepoch', '" + offset + "') "
	case "year": // 2026
		groupField = "strftime('%Y', t.create_time / 1000, 'unixepoch', '" + offset + "')"
	default:
		return nil, common.NewServiceError(http.StatusBadRequest, response.Fail(response.CodeBadRequest).WithMessage("不支持的分组查询"))
	}

	var groupConditions = []any{request.MemberId, false}

	groupStatement := &strings.Builder{}
	groupStatement.WriteString("SELECT " + groupField + " _group FROM t_resource t WHERE t.member_id = ? AND t.dir = ?")

	if request.ContentType != "" {
		groupStatement.WriteString(" AND t.content_type LIKE CONCAT(?, '%')")
		groupConditions = append(groupConditions, request.ContentType)
	}

	groupStatement.WriteString(" GROUP BY _group ORDER BY _group DESC")

	// 分页检索 group 列表
	result, err := db.PageQueryScan[*api.ResourceGroupResponse](ctx,
		request.Pager,
		groupStatement.String(),
		groupConditions,
		func(row *sql.Rows) (*api.ResourceGroupResponse, error) {
			var ret api.ResourceGroupResponse
			return &ret, row.Scan(&ret.Group)
		})

	if err != nil {
		return nil, err
	}

	// 检索分组下的项目
	var itemConditions = []any{request.MemberId, false}

	itemStatement := &strings.Builder{}
	itemStatement.WriteString(`
				SELECT
					t.id,
					t.title,
					t.content_type,
					t.create_time,
					t.update_time,
					-- 文件大小
					t1.size size,
					-- 文件状态
					t1.status status
				FROM
					t_resource t
					LEFT JOIN t_object t1 ON t1.id = t.object_id
				WHERE
					t.member_id = ?
				AND
					t.dir = ?
			`)

	if request.ContentType != "" {
		itemStatement.WriteString(" AND t.content_type CONCAT(?, '%')")
		itemConditions = append(itemConditions, request.ContentType)
	}
	itemStatement.WriteString(" AND " + groupField + " = ?")
	itemStatement.WriteString(" ORDER BY t.create_time DESC")

	// 检索每个分组的内容
	waitGroup := new(sync.WaitGroup)
	for _, row := range result.Rows {
		waitGroup.Add(1)
		// TODO 如果数据集过于庞大，则考虑在此处限制固定的项目数量，单独提供一个 “详情” 页面分页检索不同分组下的人项目
		go func(row *api.ResourceGroupResponse) {
			defer waitGroup.Done()

			conditions := make([]any, len(itemConditions))
			copy(conditions, itemConditions)

			// 分组参数
			conditions = append(conditions, row.Group)

			results, err := db.List[api.ResourceGroupItem](ctx, itemStatement.String(), conditions...)
			if err != nil {
				slog.ErrorContext(ctx, "分组资源检索异常", slog.String("err", err.Error()))
				return
			}
			row.Items = results
		}(row)
	}

	waitGroup.Wait()

	return result, nil
}

// rootAndEntries 检索资源以及所有子级资源
func (s *ResourceService) rootAndEntries(ctx context.Context, memberId, resourceId int64) (*model.Resource, []*model.Resource, error) {

	session := db.Session(ctx)

	resource, err := gorm.G[*model.Resource](session).
		Select("id", "parent_id", "member_id", "object_id", "path", "depth", "title", "content_type", "dir", "create_time").
		Where("id = ? AND member_id = ?", resourceId, memberId).
		Take(context.Background())

	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil, err
	}
	if resource.Id == 0 {
		return nil, nil, err // 记录不存在，或已经被删除了
	}

	// 如果是目录的话，检索所有子记录
	var entries []*model.Resource
	if resource.Dir {
		entries, err = gorm.G[*model.Resource](session).
			Select("id", "parent_id", "member_id", "object_id", "path", "depth", "title", "content_type", "dir", "create_time").
			Where("member_id = ? AND path LIKE CONCAT(?, '%') AND id <> ?", memberId, resource.Path, resource.Id).
			Find(ctx)
		if err != nil {
			return nil, nil, err
		}
	}
	return resource, entries, nil
}

// MoveToRecycleBin 删除资源到回收站
func (s *ResourceService) MoveToRecycleBin(ctx context.Context, request *api.ResourceDeleteRequest) error {
	session := db.Session(ctx)

	for _, rId := range request.Id {
		// 检索要删除的资源，包括子记录
		resource, entries, err := s.rootAndEntries(ctx, request.MemberId, rId)
		if err != nil {
			return err
		}
		if resource == nil {
			continue // 不存在或已被删除
		}

		// 移动到回收站
		if err := s.moveToRecycleBin(ctx, resource, entries); err != nil {
			return err
		}

		// 删除资源
		var rIds = []int64{rId}
		for _, entry := range entries {
			rIds = append(rIds, entry.Id)
		}
		affected, err := gorm.G[model.Resource](session).Where("id IN ?", rIds).Delete(ctx)
		if err != nil {
			return err
		}

		if affected != len(rIds) {
			return common.NewServiceError(http.StatusBadRequest, response.Fail(response.CodeBadRequest).WithMessage("资源删除失败"))
		}

		// 删除分享的记录，只删除文件，不删除文件夹，避免破坏分享目录的结构
		//rIds = make([]int64, 0)
		//if !resource.Dir {
		//	rIds = append(rIds, rId)
		//}
		//for _, entry := range entries {
		//	if !entry.Dir {
		//		rIds = append(rIds, entry.Id)
		//	}
		//}

		if len(rIds) > 0 {
			if _, err := gorm.G[model.ShareResource](session).Where("resource_id IN ? AND resource_dir = ?", rIds, false).Delete(ctx); err != nil {
				return err
			}
		}
	}
	return nil
}

// moveToRecycleBin 保存到回收站
func (s *ResourceService) moveToRecycleBin(ctx context.Context, root *model.Resource, entries []*model.Resource) error {

	now := util.ContextValueDefault(ctx, constant.CtxKeyRequestTime, time.Now()).UnixMilli()

	var items []*model.RecycleBin
	for _, entry := range append(entries, root) {
		items = append(items, &model.RecycleBin{
			Id:                  id.Next().Int64(),
			MemberId:            entry.MemberId,
			Root:                util.If(entry.Id == root.Id, true, false),
			CreateTime:          now,
			ResourceId:          entry.Id,
			ResourceObjectId:    entry.ObjectId,
			ResourceParentId:    entry.ParentId,
			ResourceTitle:       entry.Title,
			ResourceContentType: entry.ContentType,
			ResourceDir:         entry.Dir,
			ResourcePath:        entry.Path,
			ResourceDepth:       entry.Depth,
			ResourceCreateTime:  entry.CreateTime,
		})
	}
	return gorm.G[*model.RecycleBin](db.Session(ctx)).CreateInBatches(ctx, &items, 100)
}

// Share 资源分享
func (s *ResourceService) Share(ctx context.Context, request *api.ResourceShareRequest) (*api.ResourceShareResponse, error) {

	var m = make(map[*model.Resource][]*model.Resource)

	for _, rId := range request.Id {

		resource, entries, err := s.rootAndEntries(ctx, request.MemberId, rId)
		if err != nil {
			return nil, err
		}
		if resource == nil { // 资源已被删除了
			return nil, common.NewServiceError(http.StatusBadRequest, response.Fail(response.CodeBadRequest).WithMessage("资源不存在"))
		}
		m[resource] = entries
	}
	return s.share(ctx, request, m)
}

func (s *ResourceService) share(ctx context.Context, request *api.ResourceShareRequest, resourcesMap map[*model.Resource][]*model.Resource) (*api.ResourceShareResponse, error) {
	// 确定不能存在嵌套分享
	nestedRes, ok := s.Nested(slices.Collect(maps.Keys(resourcesMap)))
	if ok {
		// 存在嵌套
		return nil, common.NewServiceError(http.StatusBadRequest,
			response.Fail(response.CodeBadRequest).
				WithMessage("嵌套的资源："+fmt.Sprintf("%s -> %s", nestedRes[0].Title, nestedRes[1].Title)))
	}

	session := db.Session(ctx)

	// 保存新的分享记录
	now := util.ContextValueDefault(ctx, constant.CtxKeyRequestTime, time.Now())

	var shareId = id.Next().Int64()

	share := &model.Share{
		Id:         shareId,
		MemberId:   request.MemberId,
		Path:       id.PathOfId(shareId), //  Path 生成
		Enabled:    true,                 // 默认启用状态
		Password:   request.Password,
		Views:      0,
		CreateTime: now.UnixMilli(),
		UpdateTime: now.UnixMilli(),
		ExpireTime: 0,
	}

	//  计算过期时间
	switch request.Expire {
	case api.ResourceShareExpireDay:
		share.ExpireTime = now.Add(time.Hour * 24).UnixMilli() // 日
	case api.ResourceShareExpireWeek:
		share.ExpireTime = now.Add(time.Hour * 24 * 7).UnixMilli() // 周
	case api.ResourceShareExpireMonth:
		share.ExpireTime = now.Add(time.Hour * 24 * 30).UnixMilli() // 月
	case api.ResourceShareExpireYear:
		share.ExpireTime = now.Add(time.Hour * 24 * 365).UnixMilli() // 年
	case api.ResourceShareExpireForever: // 永久
		share.ExpireTime = 0
	default:
		// TODO 解析客户端传递过来的自定义的时间
		//expireTimestamp, err := strconv.ParseInt(string(request.Expire), 10, 64)
		//if err == nil {
		//	expireTime := time.UnixMilli(expireTimestamp)
		//	if expireTime.After(now) {
		//		share.ExpireTime = expireTime.UnixMilli()
		//	}
		//}
		return nil, common.NewServiceError(http.StatusBadRequest, response.Fail(response.CodeBadRequest).WithMessage("非法的有效时间"))
	}

	if err := gorm.G[model.Share](session).Create(ctx, share); err != nil {
		return nil, err
	}

	var items = make([]*model.ShareResource, 0)

	for root, resources := range resourcesMap {
		for _, item := range append(resources, root) {
			items = append(items, &model.ShareResource{
				Id:                  id.Next().Int64(),
				ShareId:             share.Id,
				Root:                util.If(item.Id == root.Id, true, false),
				ResourceId:          item.Id,
				ResourceParentId:    item.ParentId,
				ResourceObjectId:    item.ObjectId,
				ResourcePath:        item.Path,
				ResourceTitle:       item.Title,
				ResourceDir:         item.Dir,
				ResourceContentType: item.ContentType,
				ResourceDepth:       item.Depth,
				ResourceCreateTime:  item.CreateTime,
			})
		}
	}
	return &api.ResourceShareResponse{
		Id:         shareId,
		ExpireTime: share.ExpireTime,
		Path:       share.Path,
		Password:   share.Password,
	}, gorm.G[*model.ShareResource](session).CreateInBatches(ctx, &items, 100)
}

// Nested 根据 Path/Depth/Dir 检查是否存在嵌套资源
func (s *ResourceService) Nested(resources []*model.Resource) ([]*model.Resource, bool) {
	sort.SliceStable(resources, func(i, j int) bool {
		//return len(resources[i].Path) < len(resources[j].Path) 	// 按路径长度排序（短的在前，父节点一定比子节点短）
		return resources[i].Depth < resources[j].Depth // 按照深度排序，父节点一定比子节点小
	})
	for i, resource := range resources {
		if !resource.Dir {
			continue // 忽略文件
		}
		for j := i + 1; j < len(resources); j++ {
			child := resources[j]
			if strings.HasPrefix(child.Path, resource.Path) {
				return []*model.Resource{resource, child}, true
			}
		}
	}
	return nil, false
}

// TotalSize 总计资源的逻辑总存储大小
func (s *ResourceService) TotalSize(ctx context.Context, memberId int64) (*api.MemberResourceStatResponse, error) {
	m, err := gorm.G[model.Member](db.Session(ctx)).Select("used_storage_space", "max_storage_space").Where("id = ?", memberId).Take(ctx)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	return &api.MemberResourceStatResponse{
		UsedStorageSpace: m.UsedStorageSpace,
		MaxStorageSpace:  m.MaxStorageSpace,
	}, nil
}

var DefaultResourceService = NewResourceService(
	DefaultObjectService,
	DefaultMemberService,
	int64(types.KB),
	[]string{
		"video/",           // 视频，Range 支持
		"audio/",           // 音频，Range 支持
		"application/zip",  // zip 不压缩，需要在线解压
		"application/gzip", // 本身就是 gzip 的文件不进行压缩
	})
