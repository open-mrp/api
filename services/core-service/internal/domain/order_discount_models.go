package domain

import (
	"time"

	"github.com/open-mrp/api/shared/pagination"
)

type OrderDiscount struct {
	ID               string
	Name             string `audit:"name"`
	Code             string `audit:"code"`
	Percentage       string `audit:"percentage"`
	Amount           string `audit:"amount"`
	DiscountTypeCode string `audit:"discount_type_code"`
	OrderCount       int32  `audit:"order_count"`
	AccountID        string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type ListOrderDiscountsParams struct {
	AccountID string
	Cursor    *string
	Limit     int32
	Query     *string
}

type ListOrderDiscountsResult struct {
	OrderDiscounts []*OrderDiscount
	PageInfo       pagination.PageInfo
}

type GetOrderDiscountParams struct {
	AccountID       string
	OrderDiscountID string
}

type CreateOrderDiscountParams struct {
	AccountID    string
	Name         string
	Code         string
	Percentage   *string
	Amount       *string
	DiscountType string
}

type UpdateOrderDiscountParams struct {
	AccountID       string
	OrderDiscountID string
	Name            *string
	Code            *string
	Percentage      *string
	Amount          *string
	DiscountType    *string
}

type DeleteOrderDiscountParams struct {
	AccountID       string
	OrderDiscountID string
}

type FindOrderDiscountByCodeParams struct {
	AccountID      string
	Code           string
	BuyerAccountID *string
	SalesOrderID   *string
}
