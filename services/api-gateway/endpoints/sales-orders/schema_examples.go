package salesorderep

import (
	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
)

func (*CheckoutSalesOrderRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(&CheckoutSalesOrderRequest{
		SalesOrderID: apiresource.SampleSalesOrderID,
		Email:        "operations@acme.example.com",
	})
}
