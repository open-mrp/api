package apiresource

import (
	"time"

	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	"github.com/open-mrp/api/shared/constants"
)

// NewCustomer is a customer added in a report's period that has ordered, with its first order and lifetime sales.
type NewCustomer struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=new_customer"`
	// The customer's account ID.
	ID string `json:"id" validate:"required"`
	// The customer's number.
	Number string `json:"number"`
	// The customer's name: its alias, or its account's name when it has none.
	Name string `json:"name"`
	// Name of the customer's group, or null when it has none.
	CustomerGroupName *string `json:"customer_group_name"`
	// The default shipping address's locality and state, or null when it has neither.
	Location *string `json:"location"`
	// Name of the customer's default sales rep, or null when it has none.
	SalesRepName *string `json:"sales_rep_name"`
	// Lifetime sales: sales order lines priced above zero, outside the shipping and misc product lines. `value` is exact, unrounded.
	LifetimeRevenue *ComputedQuantity `json:"lifetime_revenue" validate:"required"`
	// When the customer's first such order was issued.
	FirstOrderedAt time.Time `json:"first_ordered_at" validate:"required"`
	// When the customer was added.
	AddedAt time.Time `json:"added_at" validate:"required"`
}

var sampleNewCustomerGroup = "Wholesale"
var sampleNewCustomerLocation = "Raleigh, NC"
var sampleNewCustomerRep = "Jordan Lee"

var SampleNewCustomer = &NewCustomer{
	Object:            constants.ObjectTypeNewCustomer,
	ID:                SampleCustomerID,
	Number:            "C1042",
	Name:              SampleCustomerName,
	CustomerGroupName: &sampleNewCustomerGroup,
	Location:          &sampleNewCustomerLocation,
	SalesRepName:      &sampleNewCustomerRep,
	LifetimeRevenue:   SampleSalesTotals.Revenue,
	FirstOrderedAt:    sampleSalesDay,
	AddedAt:           sampleSalesDay.AddDate(0, 0, -3),
}

func (*NewCustomer) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleNewCustomer)
}
