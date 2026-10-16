package constants

// SalesOrderStatusCode represents the status code of a sales order.
type SalesOrderStatusCode string

const (
	// SalesOrderStatusCodeEstimate indicates the order is an estimate.
	SalesOrderStatusCodeEstimate SalesOrderStatusCode = "estimate"
	// SalesOrderStatusCodeIssued indicates the order has been issued.
	SalesOrderStatusCodeIssued SalesOrderStatusCode = "issued"
	// SalesOrderStatusCodeFulfilled indicates the order has been fulfilled.
	SalesOrderStatusCodeFulfilled SalesOrderStatusCode = "fulfilled"
	// SalesOrderStatusCodeCanceled indicates the order was stopped before anything shipped.
	SalesOrderStatusCodeCanceled SalesOrderStatusCode = "canceled"
)

func (m SalesOrderStatusCode) IsValid() bool {
	switch m {
	case SalesOrderStatusCodeEstimate, SalesOrderStatusCodeIssued, SalesOrderStatusCodeFulfilled, SalesOrderStatusCodeCanceled:
		return true
	default:
		return false
	}
}

func (m SalesOrderStatusCode) EnumValues() []string {
	return []string{
		string(SalesOrderStatusCodeEstimate),
		string(SalesOrderStatusCodeIssued),
		string(SalesOrderStatusCodeFulfilled),
		string(SalesOrderStatusCodeCanceled),
	}
}

// SalesOrderStatusChange represents a status change action for a sales order.
type SalesOrderStatusChange string

const (
	// SalesOrderStatusChangeIssue transitions an order from estimate to issued.
	SalesOrderStatusChangeIssue SalesOrderStatusChange = "issue"
	// SalesOrderStatusChangeClose transitions an order from issued to fulfilled.
	SalesOrderStatusChangeClose SalesOrderStatusChange = "close"
	// SalesOrderStatusChangeUnissue transitions an order from issued to estimate.
	SalesOrderStatusChangeUnissue SalesOrderStatusChange = "unissue"
	// SalesOrderStatusChangeOpen transitions an order from fulfilled to issued.
	SalesOrderStatusChangeOpen SalesOrderStatusChange = "open"
	// SalesOrderStatusChangeCancel stops an issued order: canceled when nothing shipped, else fulfilled.
	SalesOrderStatusChangeCancel SalesOrderStatusChange = "cancel"
)

func (m SalesOrderStatusChange) IsValid() bool {
	switch m {
	case SalesOrderStatusChangeIssue, SalesOrderStatusChangeClose, SalesOrderStatusChangeUnissue, SalesOrderStatusChangeOpen, SalesOrderStatusChangeCancel:
		return true
	default:
		return false
	}
}

func (m SalesOrderStatusChange) EnumValues() []string {
	return []string{
		string(SalesOrderStatusChangeIssue),
		string(SalesOrderStatusChangeClose),
		string(SalesOrderStatusChangeUnissue),
		string(SalesOrderStatusChangeOpen),
		string(SalesOrderStatusChangeCancel),
	}
}

// SalesOrderPaymentStatus represents the payment state of a sales order.
type SalesOrderPaymentStatus string

const (
	// SalesOrderPaymentStatusUnpaid indicates no payment has been received.
	SalesOrderPaymentStatusUnpaid SalesOrderPaymentStatus = "unpaid"
	// SalesOrderPaymentStatusPartiallyPaid indicates the order is partially paid.
	SalesOrderPaymentStatusPartiallyPaid SalesOrderPaymentStatus = "partially_paid"
	// SalesOrderPaymentStatusPaid indicates the order is paid in full.
	SalesOrderPaymentStatusPaid SalesOrderPaymentStatus = "paid"
)

func (m SalesOrderPaymentStatus) IsValid() bool {
	switch m {
	case SalesOrderPaymentStatusUnpaid, SalesOrderPaymentStatusPartiallyPaid, SalesOrderPaymentStatusPaid:
		return true
	default:
		return false
	}
}

func (m SalesOrderPaymentStatus) EnumValues() []string {
	return []string{
		string(SalesOrderPaymentStatusUnpaid),
		string(SalesOrderPaymentStatusPartiallyPaid),
		string(SalesOrderPaymentStatusPaid),
	}
}

// OrderDiscountType represents the type of discount applied to an order.
type OrderDiscountType string

const (
	// OrderDiscountTypePercentage indicates a percentage-based discount.
	OrderDiscountTypePercentage OrderDiscountType = "percentage"
	// OrderDiscountTypeAmount indicates a fixed-amount discount.
	OrderDiscountTypeAmount OrderDiscountType = "amount"
)

func (m OrderDiscountType) IsValid() bool {
	switch m {
	case OrderDiscountTypePercentage, OrderDiscountTypeAmount:
		return true
	default:
		return false
	}
}

func (m OrderDiscountType) EnumValues() []string {
	return []string{
		string(OrderDiscountTypePercentage),
		string(OrderDiscountTypeAmount),
	}
}
