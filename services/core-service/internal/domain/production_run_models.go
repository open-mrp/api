package domain

import (
	"time"

	"github.com/augno/api/shared/pagination"
)

// ProductionRun represents a full production run domain model.
type ProductionRun struct {
	ID                string
	Number            string `audit:"number"`
	ResponsibleUserID string `audit:"responsible_user_id"`
	AccountID         string
	BatchCount        int32      `audit:"batch_count"`
	StartedAt         *time.Time `audit:"started_at"`
	CompletedAt       *time.Time `audit:"completed_at"`
	CreatedAt         time.Time
	UpdatedAt         time.Time

	// Joined fields for reads
	ResponsibleUserName *string
}

// ProductionRunSummary represents a production run for list views.
type ProductionRunSummary struct {
	ID                string
	Number            string
	ResponsibleUserID string
	BatchCount        int32
	StartedAt         *time.Time
	CompletedAt       *time.Time
	CreatedAt         time.Time
	UpdatedAt         time.Time

	// Joined fields
	ResponsibleUserName *string
}

// ListProductionRunsParams holds the parameters for listing production runs.
type ListProductionRunsParams struct {
	Cursor     *string
	Limit      int32
	Query      *string
	Status     *string
	ItemIDs    []string
	MachineIDs []string
	StartDate  *string
	EndDate    *string
	AccountID  string
}

// ListProductionRunsResult holds the result of listing production runs.
type ListProductionRunsResult struct {
	ProductionRuns []*ProductionRunSummary
	PageInfo       pagination.PageInfo
}

// GetProductionRunParams holds the parameters for getting a single production run.
type GetProductionRunParams struct {
	ProductionRunID string
	AccountID       string
}

// CreateProductionRunParams holds the parameters for creating a production run.
type CreateProductionRunParams struct {
	AccountID         string
	ResponsibleUserID string
}

// UpdateProductionRunParams holds the parameters for updating a production run.
type UpdateProductionRunParams struct {
	ProductionRunID   string
	AccountID         string
	Number            *string
	ResponsibleUserID *string
}

// DeleteProductionRunParams holds the parameters for deleting a production run.
type DeleteProductionRunParams struct {
	ProductionRunID string
	AccountID       string
}

// AddBatchesToProductionRunParams holds the parameters for adding batches to a production run.
type AddBatchesToProductionRunParams struct {
	ProductionRunID string
	AccountID       string
	Batches         []AddBatchInput
}

// AddBatchInput represents a single batch to add to a production run.
type AddBatchInput struct {
	ID                string
	ItemID            string
	Quantity          CreateQuantityParams
	Seconds           *CreateQuantityParams
	Waste             *CreateQuantityParams
	ProductionStepID  *string
	ScanningStationID *string
}

// ListBatchesByProductionRunParams holds the parameters for listing batches by production run.
type ListBatchesByProductionRunParams struct {
	ProductionRunID string
	AccountID       string
	Cursor          *string
	Limit           int32
	SearchQuery     *string
}

// ListBatchesByProductionRunResult holds paginated batches for a production run.
type ListBatchesByProductionRunResult struct {
	Batches  []*Batch
	PageInfo pagination.PageInfo
}
