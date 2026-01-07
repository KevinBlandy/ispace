package repo

import (
	"context"
	"ispace/db"
	"ispace/repo/model"
)

type ObjectRepo struct{}

func NewObjectRepo() *ObjectRepo {
	return &ObjectRepo{}
}

// GetIdByHash 根据 Hash 获取对象 ID
func (repo *ObjectRepo) GetIdByHash(ctx context.Context, hash string) (int64, error) {
	var id int64
	return id, db.Session(ctx).Table(model.Object{}.TableName()).Select("id").Where("hash = ?", hash).Scan(&id).Error
}

var DefaultObjectRepo = NewObjectRepo()
