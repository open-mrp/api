package constants

// SalesBreakdownGroupBy is the dimension invoiced sales are totalled by.
type SalesBreakdownGroupBy string

const (
	// SalesBreakdownGroupByCustomer totals by the buyer on the order.
	SalesBreakdownGroupByCustomer SalesBreakdownGroupBy = "customer"
	// SalesBreakdownGroupByProduct totals by the item sold.
	SalesBreakdownGroupByProduct SalesBreakdownGroupBy = "product"
	// SalesBreakdownGroupByProductLine totals by the product's product line.
	SalesBreakdownGroupByProductLine SalesBreakdownGroupBy = "product_line"
	// SalesBreakdownGroupByCustomerGroup totals by the buyer's current customer group; buyers in no group are left out.
	SalesBreakdownGroupByCustomerGroup SalesBreakdownGroupBy = "customer_group"
	// SalesBreakdownGroupBySalesRep totals by the order's sales rep; orders without one are left out.
	SalesBreakdownGroupBySalesRep SalesBreakdownGroupBy = "sales_rep"
	// SalesBreakdownGroupByDiscount totals by the order's discount; orders without one are left out.
	SalesBreakdownGroupByDiscount SalesBreakdownGroupBy = "discount"
)

func (m SalesBreakdownGroupBy) IsValid() bool {
	switch m {
	case SalesBreakdownGroupByCustomer, SalesBreakdownGroupByProduct, SalesBreakdownGroupByProductLine,
		SalesBreakdownGroupByCustomerGroup, SalesBreakdownGroupBySalesRep, SalesBreakdownGroupByDiscount:
		return true
	default:
		return false
	}
}

func (m SalesBreakdownGroupBy) EnumValues() []string {
	return []string{
		string(SalesBreakdownGroupByCustomer),
		string(SalesBreakdownGroupByProduct),
		string(SalesBreakdownGroupByProductLine),
		string(SalesBreakdownGroupByCustomerGroup),
		string(SalesBreakdownGroupBySalesRep),
		string(SalesBreakdownGroupByDiscount),
	}
}

func (m *SalesBreakdownGroupBy) StringPtr() *string {
	if m == nil {
		return nil
	}
	s := string(*m)
	return &s
}
