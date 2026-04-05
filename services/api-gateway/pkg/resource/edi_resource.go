package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleDCLocationID = "dclo_01gf7a8200er3ar3pkfrb6kk30"

// DCLocationCustomer is a lightweight customer sub-resource on a DC location.
type DCLocationCustomer struct {
	// The unique identifier for the customer.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=customer"`
	// The display name of the customer.
	Name string `json:"name" validate:"required"`
}

// DCLocation represents a DC location.
type DCLocation struct {
	// The unique identifier for the DC location.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=dc_location"`
	// The location description.
	Location string `json:"location" validate:"required"`
	// The customer associated with this DC location.
	Customer *DCLocationCustomer `json:"customer"`
	// The timestamp when the DC location was created.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// The timestamp when the DC location was last updated.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var SampleDCLocation = &DCLocation{
	ID:       SampleDCLocationID,
	Object:   constants.ObjectTypeDCLocation,
	Location: "Warehouse A - Bay 3",
	Customer: &DCLocationCustomer{
		ID:     SampleCustomerID,
		Object: constants.ObjectTypeCustomer,
		Name:   SampleCustomerName,
	},
	CreatedAt: timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt: timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*DCLocation) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleDCLocation)
}

// ---------------------------------------------------------------------------
// EDI Run
// ---------------------------------------------------------------------------

const SampleEDIRunID = "edru_01gf7a8200er3ar3pkfrb6kk30"

// EDIRun represents an EDI run.
type EDIRun struct {
	// The unique identifier for the EDI run.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=edi_run"`
	// The timestamp when the EDI run completed.
	CompletedAt time.Time `json:"completed_at" validate:"required"`
	// Whether the EDI run succeeded.
	HasSucceeded bool `json:"has_succeeded" validate:"required"`
	// The timestamp when the EDI run was created.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// The timestamp when the EDI run was last updated.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var SampleEDIRun = &EDIRun{
	ID:           SampleEDIRunID,
	Object:       constants.ObjectTypeEDIRun,
	CompletedAt:  timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	HasSucceeded: true,
	CreatedAt:    timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:    timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*EDIRun) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleEDIRun)
}
