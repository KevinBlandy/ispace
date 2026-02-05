package page

import (
	"ispace/common/util"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

var (
	DefaultPage       int64 = 1
	DefaultRows       int64 = 20
	DefaultQueryTotal       = true
)

// orderFieldRegexp 排序字段的正则，避免 SQL 注入
var orderFieldRegexp = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

// Order 排序方向
type Order string

var (
	OrderAsc  Order = "ASC"
	OrderDesc Order = "DESC"
)

// Field 排序字段
type Field string

// Valid 是否是合法的排序字段
func (f Field) Valid() bool {
	return orderFieldRegexp.MatchString(string(f))
}

// Pagination 分页返回
type Pagination[T any] struct {
	Total int64 `json:"total"` // 总记录数量
	Rows  []T   `json:"rows"`  // 记录
}

// Sort 排序
type Sort struct {
	Field Field `json:"field"` // 排序字段
	Order Order `json:"order"` // 排序方式
}

// Pager 分页对象
type Pager struct {
	Total bool   `json:"total"` // 是否 count 查询
	Page  int    `json:"page"`  // 页码
	Rows  int    `json:"rows"`  // 每页书了
	Sort  []Sort `json:"sort"`
}

// NewPagerFromQuery 从查询参数中解析分页参数
// page=1&rows=10&sort=id&order=asc&sort=title&order=desc
func NewPagerFromQuery(query url.Values) *Pager {

	// 页码和分页数量
	page, _ := strconv.ParseInt(query.Get("page"), 10, 64)
	rows, _ := strconv.ParseInt(query.Get("rows"), 10, 64)
	if page <= 0 {
		page = DefaultPage
	}
	if rows <= 0 {
		rows = DefaultRows
	}

	// 排序
	sort := query["sort"]
	order := query["order"]

	var sorts []Sort

	for i, field := range sort {
		f := Field(field)
		if !f.Valid() { // 忽略非法字段
			continue
		}
		var o = OrderAsc // 默认升序
		if len(order) > i {
			if Order(strings.ToUpper(order[i])) == OrderDesc {
				o = OrderDesc
			}
		}
		sorts = append(sorts, Sort{Field: f, Order: o})
	}

	// 是否查询总数量
	total := util.BoolQuery(query, "total")
	if total == nil {
		total = &DefaultQueryTotal
	}

	return &Pager{
		Total: *total,
		Page:  int(page),
		Rows:  int(rows),
		Sort:  sorts,
	}
}

// Offset OffSet 值
func (p *Pager) Offset() int {
	return (p.Page - 1) * p.Rows
}
