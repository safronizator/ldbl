package ldbl

import (
	"fmt"
)

type SqlQuery struct {
	what      Loadable
	condition string
	args      []interface{}
	order     Orderer
	limit     int
	offset    int
}

//TODO: Joins

type SqlQueryBilder interface {
	ItemToLoad() Loadable
	Query() string
	Args() []interface{}
}

func Select(what Loadable) *SqlQuery {
	return &SqlQuery{what: what, condition: "1", limit: -1}
}

func (q *SqlQuery) ItemToLoad() Loadable {
	return q.what
}

func (q *SqlQuery) Query() string {
	orderSql := ""
	if q.order != nil {
		orderSql = "ORDER BY " + q.order.OrderString()
	}
	limitSql := ""
	if (q.offset > 0) || (q.limit > 0) {
		limitSql = fmt.Sprintf("LIMIT %d, %d", q.offset, q.limit)
	}
	return fmt.Sprintf("SELECT * FROM `%s` WHERE %s %s %s", q.what.CollectionName(), q.condition, orderSql, limitSql)
}

func (q *SqlQuery) Args() []interface{} {
	return q.args
}

func (q *SqlQuery) Where(condition string, args ...interface{}) *SqlQuery {
	q.condition = condition
	q.args = args
	return q
}

func (q *SqlQuery) OrderBy(field string, dir OrderDirection) *SqlQuery {
	if q.order == nil {
		q.order = OrderBy(field, dir)
	} else if combined, ok := q.order.(*CombinedOrder); ok {
		q.order = combined.Then(field, dir)
	} else if simple, ok := q.order.(Order); ok {
		q.order = OrderBy(simple.Field, simple.Direction).Then(field, dir)
	} else {
		q.order = Order{field, dir}
	}
	return q
}

func (q *SqlQuery) Order(order Orderer) *SqlQuery {
	q.order = order
	return q
}

func (q *SqlQuery) Limit(count int) *SqlQuery {
	q.limit = count
	return q
}

func (q *SqlQuery) Offset(from int) *SqlQuery {
	q.offset = from
	return q
}
