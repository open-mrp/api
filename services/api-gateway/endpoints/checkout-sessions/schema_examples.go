package checkoutsessionep

import (
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
)

func (*CreateCheckoutSessionRequest) SchemaExample() any {
	po := "PO-4242"
	return apiexample.ValidateAndMarshalToMap(&CreateCheckoutSessionRequest{
		OrderID:         apiresource.SampleSalesOrderDetailID,
		OrderNumber:     apiresource.SampleSalesOrderNumber,
		OrderTotalCents: 125050,
		CustomerPO:      &po,
	})
}
