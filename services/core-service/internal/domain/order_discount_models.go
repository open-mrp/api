package domain

import (
	"time"

	"github.com/open-mrp/apikit/pagination"

	"github.com/open-mrp/api/shared/constants"
)

// OrderDiscount is a code a buyer enters to take money off an order.
type OrderDiscount struct {
	ID       string
	Name     string                  `audit:"name"`
	Code     string                  `audit:"code"`
	Status   constants.ArchiveStatus `audit:"status"`
	Discount Discount                `audit:"discount"`
	// TimesRedeemed is how many orders the code has been applied to.
	TimesRedeemed int32 `audit:"-"`
	// Percentage (a fraction), Amount and DiscountTypeCode restate Discount in the legacy columns order totals read.
	Percentage       string `audit:"-"`
	Amount           string `audit:"-"`
	DiscountTypeCode string `audit:"-"`
	AccountID        string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type ListOrderDiscountsParams struct {
	AccountID string
	Statuses  []constants.ArchiveStatus
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
	AccountID string
	Name      string
	Code      string
	Discount  *Discount
	Status    *constants.ArchiveStatus
}

type UpdateOrderDiscountParams struct {
	AccountID       string
	OrderDiscountID string
	Name            *string
	Code            *string
	Discount        *Discount
	Status          *constants.ArchiveStatus
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
