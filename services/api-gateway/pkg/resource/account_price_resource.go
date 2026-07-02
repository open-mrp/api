package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleAccountPriceID = "acpr_01dfc47cc46b1e0b66ca8eec0a"

// A customer-specific price for a product line.
//
// When an order line matches an account price's product line and constraints, the account price replaces the standard product line pricing for the recipient customer.
type AccountPrice struct {
	// Account price ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=account_price"`
	// Customer account this price applies to.
	RecipientAccount *Customer `json:"recipient_account" expandable:"true"`
	// Product line this price applies to.
	ProductLine *ProductLine `json:"product_line" expandable:"true"`
	// The price, expressed as a rate.
	//
	// The rate's numerator unit is typically a currency and its denominator unit is the quantity unit being priced (e.g. `$25.50 / kg`).
	Rate *Rate `json:"rate" validate:"required"`
	// Item categories this price is constrained to.
	//
	// When empty, the price is not restricted by item category.
	Categories *List[ItemCategory] `json:"categories" validate:"required" expandable:"true"`
	// Attributes this price is constrained to.
	//
	// When set, the price applies only to items that have every listed attribute; when empty, attributes are not considered.
	Attributes *List[Attribute] `json:"attributes" validate:"required" expandable:"true"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var SampleAccountPrice = &AccountPrice{
	ID:               SampleAccountPriceID,
	Object:           constants.ObjectTypeAccountPrice,
	RecipientAccount: SampleCustomer,
	ProductLine:      SampleProductLine,
	Rate:             SampleRate,
	Categories: NewList([]ItemCategory{
		{
			ID:     SampleItemCategoryID,
			Object: constants.ObjectTypeItemCategory,
			Name:   SampleItemCategoryName,
		},
	}, PageInfo{}),
	Attributes: NewList([]Attribute{
		{
			ID:     SampleAttributeID,
			Object: constants.ObjectTypeAttribute,
			Value:  SampleAttributeValue,
		},
	}, PageInfo{}),
	CreatedAt: timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt: timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*AccountPrice) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleAccountPrice)
}
