package apiresource

import (
	"time"

	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/timeutil"
)

const SampleRegistrationFlowID = "rgfw_5jo86wzvfpgn"
const SampleRegistrationFlowName = "Default Registration Flow"
const SampleRegistrationFlowOptionID = "rgfwo_y0bcxsctjq8x"
const SampleRegistrationFlowOptionName = "Standard Option"

// Selectable option within a registration flow.
type RegistrationFlowOption struct {
	// ID of the underlying customer group, payment term, or shipping term this option refers to.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=registration_flow_option"`
	// Display name of the underlying customer group, payment term, or shipping term.
	Name string `json:"name" validate:"required"`
}

// Configuration for customer self-registration.
//
// A registration flow defines which customer groups, payment terms, and shipping terms a customer can choose from when registering with your account.
type RegistrationFlow struct {
	// Registration flow ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=registration_flow"`
	// Display name of the registration flow.
	Name string `json:"name" validate:"required"`
	// Customer groups a registering customer can be placed into.
	CustomerGroupOptions *List[RegistrationFlowOption] `json:"customer_group_options" validate:"required"`
	// Payment terms a registering customer can choose from.
	PaymentTermOptions *List[RegistrationFlowOption] `json:"payment_term_options" validate:"required"`
	// Shipping terms a registering customer can choose from.
	ShippingTermOptions *List[RegistrationFlowOption] `json:"shipping_term_options" validate:"required"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var SampleRegistrationFlowOption = RegistrationFlowOption{
	ID:     SampleRegistrationFlowOptionID,
	Object: constants.ObjectTypeRegistrationFlowOption,
	Name:   SampleRegistrationFlowOptionName,
}

var SampleRegistrationFlow = &RegistrationFlow{
	ID:                   SampleRegistrationFlowID,
	Object:               constants.ObjectTypeRegistrationFlow,
	Name:                 SampleRegistrationFlowName,
	CustomerGroupOptions: NewList([]RegistrationFlowOption{SampleRegistrationFlowOption}, PageInfo{}),
	PaymentTermOptions:   NewList([]RegistrationFlowOption{SampleRegistrationFlowOption}, PageInfo{}),
	ShippingTermOptions:  NewList([]RegistrationFlowOption{SampleRegistrationFlowOption}, PageInfo{}),
	CreatedAt:            timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:            timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*RegistrationFlowOption) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(&SampleRegistrationFlowOption)
}

func (*RegistrationFlow) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleRegistrationFlow)
}
