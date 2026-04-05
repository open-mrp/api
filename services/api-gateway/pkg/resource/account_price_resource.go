package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleAccountPriceID = "acpr_01jm4r6700f8nwq3v5hx2d9ktp"

// AccountPrice represents a customer-specific price for a product line.
type AccountPrice struct {
	// The unique identifier for the account price.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=account_price"`
	// The customer account this price applies to.
	RecipientAccount *Customer `json:"recipient_account" expandable:"true"`
	// The product line this price applies to.
	ProductLine *ProductLine `json:"product_line" expandable:"true"`
	// The rate (price per unit) for this account price.
	Rate *Rate `json:"rate" validate:"required"`
	// The item categories this price is constrained to.
	Categories *List[ItemCategory] `json:"categories" validate:"required" expandable:"true"`
	// The attributes this price is constrained to.
	Attributes *List[Attribute] `json:"attributes" validate:"required" expandable:"true"`
	// When this account price was created.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// When this account price was last updated.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var SampleAccountPrice = &AccountPrice{
	ID:     SampleAccountPriceID,
	Object: constants.ObjectTypeAccountPrice,
	RecipientAccount: &Customer{
		ID:     SampleCustomerID,
		Object: constants.ObjectTypeCustomer,
		Name:   SampleCustomerName,
	},
	ProductLine: SampleProductLine,
	Rate:        SampleRate,
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
