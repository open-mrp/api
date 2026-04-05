package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

// ProductionRunDetail represents a full production run resource for single-object responses.
type ProductionRunDetail struct {
	// The unique identifier for the production run.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=production_run"`
	// The production run number, unique per account.
	Number string `json:"number" validate:"required"`
	// The user responsible for this production run.
	ResponsibleUser *AccountUser `json:"responsible_user" expandable:"true"`
	// The number of batches in this production run.
	BatchCount int32 `json:"batch_count" validate:"required"`
	// When the production run was started.
	StartedAt *time.Time `json:"started_at"`
	// When the production run was completed.
	CompletedAt *time.Time `json:"completed_at"`
	// When the production run was created.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// When the production run was last updated.
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

// ProductionRunSummary represents a production run for list views.
type ProductionRunSummary struct {
	// The unique identifier for the production run.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=production_run"`
	// The production run number, unique per account.
	Number string `json:"number" validate:"required"`
	// The user responsible for this production run.
	ResponsibleUser *AccountUser `json:"responsible_user"`
	// The number of batches in this production run.
	BatchCount int32 `json:"batch_count" validate:"required"`
	// When the production run was started.
	StartedAt *time.Time `json:"started_at"`
	// When the production run was completed.
	CompletedAt *time.Time `json:"completed_at"`
	// When the production run was created.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// When the production run was last updated.
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
