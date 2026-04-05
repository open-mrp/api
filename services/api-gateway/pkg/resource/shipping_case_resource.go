package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleShippingCaseID = "shcs_01jm4r6700f8nwq3v5hx2d9ktp"

// ShippingCase represents a physical shipping case within a shipment.
type ShippingCase struct {
	// The unique identifier for the shipping case.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=shipping_case"`
	// The human-readable case number.
	Number string `json:"number" validate:"required"`
	// The Serial Shipping Container Code.
	SSCC *string `json:"sscc"`
	// The carrier tracking number for this case.
	TrackingNumber *string `json:"tracking_number"`
	// The timestamp when the case was shipped.
	ShippedAt *time.Time `json:"shipped_at"`
	// The freight amount for this case.
	FreightAmount *Quantity `json:"freight_amount" expandable:"true"`
	// The freight weight for this case.
	FreightWeight *Quantity `json:"freight_weight" expandable:"true"`
	// The shipment this case belongs to.
	Shipment *ShipmentDetail `json:"shipment" expandable:"true"`
	// The carrier for this case.
	Carrier *Carrier `json:"carrier" expandable:"true"`
	// The timestamp when the shipping case was created.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// The timestamp when the shipping case was last updated.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

// ShippingCaseLabelURL represents the response for a shipping case label URL.
type ShippingCaseLabelURL struct {
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=shipping_case_label_url"`
	// The presigned URL for the label, or null if no label exists.
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
