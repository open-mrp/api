package domain

import (
	"time"

	"github.com/open-mrp/api/services/core-service/internal/scheduling"
	"github.com/open-mrp/api/shared/constants"
)

// Resource-setting scopes. A setting attaches to a machine, a department or a production step; the scope says which. These alias the shared enum so the vocabulary has a single source of truth.
const (
	ScheduleResourceScopeMachine        = string(constants.ScheduleResourceScopeMachine)
	ScheduleResourceScopeDepartment     = string(constants.ScheduleResourceScopeDepartment)
	ScheduleResourceScopeProductionStep = string(constants.ScheduleResourceScopeProductionStep)
)

// StepGraph is the production-step DAG plus the metadata the explosion needs.
type StepGraph struct {
	Edges []scheduling.StepEdge
	Steps map[string]scheduling.StepInfo
}

// ProductionScheduleDerivedLine is downstream department work implied by a constraint campaign.
type ProductionScheduleDerivedLine struct {
	ID                   string
	ProductionScheduleID string
	SourceLineID         string

	ProductionStepID string
	DepartmentID     *string
	ItemID           string

	WeekIndex     int32
	WeekStartDate time.Time

	Quantity      float64
	PlannedUnitID *string

	ExplosionDepth int32
	OffsetWeeks    int32
	StatusCode     string

	CreatedAt time.Time
	UpdatedAt time.Time
}

type ListDerivedLinesParams struct {
	AccountID     string
	ScheduleID    string
	DepartmentIDs []string
	WeekIndex     *int32
}

// GenerationCadence is one account's request for a schedule on a timer.
type GenerationCadence struct {
	AccountID       string
	Cron            string
	Timezone        string
	AutoPublish     bool
	LastGeneratedAt *time.Time
	CreatedAt       time.Time
}

// RunScheduledGenerationParams drives a queued solve.
type RunScheduledGenerationParams struct {
	AccountID    string
	ScheduleID   string
	PlanningAsOf time.Time
	AutoPublish  bool
}

type EnqueueGenerationParams struct {
	AccountID string
	// ScheduleID is the placeholder row the consumer solves into. It exists before the message is published, so a tick that enqueued and then died still leaves a record the reaper can fail rather than the generation vanishing without trace.
	ScheduleID   string
	PlanningAsOf time.Time
	AutoPublish  bool
}

// CreateGeneratingScheduleParams creates the placeholder row a queued solve fills in.
//
// The row exists before the message is published so a tick that enqueued and then died still leaves a visible record; the reaper can then fail it rather than the generation vanishing without trace.
type CreateGeneratingScheduleParams struct {
	ID               string
	AccountID        string
	Version          int32
	Name             *string
	PlanningAsOf     time.Time
	HorizonStartDate time.Time
	HorizonEndDate   time.Time
	HorizonWeeks     int32
	FrozenWeeks      int32
	DemandBasisCode  string
}
