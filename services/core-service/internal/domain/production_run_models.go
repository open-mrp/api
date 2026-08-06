package domain

import (
	"time"

	"github.com/augno/api/shared/pagination"
)

// ProductionRun represents a full production run domain model.
// carries an export's filters — the list params without pagination, plus the cap
// that keeps one request from building an unbounded workbook
type ExportProductionRunsParams struct {
	AccountID string
	Query     *string
	Limit     int32
}

// carries one run and its batches as the export sheet lays them out. The read
// model has neither batches nor a sales order, so the export reads its own shape.
type ProductionRunExport struct {
	ID                  string
	Number              string
	ResponsibleUserName string
	StartedAt           *time.Time
	CompletedAt         *time.Time
	OrderID             *string
	Batches             []ProductionRunExportBatch
}

// carries one batch of an exported run, one sheet row each
type ProductionRunExportBatch struct {
	ID             string
	ItemSKU        string
	QuantityValue  string
	QuantityUnit   string
	DepartmentName *string
	MachineNames   []string
	ScannedAt      *time.Time
}

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
	ResponsibleUserName       *string
	ResponsibleUserStatusCode *string
	ResponsibleUserCreatedAt  *time.Time
	ResponsibleUserUpdatedAt  *time.Time
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
	ResponsibleUserName       *string
	ResponsibleUserStatusCode *string
	ResponsibleUserCreatedAt  *time.Time
	ResponsibleUserUpdatedAt  *time.Time
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

// BulkCreateBatchParams is a single batch in a bulk production run create, with the
// item referenced by SKU (resolved server-side) and everything else by ID — all
// validated server-side.
type BulkCreateBatchParams struct {
	Item             ItemIdentifier
	QuantityValue    string
	QuantityUnit     UnitIdentifier
	SecondsValue     *string
	SecondsUnit      *UnitIdentifier
	WasteValue       *string
	WasteUnit        *UnitIdentifier
	ProductionStepID *string
	ScanningStation  *ObjectIdentifier
}

// BulkCreateProductionRunParams is a single production run in a bulk create, owning
// the batches created with it. The run number is auto-assigned sequentially.
type BulkCreateProductionRunParams struct {
	ResponsibleUserID string
	Batches           []BulkCreateBatchParams
}

// BulkCreateProductionRunsParams holds the parameters for bulk creating production
// runs with their batches.
type BulkCreateProductionRunsParams struct {
	ProductionRuns []BulkCreateProductionRunParams
}

// The BulkCreateProductionRunEvent* types below are resolved runs stored on the bulk
// create job: every reference resolved to an ID, every quantity a validated decimal,
// and the run and batch IDs pre-generated so redeliveries converge on the same rows.
// The account comes from the identity restored with the message, not from the payload.
//
// They carry no JSON tags: the job_items payload is marshaled and unmarshaled by the
// engine against these same types, and it is an internal column no client reads.

// BulkCreateProductionRunEventRun is one resolved run stored on the bulk create job.
type BulkCreateProductionRunEventRun struct {
	ProductionRunID   string
	ResponsibleUserID string
	Batches           []BulkCreateProductionRunEventBatch
}

// BulkCreateProductionRunEventBatch is one batch in a bulk create event.
type BulkCreateProductionRunEventBatch struct {
	BatchID           string
	ItemID            string
	QuantityValue     string
	QuantityUnitID    string
	SecondsValue      *string
	SecondsUnitID     *string
	WasteValue        *string
	WasteUnitID       *string
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
