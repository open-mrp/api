package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleRegistrationFlowID = "rgfw_01jm4r6700f8nwq3v5hx2d9ktp"
const SampleRegistrationFlowName = "Default Registration Flow"
const SampleRegistrationFlowOptionID = "rgfwo_01jm4r6700f8nwq3v5hx2d9ktp"
const SampleRegistrationFlowOptionName = "Standard Option"

// Selectable option within a registration flow.
type RegistrationFlowOption struct {
	// Registration flow option ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=registration_flow_option"`
	// Display name.
	Name string `json:"name" validate:"required"`
}

// Registration flow for customer onboarding.
type RegistrationFlow struct {
	// Registration flow ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=registration_flow"`
	// Display name.
	Name string `json:"name" validate:"required"`
	// Customer group options available in this flow.
	CustomerGroupOptions *List[RegistrationFlowOption] `json:"customer_group_options" validate:"required"`
	// Payment term options available in this flow.
	PaymentTermOptions *List[RegistrationFlowOption] `json:"payment_term_options" validate:"required"`
	// Shipping term options available in this flow.
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
