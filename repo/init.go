package repo

import (
	"ispace/db"
	"ispace/repo/model"
)

func Initialization() error {
	//migrator := db.Get().Migrator()
	//if !db.Get().Migrator().HasTable(&model.Member{}) {
	//	if err := migrator.CreateTable(&model.Member{}); err != nil {
	//		return err
	//	}
	//}
	return db.Get().AutoMigrate(
		&model.Admin{},
		&model.Member{},
		&model.MemberDeletedQueue{},
		&model.Object{},
		&model.Resource{},
		&model.Share{},
		&model.ShareResource{},
		&model.RecycleBin{},
		&model.SysConfig{},
	)
}
