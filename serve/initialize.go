package serve

import (
	"context"
	"ispace/common/id"
	"ispace/db"
	"ispace/repo"
	"ispace/repo/model"
	"ispace/web/service"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// initialization 系统资源初始化
func initialization() error {

	// 数据源
	if err := db.Initialization(); err != nil {
		return err
	}

	// 数据表
	if err := repo.Initialization(); err != nil {
		return err
	}
	//// redis
	//if err := rdb.Initialization(); err != nil {
	//	return err
	//}

	// 初始化 root 账户
	if err := rootAccount(); err != nil {
		return err
	}

	// 初始化基本配置
	if err := baseSysConfig(); err != nil {
		return err
	}
	return nil
}

// rootAccount root 账户初始化
func rootAccount() error {
	return db.TransactionWithOutResult(context.Background(), func(ctx context.Context) error {
		result, err := gorm.G[model.Admin](db.Session(ctx)).Count(ctx, "id")
		if err != nil {
			return err
		}
		if result != 0 {
			return nil
		}

		// 无 root 用户，进行初始化
		defaultAccount := "root"
		defaultPassword := "root"

		now := time.Now().UnixMilli()

		password, err := bcrypt.GenerateFromPassword([]byte(defaultPassword), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		return gorm.G[model.Admin](db.Session(ctx)).Create(ctx, &model.Admin{
			Id:         id.Next().Int64(),
			Avatar:     "",
			Account:    defaultAccount,
			Password:   string(password),
			Email:      "",
			Enabled:    true,
			CreateTime: now,
			UpdateTime: now,
		})
	})
}

// baseSysConfig 基本的系统配置信息初始化
func baseSysConfig() error {
	return db.TransactionWithOutResult(context.Background(), func(ctx context.Context) (err error) {

		var now = time.Now().UnixMilli()

		// Session 签名密钥
		if err = service.DefaultSysConfigService.SaveOnNotExists(ctx, &model.SysConfig{
			Id:         id.Next().Int64(),
			Key:        model.SysConfigKeySessionSecret,
			Value:      id.UUID(),
			ValueType:  model.SysConfigValueTypeString,
			Remark:     "Session 签名密钥",
			CreateTime: now,
			UpdateTime: now,
		}); err != nil {
			return
		}

		// Session 过期时间
		err = service.DefaultSysConfigService.SaveOnNotExists(ctx, &model.SysConfig{
			Id:         id.Next().Int64(),
			Key:        model.SysConfigKeySessionExpire,
			Value:      "168h", // 默认 24 * 7 小时
			ValueType:  model.SysConfigValueTypeDuration,
			Remark:     "Session 过期时间",
			CreateTime: now,
			UpdateTime: now,
		})
		return
	})
}
