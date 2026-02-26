package service

import (
	"context"
	"database/sql"
	"errors"
	"ispace/common"
	"ispace/common/concurrent"
	"ispace/common/constant"
	"ispace/common/id"
	"ispace/common/response"
	"ispace/common/util"
	"ispace/db"
	"ispace/repo/model"
	"ispace/web/handler/api"
	"net/http"
	"time"

	"gorm.io/gorm"
)

type SysConfigService struct {
	cache *concurrent.Map[model.SysConfigKey, *model.SysConfig]
}

// Remove 从缓存中移除
func (s *SysConfigService) Remove(ctx context.Context, key model.SysConfigKey) (*model.SysConfig, bool) {
	return s.cache.LoadAndDelete(key)
}

// Get 从缓存中读取
func (s *SysConfigService) Get(ctx context.Context, key model.SysConfigKey) *model.SysConfig {
	config, loaded := s.cache.Load(key)
	if !loaded {
		// TODO  并发问题
		var err error
		config, err = db.Transaction(ctx, func(ctx context.Context) (*model.SysConfig, error) {
			return gorm.G[*model.SysConfig](db.Session(ctx)).Where("key = ?", key).Take(ctx)
		}, db.TxReadOnly)
		if err != nil {
			// 不存在的情况下返回 null
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil
			}
			//slog.ErrorContext(ctx, "查询全局配置异常", slog.String("err", err.Error()))
			//return nil
			panic(err.Error())
		}
		s.cache.Store(key, config)
	}
	return config
}

// SaveOnNotExists 如果指定的配置项目不存在，则新增
func (s *SysConfigService) SaveOnNotExists(ctx context.Context, config *model.SysConfig) error {
	_, err := gorm.G[model.SysConfig](db.Session(ctx)).Where("key = ?", config.Key).Take(context.Background())
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return gorm.G[model.SysConfig](db.Session(ctx)).Create(ctx, config)
		}
		return err
	}
	return nil
}

func (s *SysConfigService) List(ctx context.Context) ([]*api.SysConfigListResponse, error) {
	list, err := gorm.G[*model.SysConfig](db.Session(ctx)).Find(ctx)
	if err != nil {
		return nil, err
	}
	var ret = make([]*api.SysConfigListResponse, 0)
	for _, config := range list {
		ret = append(ret, &api.SysConfigListResponse{
			Id:         config.Id,
			Key:        config.Key,
			Value:      config.Value,
			ValueType:  config.ValueType,
			Remark:     config.Remark,
			CreateTime: config.CreateTime,
			UpdateTime: config.UpdateTime,
		})
	}
	return ret, nil
}

// ExistsByKey 根据 KEY 检索记录是否存在，返回对应记录的 ID
func (s *SysConfigService) ExistsByKey(ctx context.Context, key model.SysConfigKey) (ret struct {
	Id  int64
	Key model.SysConfigKey
}, err error) {
	err = db.Session(ctx).Raw("SELECT id, key FROM sys_config WHERE key = ?", key).Row().Scan(&ret.Id, &ret.Key)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = nil // 无记录，不作为异常
		}
	}
	return
}

func (s *SysConfigService) Create(ctx context.Context, request *api.SysConfigCreateRequest) error {
	i, err := s.ExistsByKey(ctx, request.Key)
	if err != nil {
		return err
	}
	if i.Id != 0 {
		return common.NewServiceError(http.StatusBadRequest, response.Fail(response.CodeBadRequest).WithMessage("key 重复："+string(request.Key)))
	}

	now := util.ContextValueDefault(ctx, constant.CtxKeyRequestTime, time.Now()).UnixMilli()

	return gorm.G[model.SysConfig](db.Session(ctx)).Create(ctx, &model.SysConfig{
		Id:         id.Next().Int64(),
		Key:        request.Key,
		Value:      request.Value,
		ValueType:  request.ValueType,
		Remark:     request.Remark,
		CreateTime: now,
		UpdateTime: now,
	})
}

func (s *SysConfigService) Update(ctx context.Context, request *api.SysConfigUpdateRequest) (*model.SysConfig, error) {
	i, err := s.ExistsByKey(ctx, request.Key)
	if err != nil {
		return nil, err
	}
	if i.Id != 0 && i.Id != request.Id {
		return nil, common.NewServiceError(http.StatusBadRequest, response.Fail(response.CodeBadRequest).WithMessage("key 重复："+string(request.Key)))
	}

	// 检索旧的记录
	beforeConfig, err := gorm.G[*model.SysConfig](db.Session(ctx)).Where("id = ?", request.Id).Take(ctx)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if beforeConfig == nil || beforeConfig.Id == 0 {
		return nil, common.NewServiceError(http.StatusBadRequest, response.Fail(response.CodeBadRequest).WithMessage("更新记录不存在"))
	}

	now := util.ContextValueDefault(ctx, constant.CtxKeyRequestTime, time.Now()).UnixMilli()
	result := db.Session(ctx).
		Table(model.SysConfig{}.TableName()).
		Where("id = ?", request.Id).
		UpdateColumns(map[string]any{
			"key":         request.Key,
			"value":       request.Value,
			"value_type":  request.ValueType,
			"remark":      request.Remark,
			"update_time": now,
		})
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, common.NewServiceError(http.StatusBadRequest, response.Fail(response.CodeBadRequest).WithMessage("更新失败，请刷新后重试"))
	}
	return beforeConfig, nil
}

func (s *SysConfigService) Delete(ctx context.Context, request *api.SysConfigDeleteRequest) ([]*model.SysConfig, error) {
	list, err := gorm.G[*model.SysConfig](db.Session(ctx)).Where("id IN ?", request.Id).Find(ctx)
	if err != nil {
		return nil, err
	}
	if len(list) != len(request.Id) {
		return nil, common.NewServiceError(http.StatusBadRequest, response.Fail(response.CodeBadRequest).WithMessage("数据错误"))
	}
	affected, err := gorm.G[*model.SysConfig](db.Session(ctx)).Where("id IN ?", request.Id).Delete(ctx)
	if err != nil {
		return nil, err
	}
	if affected != len(request.Id) {
		return nil, common.NewServiceError(http.StatusBadRequest, response.Fail(response.CodeBadRequest).WithMessage("删除失败，刷新后重试"))
	}
	return list, nil
}

var DefaultSysConfigService = &SysConfigService{
	cache: concurrent.NewMap[model.SysConfigKey, *model.SysConfig](),
}
