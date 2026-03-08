package service

import (
	"context"
	"errors"
	"io/fs"
	"ispace/common"
	"ispace/common/constant"
	"ispace/common/id"
	"ispace/common/response"
	"ispace/common/util"
	"ispace/db"
	"ispace/repo/model"
	"ispace/store"
	"ispace/web/handler/api"
	"log/slog"
	"net/http"
	"path"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"gorm.io/gorm"
)

type ResourceChunkService struct {
	resourceService *ResourceService
}

// NewChunkedResource 初始化分配上传
func (s ResourceChunkService) NewChunkedResource(ctx context.Context, request *api.ChunkedResourceRequest) (*api.ChunkedResourceResponse, error) {
	// 是否已经有上传信息
	r, err := gorm.G[model.ResourceChunk](db.Session(ctx)).Where("member_id = ? AND sha256 = ?", request.MemberId, request.Sha256).Count(ctx, "id")
	if err != nil {
		return nil, err
	}
	if r > 0 {
		return nil, common.NewServiceError(http.StatusBadRequest, response.Fail(response.CodeBadRequest).WithMessage("上传任务已存在"))
	}

	// TODO 限制分片上传的资源数量

	m := &model.ResourceChunk{
		Id:         id.Next().Int64(),
		ParentId:   request.ParentId,
		MemberId:   request.MemberId,
		Title:      request.Title,
		Size:       request.Size,
		Sha256:     request.Sha256,
		Path:       path.Join(strconv.FormatInt(request.MemberId, 10), id.UUID()), // 根据用户的 ID 生成唯一的路径
		CreateTime: util.ContextValueDefault(ctx, constant.CtxKeyRequestTime, time.Now()).UnixMilli(),
	}
	return &api.ChunkedResourceResponse{
		Id:       m.Id,
		Title:    m.Title,
		Size:     m.Size,
		Sha256:   m.Sha256,
		Received: 0,
	}, gorm.G[model.ResourceChunk](db.Session(ctx)).Create(ctx, m)
}

// ChunkedResource 查询分片信息
func (s ResourceChunkService) ChunkedResource(ctx context.Context, memberId int64) ([]*api.ChunkedResourceResponse, error) {
	// 是否已经有上传信息
	list, err := gorm.G[*model.ResourceChunk](db.Session(ctx)).Where("member_id = ?", memberId).Order("create_time DESC").Find(ctx)
	if err != nil {
		return nil, err
	}

	var ret = make([]*api.ChunkedResourceResponse, len(list))

	var wg = &sync.WaitGroup{}

	for i, v := range list {
		wg.Add(1)
		go func(i int, v *model.ResourceChunk) {
			defer wg.Done()
			// 文件大小即已接收的字节数量
			var received int64 = 0
			stat, _ := store.DefaultChunkStore().Stat(v.Path)
			if stat != nil {
				received = stat.Size()
			}

			ret[i] = &api.ChunkedResourceResponse{
				Id:       v.Id,
				Title:    v.Title,
				Size:     v.Size,
				Sha256:   v.Sha256,
				Received: received,
			}
		}(i, v)
	}

	wg.Wait()

	return ret, nil
}

func (s ResourceChunkService) Find(ctx context.Context, memberId int64, sourceId int64) (*model.ResourceChunk, error) {
	return gorm.G[*model.ResourceChunk](db.Session(ctx)).Where("id = ? AND member_id = ?", sourceId, memberId).Take(ctx)
}

// UploadComplete 上传文件
func (s ResourceChunkService) UploadComplete(ctx context.Context, chunk *model.ResourceChunk, resource *LocalFileResource) error {

	// 检索父项目是否存在
	if chunk.ParentId != model.DefaultResourceParentId {
		parent, err := gorm.G[model.Resource](db.Session(ctx)).
			Select("id").
			Where("member_id = ? AND id = ? AND dir = ?", chunk.MemberId, chunk.ParentId, true).
			Take(ctx)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		// 父目录不存在了，默认上传到根目录
		if parent.Id < 1 {
			chunk.ParentId = model.DefaultResourceParentId
		}
	}

	// 执行上传
	if err := s.resourceService.Upload(ctx, chunk.MemberId, chunk.ParentId, resource); err != nil {
		return err
	}

	// 删除分片上传记录
	affected, err := gorm.G[model.ResourceChunk](db.Session(ctx)).Where("id = ?", chunk.Id).Delete(ctx)
	if err != nil {
		return err
	}
	if affected == 0 {
		return common.NewServiceError(http.StatusBadRequest, response.Fail(response.CodeBadRequest).WithMessage("上传任务已取消"))
	}
	return nil
}

func (s ResourceChunkService) Cancel(ctx context.Context, memberId int64, sourceId int64) (*model.ResourceChunk, error) {
	ret, err := s.Find(ctx, memberId, sourceId)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	affected, err := gorm.G[model.ResourceChunk](db.Session(ctx)).Where("id = ? AND member_id = ?", sourceId, memberId).Delete(ctx)
	if err != nil {
		return nil, err
	}
	if affected == 0 {
		return nil, common.NewServiceError(http.StatusBadRequest, response.Fail(response.CodeBadRequest).WithMessage("上传任务已取消"))
	}
	return ret, nil
}

// InvalidClean 清理无效的分片资源
func (s ResourceChunkService) InvalidClean(ctx context.Context) error {
	// 7 天前
	weekAgo := time.Now().AddDate(0, 0, -7)

	bucket := store.DefaultChunkStore()
	err := fs.WalkDir(bucket.FS(), ".", func(f string, d fs.DirEntry, err error) error {

		// 文件信息
		stat, err := d.Info()
		if err != nil {
			return err
		}

		// 文件最后修改时间为 7 天前
		if stat.ModTime().After(weekAgo) {
			return nil
		}

		// 忽略文件夹
		if d.IsDir() {
			// TODO 如果是空目录，则直接删除
			return nil
		}

		//// 相对路径
		//relPath, err := filepath.Rel(bucket.Name(), f)
		//if err != nil {
		//	return err
		//}

		localFilePath := filepath.ToSlash(f)

		// 检索文件是否存在
		resourceId, err := db.Transaction(ctx, func(ctx context.Context) (int64, error) {
			var resourceId int64
			// TODO FOR UPDATE
			return resourceId, db.Session(ctx).Raw("SELECT id FROM t_resource_chunk WHERE path = ?", localFilePath).Scan(&resourceId).Error
		})
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		// 文件不存在，则删除无效资源
		if resourceId == 0 {
			slog.InfoContext(ctx, "删除无效的分片文件",
				slog.String("path", localFilePath),
				slog.Time("modTime", stat.ModTime()),
			)
			if err := bucket.Remove(f); err != nil {
				slog.ErrorContext(ctx, "删除无效的分片文件异常",
					slog.String("err", err.Error()),
					slog.String("path", localFilePath),
				)
				return err
			}
		}
		return nil
	})
	return err
}

func NewResourceChunkService(resourceService *ResourceService) *ResourceChunkService {
	return &ResourceChunkService{resourceService: resourceService}
}

var DefaultResourceChunkService = NewResourceChunkService(DefaultResourceService)
