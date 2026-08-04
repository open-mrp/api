package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleAccountPriceID = "acpr_7l4j483kf32p"

// A customer-specific price for a product line.
//
// When a sales order line matches an account price, that price replaces the unit price the line would otherwise be given — including the effect of any volume discount — rather than discounting it. If more than one account price matches a line, the most recently created one wins.
type AccountPrice struct {
	// Account price ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=account_price"`
	// The customer this price is offered to.
	//
	// A price recorded against a parent customer account also applies to orders placed by its child accounts.
	RecipientAccount *Customer `json:"recipient_account" expandable:"true"`
	// The product line whose products this price applies to.
	//
	// A product that is not assigned to a product line never matches an account price.
	ProductLine *ProductLine `json:"product_line" expandable:"true"`
	// The price, expressed as a rate.
	//
	// The rate's numerator unit is typically a currency and its denominator unit is the quantity unit being priced (e.g. `$25.50 / kg`). A matching order line takes both its unit price and its price units from this rate, exactly as entered.
	Rate *Rate `json:"rate" validate:"required"`
	// Item categories recorded on this price.
	//
	// Order pricing matches an account price on its product line and attributes only, so categories recorded here do not narrow which products the price applies to.
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
