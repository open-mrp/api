package apiresource

import (
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
)

// Freight describes the carrier selection and freight billing for a record.
//
// It is a generic, reusable sub-resource shared by anything that carries shipping configuration — a sales order, a purchase order, or a shipment.
type Freight struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=freight"`
	// How freight is arranged and billed for the record.
	//
	// - `free_freight`: no shipping cost to the buyer.
	// - `billed_freight`: freight is billed to the buyer.
	//
	// Sales orders, purchase orders, and shipments do not carry a policy of their own. Freight on those records is waived when the customer's freight preferences, the customer's type group, any of its pricing groups, the customer's shipping term, or any product line on the order is `free_freight`.
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

var sampleFreightPolicy = constants.FreightPolicyBilled
var sampleFreightBillingType = constants.CarrierBillingTypeThirdParty
var sampleFreightBillingAccountNumber = "123456789"

var SampleFreight = &Freight{
	Object:               constants.ObjectTypeFreight,
	Policy:               &sampleFreightPolicy,
	Carrier:              SampleCarrier,
	ServiceLevel:         SampleServiceLevel,
	BillingType:          &sampleFreightBillingType,
	BillingAccountNumber: &sampleFreightBillingAccountNumber,
}

func (*Freight) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleFreight)
}
