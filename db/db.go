package db

import (
	"context"
	"database/sql"
	"ispace/config"
	"log/slog"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Database 数据源
var db *gorm.DB

// Initialization 初始化数据源
func Initialization(config *config.DataSource) (err error) {

	slog.Info("[db] 数据源初始化")

	db, err = gorm.Open(mysql.Open(config.Url), &gorm.Config{
		Logger: logger.NewSlogLogger(slog.Default(), logger.Config{
			SlowThreshold:             time.Second * 2,
			Colorful:                  false,
			IgnoreRecordNotFoundError: false,
			ParameterizedQueries:      false,
			LogLevel:                  logger.Warn,
		}),
	})
	if err != nil {
		return err
	}

	database, err := db.DB()
	if err != nil {
		return err
	}
	database.SetMaxIdleConns(config.Pool.MaxIdleConn)
	database.SetMaxOpenConns(config.Pool.MaxOpenConn)
	database.SetConnMaxIdleTime(config.Pool.MaxIdleTime)
	database.SetConnMaxLifetime(config.Pool.MaxLifetime)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	return database.PingContext(ctx)
}

// Stats 数据库统计
func Stats() sql.DBStats {
	if db == nil {
		return sql.DBStats{}
	}
	rawDB, err := db.DB()
	if err != nil {
		return sql.DBStats{}
	}
	return rawDB.Stats()
}

func Get() *gorm.DB {
	return db
}
