package db

import (
	"context"
	"ispace/common/util"
)

// List 检索数据集合
func List[T any](ctx context.Context, query string, args ...any) ([]*T, error) {

	session := Session(ctx)

	rows, err := session.Raw(query, args...).Rows()
	if err != nil {
		return nil, err
	}
	defer util.SafeClose(rows)

	entries := make([]*T, 0)

	for rows.Next() {
		var item T
		if err := session.ScanRows(rows, &item); err != nil {
			return nil, err
		}
		entries = append(entries, &item)
	}
	return entries, nil
}
