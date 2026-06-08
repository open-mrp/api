package apiresource

import (
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
)

// Freight describes the carrier selection and freight billing for a record. It
// is a generic, reusable sub-resource shared by anything that carries shipping
// configuration — e.g. a sales order's chosen freight, or a customer's default
// freight preferences. It is itself expanded via its parent (e.g.
// include[]=freight); when present, the full carrier and service level are
// included.
type Freight struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=freight"`
	// Freight policy (who arranges and pays for freight). Populated where a
	// policy applies, such as customer defaults.
	Policy *constants.FreightPolicy `json:"policy"`
	// Carrier.
	Carrier *Carrier `json:"carrier"`
	// Service level.
	ServiceLevel *ServiceLevel `json:"service_level"`
	// Who is billed for freight (sender or third_party).
	BillingType *constants.CarrierBillingType `json:"billing_type"`
	// Carrier billing account number, used when a third party is billed.
	BillingAccountNumber *string `json:"billing_account_number"`
}

var sampleFreightBillingType = constants.CarrierBillingTypeThirdParty
var sampleFreightBillingAccountNumber = "123456789"

var SampleFreight = &Freight{
	Object:               constants.ObjectTypeFreight,
	Carrier:              SampleCarrier,
	ServiceLevel:         SampleServiceLevel,
	BillingType:          &sampleFreightBillingType,
	BillingAccountNumber: &sampleFreightBillingAccountNumber,
}

func (*Freight) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleFreight)
}
