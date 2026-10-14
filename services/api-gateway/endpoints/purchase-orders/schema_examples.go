package purchaseorderep

import (
	apirequest "github.com/open-mrp/api/services/api-gateway/pkg/request"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	apiexample "github.com/open-mrp/apikit/example"
	"github.com/open-mrp/apikit/field"
)

func (*CreatePurchaseOrderLineRequest) SchemaExample() any {
	itemID := apiresource.SampleItemID
	desc := "6061-T6 Aluminum Sheet 4x8"
	return apiexample.ValidateAndMarshalToMap(&CreatePurchaseOrderLineRequest{
		PurchaseOrderID: apiresource.SamplePurchaseOrderID,
		OrderLineInput: apirequest.OrderLineInput{
			ItemID:             field.Some(itemID),
			ProductSKU:         apiresource.SampleItemSKU,
			ProductDescription: field.Some(desc),
			Quantity:           apirequest.QuantityInput{Value: "10", UnitID: apiresource.SampleUnitID},
			UnitPrice:          apirequest.UnitPriceInput{Amount: "25.50", Currency: "usd", PerUnitID: apiresource.SampleUnitID},
		},
	})
}
