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

// RegistrationFlowOption represents a selectable option within a registration flow.
type RegistrationFlowOption struct {
	// The unique identifier for the registration flow option.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=registration_flow_option"`
	// The display name of the registration flow option.
	Name string `json:"name" validate:"required"`
}

// RegistrationFlow represents a configured registration flow for customer onboarding.
type RegistrationFlow struct {
	// The unique identifier for the registration flow.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=registration_flow"`
	// The display name of the registration flow.
	Name string `json:"name" validate:"required"`
	// The customer group options available in this registration flow.
	CustomerGroupOptions []RegistrationFlowOption `json:"customer_group_options" validate:"required"`
	// The payment term options available in this registration flow.
	PaymentTermOptions []RegistrationFlowOption `json:"payment_term_options" validate:"required"`
	// The shipping term options available in this registration flow.
	ShippingTermOptions []RegistrationFlowOption `json:"shipping_term_options" validate:"required"`
	// When this registration flow was created.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// When this registration flow was last updated.
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
	CustomerGroupOptions: []RegistrationFlowOption{SampleRegistrationFlowOption},
	PaymentTermOptions:   []RegistrationFlowOption{SampleRegistrationFlowOption},
	ShippingTermOptions:  []RegistrationFlowOption{SampleRegistrationFlowOption},
	CreatedAt:            timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:            timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*RegistrationFlowOption) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(&SampleRegistrationFlowOption)
}

func (*RegistrationFlow) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleRegistrationFlow)
}
