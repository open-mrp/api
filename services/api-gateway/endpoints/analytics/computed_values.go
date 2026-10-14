package analyticsep

import (
	"github.com/shopspring/decimal"

	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
)

// analyticsPrice renders a computed price per perUnitID, to four decimal places. Its unit loads before the response is written.
func analyticsPrice(value, perUnitID string) *apiresource.UnitPrice {
	return apiresource.NewUnitPrice(fixed4(value), perUnitID)
}

// analyticsQuantity renders a computed quantity in unitID, to four decimal places. Its unit loads before the response is written.
func analyticsQuantity(value, unitID string) *apiresource.Quantity {
	return apiresource.NewQuantity(fixed4(value), unitID)
}

func fixed4(value string) string {
	amount, err := decimal.NewFromString(value)
	if err != nil {
		amount = decimal.Zero
	}
	return amount.StringFixed(4)
}
