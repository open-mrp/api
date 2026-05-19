package salesorderep

import (
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
)

func (*CheckoutSalesOrderRequest) SchemaExample() any {
	ok := "https://dashboard.example.com/checkout/success"
	cancel := "https://dashboard.example.com/checkout/cancel"
	return apiexample.ValidateAndMarshalToMap(&CheckoutSalesOrderRequest{
		SalesOrderID: apiresource.SampleSalesOrderDetailID,
		Email:        "operations@acme.example.com",
		SuccessURL:   &ok,
		CancelURL:    &cancel,
	})
}
