package db

import (
	"context"
	"database/sql"
	"ispace/common/page"
	"ispace/common/util"
	"math"
	"strconv"
)

// Count 总数量查询
func Count(ctx context.Context, query string, args []any, dest any) error {
	return Session(ctx).Raw("SELECT COUNT(1) FROM ("+query+")", args...).Scan(dest).Error
}

// PageQuery 分页查询，自动映射结构体
func PageQuery[T any](
	ctx context.Context,
	pager *page.Pager,
	query string,
	args []any) (*page.Pagination[*T], error) {

	session := Session(ctx)

	return pageQuery(ctx, pager, query, args, func(rows *sql.Rows) ([]*T, error) {
		var results []*T
		for rows.Next() {
			var item T
			if err := session.ScanRows(rows, &item); err != nil {
				return nil, err
			}
			results = append(results, &item)
		}
		return results, nil
	})
}

// PageQueryScan 分页查询，手动映射数据
func PageQueryScan[T any](
	ctx context.Context,
	pager *page.Pager,
	query string,
	args []any,
	mapper func(rows *sql.Rows) (T, error),
) (*page.Pagination[T], error) {

	return pageQuery[T](ctx, pager, query, args, func(rows *sql.Rows) ([]T, error) {
		var items []T
		for rows.Next() {
			m, err := mapper(rows)
			if err != nil {
				return nil, err
			}
			items = append(items, m)
		}
		return items, nil
	})
}

// pageQuery 分页查询
func pageQuery[T any](
	ctx context.Context,
	pager *page.Pager,
	query string,
	args []any,
	rowsHandler func(row *sql.Rows) ([]T, error),
) (*page.Pagination[T], error) {

	var result = new(page.Pagination[T])

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

	// 结果集处理
	result.Rows, err = rowsHandler(rows)
	if err != nil {
		return nil, err
	}
	return result, nil
}
