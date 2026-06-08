package salesorderep

import (
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/field"
)

func (*CheckoutSalesOrderRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(&CheckoutSalesOrderRequest{
		SalesOrderID: apiresource.SampleSalesOrderID,
		Email:        "operations@acme.example.com",
		SuccessURL:   field.Some("https://dashboard.example.com/checkout/success"),
		CancelURL:    field.Some("https://dashboard.example.com/checkout/cancel"),
	})
}
