package service

import (
	"context"
	"errors"
	"ispace/common/concurrent"
	"ispace/db"
	"ispace/repo/model"
	"log/slog"

	"gorm.io/gorm"
)

type SysConfigService struct {
	cache *concurrent.Map[model.SysConfigKey, *model.SysConfig]
}

// Get 从缓存中读取全局配置
func (s *SysConfigService) Get(ctx context.Context, key model.SysConfigKey) *model.SysConfig {
	config, loaded := s.cache.Load(key)
	if !loaded {
		var err error
		config, err = db.Transaction(ctx, func(ctx context.Context) (*model.SysConfig, error) {
			return gorm.G[*model.SysConfig](db.Session(ctx)).Where("key = ?", key).Take(ctx)
		}, db.TxReadOnly)
		if err != nil {
			slog.ErrorContext(ctx, "查询全局配置异常", slog.String("err", err.Error()))
			return nil
		}
		s.cache.Store(key, config)
	}
	return config
}

// SaveOnNotExists 设置配置
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

//// ReloadCache 数据库加载最新缓存
//func (s *SysConfigService) ReloadCache(ctx context.Context) error {
//	configs, err := gorm.G[*model.SysConfig](db.Session(ctx)).Find(ctx)
//	if err != nil {
//		return err
//	}
//	var cache = concurrent.NewMap[model.SysConfigKey, *model.SysConfig]()
//	for _, config := range configs {
//		cache.Store(config.Key, config)
//	}
//	s.cache = cache
//	return nil
//}

var DefaultSysConfigService = &SysConfigService{
	cache: concurrent.NewMap[model.SysConfigKey, *model.SysConfig](),
}
