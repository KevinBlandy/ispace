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
	"ispace/common/constant"
	"ispace/common/id"
	"ispace/common/response"
	"ispace/common/types"
	"ispace/common/util"
	"ispace/db"
	"ispace/repo/model"
	"ispace/store"
	"ispace/web/handler/api"
	"log/slog"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sort"
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

type ResourceService struct {
	objectService *ObjectService
}

func NewResourceService(service *ObjectService) *ResourceService {
	return &ResourceService{objectService: service}
}

// List 查询资源列表
func (s *ResourceService) List(ctx context.Context, request *api.ResourceListRequest) ([]*api.ResourceListResponse, error) {

	var ret = make([]*api.ResourceListResponse, 0)

	params := []any{request.MemberId, request.ParentId}

	statement := &strings.Builder{}
	_, _ = statement.WriteString(`SELECT
				t.id,
				t.parent_id,
				t.title,
				t.dir,
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
				t.parent_id = ?`)

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
		resource := &api.ResourceListResponse{}
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
	Status      model.ObjectStatus      // 文件状态
	Path        string                  // 相对路径
}, err error) {
	row := db.Session(ctx).Raw(`
			SELECT
				t.title,
				t1.compression,
				t1.content_type,
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

	var err error
	title, err = s.UniqueTitle(ctx, false, title, resourceId, memberId, parentId)
	if err != nil {
		return err
	}

	now := time.Now().UnixMilli()

	err = gorm.G[model.Resource](db.Session(ctx)).Create(ctx, &model.Resource{
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
	if strings.TrimSpace(fileHeader.Filename) == "" {
		return common.NewServiceError(http.StatusBadRequest, response.Fail(response.CodeBadRequest).WithMessage("文件名称不能为空"))
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
	objectId, err := s.objectService.Exists(ctx, "hash", hash)
	if err != nil {
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

	// 目录打散 & 随机文件名称
	newFilePath := path.Join(path.Join(s.RandDir()...), id.UUID())

	// 创建文件
	newFile, err := store.DefaultStore().OpenFile(newFilePath, os.O_CREATE|os.O_EXCL|os.O_RDWR, os.ModePerm)
	if err != nil {
		return err
	}
	defer util.SafeClose(newFile)

	var writer io.WriteCloser = newFile

	// 非音/视频文件且体积大于阈值，才进行压缩
	var compress = !(strings.HasPrefix(contentType, "video/") || strings.HasPrefix(contentType, "audio/")) &&
		fileHeader.Size > compressionThreshold
	if compress {
		writer = gzip.NewWriter(newFile)
		defer util.SafeClose(writer)
	}

	// 写入
	written, err := io.Copy(writer, file)
	if err != nil {
		return err
	}

	// 查询文件状态
	stat, err := newFile.Stat()
	if err != nil {
		return err
	}

	slog.InfoContext(ctx, "新文件",
		slog.String("name", fileHeader.Filename),
		slog.Int64("size", fileHeader.Size),
		slog.String("path", newFilePath),
		slog.Int64("written", written),
		slog.String("hash", hash),
	)

	// 持久化数据
	now := time.Now().UnixMilli()

	object := &model.Object{
		Id:          id.Next().Int64(),
		Path:        newFilePath,
		Compression: util.If(compress, model.ObjectCompressionGzip, model.ObjectCompressionNone),
		Hash:        hash,
		Size:        fileHeader.Size,
		FileSize:    stat.Size(),
		RefCount:    0,
		ContentType: contentType,
		Status:      model.ObjectStatusPendingReview, // 默认待审核状态
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
	now := time.Now().UnixMilli()
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

// Delete 删除资源
func (s *ResourceService) Delete(ctx context.Context, request *api.ResourceDeleteRequest) error {

	session := db.Session(ctx)

	// 查询要删除的资源
	for _, resourceId := range request.Id {
		resource, err := gorm.G[*model.Resource](session).
			Select("id", "path", "object_id", "dir").
			Where("id = ? AND member_id = ?", resourceId, request.MemberId).Take(ctx)

		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			continue
		}

		if resource.Dir {
			err := func() error {
				// 删除的是目录，查询所有子级资源
				rows, err := session.Table(model.Resource{}.TableName()).
					Select("id", "path", "object_id", "dir").
					Where("member_id = ? AND path LIKE ?", request.MemberId, resource.Path+"%").Rows()

				if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
					return err
				}
				defer util.SafeClose(rows)

				for rows.Next() {
					var subResource = &model.Resource{}
					if err := session.ScanRows(rows, subResource); err != nil {
						return err
					}
					if err := s.delete(ctx, subResource); err != nil {
						return err
					}
				}
				return nil
			}()

			if err != nil {
				return err
			}
		} else {
			// 删除文件，
			if err := s.delete(ctx, resource); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *ResourceService) delete(ctx context.Context, resource *model.Resource) error {
	affected, err := gorm.G[model.Resource](db.Session(ctx)).Where("id = ?", resource.Id).Delete(ctx)
	if err != nil {
		return err
	}
	if affected == 0 {
		return nil
	}

	if !resource.Dir {
		// 更新引用
		result := db.Session(ctx).
			Table(model.Object{}.TableName()).
			Where("id = ?", resource.ObjectId).UpdateColumns(map[string]any{
			"update_time": time.Now().UnixMilli(),
			"ref_count":   gorm.Expr("ref_count - ?", 1),
		})
		if result.Error != nil {
			return result.Error
		}

		if result.RowsAffected != 1 {
			return common.NewServiceError(http.StatusBadRequest, response.Fail(response.CodeBadRequest).WithMessage("存储引用更新失败"))
		}
	}

	// TODO 关联的业务数据处理

	return nil
}

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

	// 按路径长度排序（短的在前，父节点一定比子节点短）
	sort.SliceStable(resources, func(i, j int) bool {
		return len(resources[i].Path) < len(resources[j].Path)
	})

	for i, resource := range resources {
		if !resource.Dir {
			continue // 忽略文件
		}
		for j := i + 1; j < len(resources); j++ {
			child := resources[j]
			if strings.HasPrefix(child.Path, resource.Path) {
				return common.NewServiceError(http.StatusBadRequest,
					response.Fail(response.CodeBadRequest).
						WithMessage("嵌套的资源："+fmt.Sprintf("%s -> %s", resource.Title, child.Title)))
			}
		}
	}

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
	var now = time.Now().UnixMilli()

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
				"path":        gorm.Expr("? || replace(path, ?, '')", parentPath, commonPrefix),
				"depth":       gorm.Expr("depth - ? + 1", diffDepth),
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

// Tree 查询完整的资源树
func (s *ResourceService) Tree(ctx context.Context, memberId int64) ([]*api.ResourceTreeResponse, error) {
	session := db.Session(ctx)
	rows, err := session.Raw(`
			SELECT
				t.id,
				t.parent_id,
				t.title,
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
				t.member_id = 1
			ORDER BY dir DESC, title ASC`, memberId).Rows()
	if err != nil {
		return nil, err
	}
	defer util.SafeClose(rows)

	var resources = make(map[int64]*api.ResourceTreeResponse)

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
				if err := s.Upload(ctx, memberId, parent.Id, file); err != nil {
					return err
				}
			} else {
				// 目录
				// TODO 如果目录已存在，则应该直接返回
				parent, err = s.mkdir(ctx, &api.ResourceMkdirRequest{
					MemberId: memberId,
					ParentId: parent.Id,
					Title:    section,
				})
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

var DefaultResourceService = NewResourceService(DefaultObjectService)
