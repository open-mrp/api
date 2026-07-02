package apiresource

import (
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
)

// Freight describes the carrier selection and freight billing for a record.
//
// It is a generic, reusable sub-resource shared by anything that carries shipping configuration — for example a sales order's chosen freight, or a customer's default freight preferences.
type Freight struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=freight"`
	// How freight is arranged and billed for the record.
	//
	// Populated where a freight policy applies, such as a customer's default preferences.
	//
	// - `free_freight`: no shipping cost to the buyer.
	// - `billed_freight`: freight is billed to the buyer.
	Policy *constants.FreightPolicy `json:"policy"`
	// The shipping carrier selected to fulfill the shipment.
	Carrier *Carrier `json:"carrier"`
	// The carrier service level selected for the shipment (e.g. ground, overnight).
	ServiceLevel *ServiceLevel `json:"service_level"`
	// Which party the carrier bills for the shipment.
	//
	// - `sender`: the shipper (your account) is billed.
	// - `third_party`: a third party is billed via `billing_account_number`.
	BillingType *constants.CarrierBillingType `json:"billing_type"`
	// Carrier account number to bill, used when `billing_type` is `third_party`.
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
