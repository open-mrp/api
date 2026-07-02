package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleShippingCaseID = "shcs_01207a101ea1475c687a39cf76"

// A physical case packed within a shipment.
//
// Each case carries its own SSCC, carrier tracking number, shipping label, and freight cost and weight.
type ShippingCase struct {
	// Shipping case ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=shipping_case"`
	// Human-readable case number.
	Number string `json:"number" validate:"required"`
	// Serial Shipping Container Code.
	//
	// A GS1 SSCC-18 identifier assigned automatically when the shipment ships.
	SSCC *string `json:"sscc"`
	// Carrier tracking number.
	TrackingNumber *string `json:"tracking_number"`
	// When the case shipped.
	ShippedAt *time.Time `json:"shipped_at"`
	// Freight cost charged for this case.
	FreightAmount *Quantity `json:"freight_amount" expandable:"true"`
	// Shipping weight of this case.
	FreightWeight *Quantity `json:"freight_weight" expandable:"true"`
	// The shipment this case belongs to.
	Shipment *Shipment `json:"shipment" expandable:"true"`
	// The carrier transporting this case.
	Carrier *Carrier `json:"carrier" expandable:"true"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

// Presigned link to a shipping case's label image.
type ShippingCaseLabelURL struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=shipping_case_label_url"`
	// Presigned link to the shipping case's label image.
	//
	// The URL expires one hour after it is issued.
	URL *string `json:"url"`
}

var SampleShippingCase = &ShippingCase{
	ID:             SampleShippingCaseID,
	Object:         constants.ObjectTypeShippingCase,
	Number:         "SC-0001",
	SSCC:           new("003456789000000018"),
	TrackingNumber: new("1Z999AA10123456784"),
	ShippedAt:      timeutil.TimestampToTimePtr(sampleUpdatedAtTimestamp),
	FreightAmount: &Quantity{
		ID:           SampleQuantityID,
		Object:       constants.ObjectTypeQuantity,
		Value:        "12.500000000000000000000000000000",
		DisplayValue: "$12.50",
		Unit:         newSampleUnit("US Dollar", "$", constants.UnitTypeCurrency),
	},
	FreightWeight: &Quantity{
		ID:           SampleQuantityID,
		Object:       constants.ObjectTypeQuantity,
		Value:        "5.000000000000000000000000000000",
		DisplayValue: "5 lb",
		Unit:         newSampleUnit("Pound", "lb", constants.UnitTypeMass),
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
