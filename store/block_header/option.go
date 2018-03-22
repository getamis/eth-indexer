package blockheader

import (
	"fmt"

	"github.com/jinzhu/gorm"
)

type OrderType string

const (
	ORDER_ASC  = OrderType("asc")
	ORDER_DESC = OrderType("desc")
)

var (
	ErrRecordNotFound = gorm.ErrRecordNotFound
)

type QueryOption struct {
	OrderBy string
	Order   OrderType
	Limit   int
	Page    int
	Since   string
	Until   string
}

func (o *QueryOption) OrderString() string {
	if o.OrderBy == "" {
		return ""
	}
	if string(o.Order) == "" {
		o.Order = ORDER_ASC
	}
	return fmt.Sprintf("%s %s", o.OrderBy, o.Order)
}
