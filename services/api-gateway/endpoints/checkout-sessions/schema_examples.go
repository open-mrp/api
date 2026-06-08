package checkoutsessionep

import (
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/field"
)

func (*CreateCheckoutSessionRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(&CreateCheckoutSessionRequest{
		OrderID:         apiresource.SampleSalesOrderID,
		OrderNumber:     apiresource.SampleSalesOrderNumber,
		OrderTotalCents: 125050,
		CustomerPO:      field.Some("PO-4242"),
	})
}
