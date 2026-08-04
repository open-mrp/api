package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

// A production run: the group of shop-floor batches that are executed together, tracked from the first batch scan through to completion.
type ProductionRun struct {
	// Production run ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=production_run"`
	// Production run number, unique per account.
	//
	// Assigned automatically at creation as the next sequential number for the account; can be changed via update.
	Number string `json:"number" validate:"required"`
	// Account user accountable for executing the run.
	ResponsibleUser *AccountUser `json:"responsible_user" expandable:"true"`
	// Number of batches currently recorded against this run.
	BatchCount int32 `json:"batch_count" validate:"required"`
	// Time the run started production.
	//
	// Set automatically the first time a batch in the run is scanned at a station.
	StartedAt *time.Time `json:"started_at"`
	// Time the run finished production.
	//
	// Set automatically once every batch in the run has been scanned or deleted. From that point the run can no longer be updated and no further batches can be added to it.
	CompletedAt *time.Time `json:"completed_at"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last-updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var SampleProductionRun = &ProductionRun{
	ID:              SampleProductionRunID,
	Object:          constants.ObjectTypeProductionRun,
	Number:          "1",
	ResponsibleUser: SampleAccountUser,
	BatchCount:      3,
	StartedAt:       timeutil.TimestampToTimePtr(sampleUpdatedAtTimestamp),
	CreatedAt:       timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:       timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*ProductionRun) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleProductionRun)
}
