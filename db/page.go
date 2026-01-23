package db

import (
	"context"
	"ispace/common/page"
	"ispace/common/util"
	"math"
	"strconv"
)

// Count 总数量查询
func Count(ctx context.Context, query string, args []any, dest any) error {
	return Session(ctx).Raw("SELECT COUNT(1) FROM ("+query+")", args...).Scan(dest).Error
}

// PageQuery 分页查询
func PageQuery[T any](
	ctx context.Context,
	pager *page.Pager,
	query string,
	args []any) (*page.Pagination[*T], error) {

	var result = new(page.Pagination[*T])

	// 查询总数量
	if pager.Total {
		if err := Count(ctx, query, args, &result.Total); err != nil {
			return result, err
		}
		// 总数量为0或者是页码不合法
		if result.Total == 0 || (pager.Rows > 0 && pager.Page > int(math.Ceil(float64(result.Total)/float64(pager.Rows)))) {
			return result, nil
		}
	}

	//selectSql := "SELECT `_tmp`.* FROM (" + query + ") AS `_tmp`"

	// 查询记录
	selectSql := query

	// 排序
	if len(pager.Sort) > 0 {

		selectSql += " ORDER BY "

		for i, sort := range pager.Sort {
			if i != 0 {
				selectSql += ", "
			}
			selectSql += string(sort.Field) + " " + string(sort.Order)
		}
	}
	if pager.Rows > 0 {
		selectSql += " LIMIT " + strconv.Itoa(pager.Rows)
	}
	if pager.Page > 0 {
		selectSql += " OFFSET " + strconv.Itoa(pager.Offset())
	}

	// 执行查询
	session := Session(ctx)
	rows, err := session.Raw(selectSql, args...).Rows()
	if err != nil {
		return result, err
	}
	defer util.SafeClose(rows)

	for rows.Next() {
		var item T
		if err := session.ScanRows(rows, &item); err != nil {
			return result, err
		}
		result.Rows = append(result.Rows, &item)
	}
	return result, nil
}
