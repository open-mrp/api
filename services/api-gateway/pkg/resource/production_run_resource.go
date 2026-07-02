package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

// Production run resource for single-object responses.
type ProductionRunDetail struct {
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
	// Set automatically when the first batch in the run is scanned, and unset until then.
	StartedAt *time.Time `json:"started_at"`
	// Time the run was marked complete.
	//
	// Set automatically once every batch in the run has been scanned or deleted, and unset while the run is still in progress. Once set, the run can no longer be updated and new batches can no longer be added.
	CompletedAt *time.Time `json:"completed_at"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last-updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var SampleProductionRunDetail = &ProductionRunDetail{
	ID:              SampleProductionRunID,
	Object:          constants.ObjectTypeProductionRun,
	Number:          "1",
	ResponsibleUser: SampleAccountUser,
	BatchCount:      3,
	StartedAt:       timeutil.TimestampToTimePtr(sampleUpdatedAtTimestamp),
	CreatedAt:       timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:       timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*ProductionRunDetail) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleProductionRunDetail)
}

// Production run resource for list views.
type ProductionRunSummary struct {
	// Production run ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=production_run"`
	// Production run number, unique per account.
	//
	// Assigned automatically at creation as the next sequential number for the account; can be changed via update.
	Number string `json:"number" validate:"required"`
	// Account user accountable for executing the run.
	ResponsibleUser *AccountUser `json:"responsible_user"`
	// Number of batches currently recorded against this run.
	BatchCount int32 `json:"batch_count" validate:"required"`
	// Time the run started production.
	//
	// Set automatically when the first batch in the run is scanned, and unset until then.
	StartedAt *time.Time `json:"started_at"`
	// Time the run was marked complete.
	//
	// Set automatically once every batch in the run has been scanned or deleted, and unset while the run is still in progress. Once set, the run can no longer be updated and new batches can no longer be added.
	CompletedAt *time.Time `json:"completed_at"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last-updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var SampleProductionRunSummary = &ProductionRunSummary{
	ID:              SampleProductionRunID,
	Object:          constants.ObjectTypeProductionRun,
	Number:          "1",
	ResponsibleUser: SampleAccountUser,
	BatchCount:      3,
	StartedAt:       timeutil.TimestampToTimePtr(sampleUpdatedAtTimestamp),
	CreatedAt:       timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:       timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*ProductionRunSummary) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleProductionRunSummary)
}
