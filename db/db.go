package db

import (
	"context"
	"database/sql"
	"ispace/config"
	"log/slog"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Database 数据源
var db *gorm.DB

// Initialization 初始化数据源
func Initialization() (err error) {

	//logger.RecorderParamsFilter = func(ctx context.Context, sql string, params ...interface{}) (string, []interface{}) {
	//	sql = strings.NewReplacer(
	//		"\r\n", " ",
	//		"\n", " ",
	//		"\t", " ",
	//	).Replace(sql)
	//
	//	// 可选：压缩多余空格
	//	return strings.Join(strings.Fields(sql), " "), params
	//}

	db, err = gorm.Open(sqlite.Open(*config.DB), &gorm.Config{
		Logger: logger.NewSlogLogger(slog.Default(), logger.Config{
			SlowThreshold:             time.Second * 2,
			Colorful:                  false,
			IgnoreRecordNotFoundError: false,
			ParameterizedQueries:      false,
			LogLevel:                  logger.Info,
		}),
	})
	if err != nil {
		return err
	}

	database, err := db.DB()
	if err != nil {
		return err
	}
	//database.SetMaxIdleConns(*config.DBPoolMaxIdleConn)
	//database.SetMaxOpenConns(*config.DBPoolMaxOpenConn)
	//database.SetConnMaxIdleTime(*config.DBPoolMaxIdleTime)
	//database.SetConnMaxLifetime(*config.DBPoolMaxLifetime)

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

func Close() error {
	rawDB, err := db.DB()
	if err != nil {
		return err
	}
	return rawDB.Close()
}
