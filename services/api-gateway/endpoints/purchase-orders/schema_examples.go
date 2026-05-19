package purchaseorderep

import (
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apirequest "github.com/augno/api/services/api-gateway/pkg/request"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
)

func (*CreatePurchaseOrderLineRequest) SchemaExample() any {
	itemID := apiresource.SampleItemID
	desc := "6061-T6 Aluminum Sheet 4x8"
	return apiexample.ValidateAndMarshalToMap(&CreatePurchaseOrderLineRequest{
		PurchaseOrderID: apiresource.SamplePurchaseOrderDetailID,
		OrderLineInput: apirequest.OrderLineInput{
			ProductID:                  apiresource.SampleProductID,
			ItemID:                     &itemID,
			ProductSKU:                 apiresource.SampleItemSKU,
			ProductDescription:         &desc,
			QuantityValue:              "10",
			QuantityUnitID:             apiresource.SampleUnitID,
			UnitPriceValue:             apiresource.SampleRateValue,
			UnitPriceNumeratorUnitID:   apiresource.SampleUnitID,
			UnitPriceDenominatorUnitID: apiresource.SampleUnitID,
		},
	})
}
