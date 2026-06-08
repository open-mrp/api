package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleShippingCaseID = "shcs_01207a101ea1475c687a39cf76"

// Physical shipping case within a shipment.
type ShippingCase struct {
	// Shipping case ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=shipping_case"`
	// Human-readable case number.
	Number string `json:"number" validate:"required"`
	// Serial Shipping Container Code.
	SSCC *string `json:"sscc"`
	// Carrier tracking number.
	TrackingNumber *string `json:"tracking_number"`
	// Shipped timestamp.
	ShippedAt *time.Time `json:"shipped_at"`
	// Freight amount.
	FreightAmount *Quantity `json:"freight_amount" expandable:"true"`
	// Freight weight.
	FreightWeight *Quantity `json:"freight_weight" expandable:"true"`
	// Associated shipment.
	Shipment *Shipment `json:"shipment" expandable:"true"`
	// Carrier.
	Carrier *Carrier `json:"carrier" expandable:"true"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

// Shipping case label URL.
type ShippingCaseLabelURL struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=shipping_case_label_url"`
	// Presigned label URL, or null if no label exists.
	URL *string `json:"url"`
}

var SampleShippingCase = &ShippingCase{
	ID:     SampleShippingCaseID,
	Object: constants.ObjectTypeShippingCase,
	Number: "SC-0001",
	FreightAmount: &Quantity{
		ID:           SampleQuantityID,
		Object:       constants.ObjectTypeQuantity,
		Value:        "12.500000000000000000000000000000",
		DisplayValue: "$12.50",
		Unit: &Unit{
			ID:           SampleUnitID,
			Object:       constants.ObjectTypeUnit,
			Name:         "US Dollar",
			Abbreviation: "$",
			Type:         constants.UnitTypeCurrency,
		},
	},
	FreightWeight: &Quantity{
		ID:           SampleQuantityID,
		Object:       constants.ObjectTypeQuantity,
		Value:        "5.000000000000000000000000000000",
		DisplayValue: "5 lb",
		Unit: &Unit{
			ID:           SampleUnitID,
			Object:       constants.ObjectTypeUnit,
			Name:         "Pound",
			Abbreviation: "lb",
			Type:         constants.UnitTypeMass,
		},
	},
	Shipment:  nil,
	Carrier:   nil,
	CreatedAt: timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt: timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*ShippingCase) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleShippingCase)
}

var SampleShippingCaseLabelURL = &ShippingCaseLabelURL{
	Object: constants.ObjectTypeShippingCaseLabelURL,
	URL:    nil,
}

func (*ShippingCaseLabelURL) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleShippingCaseLabelURL)
}
