package service

import (
	"context"
	"errors"
	"ispace/db"
	"ispace/repo/model"

	"gorm.io/gorm"
)

type ObjectService struct{}

func NewObjectService() *ObjectService {
	return &ObjectService{}
}

// Exists 根据 column = value 判断记录是否存在
func (o *ObjectService) Exists(ctx context.Context, column string, value any) (id int64, err error) {
	err = db.Session(ctx).Table(model.Object{}.TableName()).Select("id").Where(column+" = ?", value).Scan(&id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, nil
	}
	return
}

var DefaultObjectService = NewObjectService()
