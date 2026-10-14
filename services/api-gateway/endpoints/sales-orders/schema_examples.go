package salesorderep

import (
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	apiexample "github.com/open-mrp/apikit/example"
)

func (*CheckoutSalesOrderRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(&CheckoutSalesOrderRequest{
		SalesOrderID: apiresource.SampleSalesOrderID,
		Email:        "operations@acme.example.com",
	})
}
