package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleShippingCaseID = "shcs_fgqy1eu256af"

// A physical case packed within a shipment.
//
// Cases are created when a pick is packed, one for each case counted on the pack, and each carries its own SSCC, carrier tracking number, shipping label, freight cost and shipping weight.
type ShippingCase struct {
	// Shipping case ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=shipping_case"`
	// Human-readable case number.
	//
	// Built from the shipment's number and the case's position within that shipment when the case is created.
	Number string `json:"number" validate:"required"`
	// Serial Shipping Container Code (SSCC) identifying this case.
	//
	// An 18-digit code assigned automatically when the shipment ships, if the case does not already have one. It is kept when the shipment is voided, so a case that ships again keeps the same code.
	SSCC *string `json:"sscc"`
	// Carrier tracking number.
	//
	// Recorded when a label is purchased for the case, can be overwritten manually, and is cleared if the shipment is voided.
	TrackingNumber *string `json:"tracking_number"`
	// When the case shipped.
	//
	// Stamped on every case in the shipment when the shipment ships, and cleared if the shipment is voided.
	ShippedAt *time.Time `json:"shipped_at"`
	// Freight cost charged for this case.
	//
	// Starts at zero when the case is created, and is reset to zero if the shipment is voided.
	FreightAmount *Quantity `json:"freight_amount" expandable:"true"`
	// Shipping weight of this case.
	FreightWeight *Quantity `json:"freight_weight" expandable:"true"`
	// The shipment this case belongs to.
	Shipment *Shipment `json:"shipment" expandable:"true"`
	// The carrier transporting this case.
	//
	// Copied from the sales order's carrier when the case is created.
	Carrier *Carrier `json:"carrier" expandable:"true"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

// A temporary download link for a shipping case's label image.
type ShippingCaseLabelURL struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=shipping_case_label_url"`
	// Presigned link to the shipping case's label image.
	//
	// The link expires one hour after it is issued, and is absent until a label has been generated for the case.
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
