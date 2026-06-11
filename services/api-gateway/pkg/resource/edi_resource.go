package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleDCLocationID = "dclo_0191ce9223b21dc31c9ee09b3e"

// Customer sub-resource on a DC location.
type DCLocationCustomer struct {
	// Customer ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=customer"`
	// Display name.
	Name string `json:"name" validate:"required"`
}

// DC location resource.
type DCLocation struct {
	// DC location ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=dc_location"`
	// Free-form description identifying this distribution-center location, such as a warehouse name and bay (for example, `Warehouse A - Bay 3`).
	Location string `json:"location" validate:"required"`
	// The customer this DC location belongs to.
	//
	// Every DC location is tied to a customer.
	Customer *DCLocationCustomer `json:"customer"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
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

const SampleEDIRunID = "edru_016aa43a99df34b744f6e2b878"

// EDI run resource.
type EDIRun struct {
	// EDI run ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=edi_run"`
	// Timestamp when the EDI run completed.
	CompletedAt time.Time `json:"completed_at" validate:"required"`
	// Whether the EDI run succeeded.
	HasSucceeded bool `json:"has_succeeded" validate:"required"`
	// Timestamp when the EDI run was created.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Timestamp when the EDI run was last updated.
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
