package ldbl

import (
	"fmt"
	"strings"
)

type OrderDirection string

const (
	ASC  OrderDirection = "ASC"
	DESC                = "DESC"
)

type Orderer interface {
	OrderString() string
}

type Order struct {
	Field     string
	Direction OrderDirection
}

type CombinedOrder struct {
	orders []Order
}

func OrderBy(field string, dir OrderDirection) *CombinedOrder {
	return &CombinedOrder{[]Order{Order{field, dir}}}
}

func (o *CombinedOrder) Then(field string, dir OrderDirection) *CombinedOrder {
	orders := append(o.orders, Order{field, dir})
	return &CombinedOrder{orders}
}

func (o Order) OrderString() string {
	return fmt.Sprintf("%s %s", o.Field, string(o.Direction))
}

func (o *CombinedOrder) OrderString() string {
	sqls := make([]string, 0, len(o.orders))
	for _, order := range o.orders {
		sqls = append(sqls, order.OrderString())
	}
	return strings.Join(sqls, ", ")
}
