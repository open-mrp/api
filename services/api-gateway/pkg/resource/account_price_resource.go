package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleAccountPriceID = "acpr_01jm4r6700f8nwq3v5hx2d9ktp"

// Customer-specific price for a product line.
type AccountPrice struct {
	// Account price ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=account_price"`
	// Customer account this price applies to.
	RecipientAccount *Customer `json:"recipient_account" expandable:"true"`
	// Product line this price applies to.
	ProductLine *ProductLine `json:"product_line" expandable:"true"`
	// Rate (price per unit).
	Rate *Rate `json:"rate" validate:"required"`
	// Item categories this price is constrained to.
	Categories *List[ItemCategory] `json:"categories" validate:"required" expandable:"true"`
	// Attributes this price is constrained to.
	Attributes *List[Attribute] `json:"attributes" validate:"required" expandable:"true"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
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
