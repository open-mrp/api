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
	Number string `json:"number" validate:"required"`
	// Account user accountable for executing the run.
	ResponsibleUser *AccountUser `json:"responsible_user" expandable:"true"`
	// Number of batches currently recorded against this run.
	BatchCount int32 `json:"batch_count" validate:"required"`
	// Time the run started production, or `null` if it has not started yet.
	StartedAt *time.Time `json:"started_at"`
	// Time the run was marked complete, or `null` if still in progress.
	//
	// Once set, new batches can no longer be added to the run.
	CompletedAt *time.Time `json:"completed_at"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last-updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var SampleProductionRunDetail = &ProductionRunDetail{
	ID:         SampleProductionRunID,
	Object:     constants.ObjectTypeProductionRun,
	Number:     "1",
	BatchCount: 3,
	CreatedAt:  timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:  timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
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
	Number string `json:"number" validate:"required"`
	// Account user accountable for executing the run.
	ResponsibleUser *AccountUser `json:"responsible_user"`
	// Number of batches currently recorded against this run.
	BatchCount int32 `json:"batch_count" validate:"required"`
	// Time the run started production, or `null` if it has not started yet.
	StartedAt *time.Time `json:"started_at"`
	// Time the run was marked complete, or `null` if still in progress.
	//
	// Once set, new batches can no longer be added to the run.
	CompletedAt *time.Time `json:"completed_at"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last-updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var SampleProductionRunSummary = &ProductionRunSummary{
	ID:         SampleProductionRunID,
	Object:     constants.ObjectTypeProductionRun,
	Number:     "1",
	BatchCount: 3,
	CreatedAt:  timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:  timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*ProductionRunSummary) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleProductionRunSummary)
}
