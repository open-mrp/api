package purchaseorderep

import (
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apirequest "github.com/augno/api/services/api-gateway/pkg/request"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/field"
)

func (*CreatePurchaseOrderLineRequest) SchemaExample() any {
	itemID := apiresource.SampleItemID
	desc := "6061-T6 Aluminum Sheet 4x8"
	return apiexample.ValidateAndMarshalToMap(&CreatePurchaseOrderLineRequest{
		PurchaseOrderID: apiresource.SamplePurchaseOrderID,
		OrderLineInput: apirequest.OrderLineInput{
			ProductID:          apiresource.SampleProductID,
			ItemID:             field.Some(itemID),
			ProductSKU:         apiresource.SampleItemSKU,
			ProductDescription: field.Some(desc),
			Quantity:           apirequest.QuantityInput{Value: "10", UnitID: apiresource.SampleUnitID},
			UnitPrice:          apirequest.RateInput{Value: apiresource.SampleRateValue, NumeratorUnitID: apiresource.SampleUnitID, DenominatorUnitID: apiresource.SampleUnitID},
		},
	})
}
