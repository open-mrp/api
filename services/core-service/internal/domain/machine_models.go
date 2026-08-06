package domain

import (
	"time"

	"github.com/augno/api/shared/pagination"
)

type Machine struct {
	ID                  string
	Name                string  `audit:"name"`
	SerialNumber        string  `audit:"serial_number"`
	Notes               *string `audit:"notes"`
	DepartmentID        *string
	DepartmentName      *string `audit:"department_name"`
	DepartmentCreatedAt *time.Time
	DepartmentUpdatedAt *time.Time
	ProductionStepID    *string `audit:"production_step_id"`
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

type ListMachinesParams struct {
	AccountID string
	Cursor    *string
	Limit     int32
	Query     *string
}

type ListMachinesResult struct {
	Machines []*Machine
	PageInfo pagination.PageInfo
}

// carries an export's filters — the list params without pagination, plus the cap
// that keeps one request from building an unbounded workbook
type ExportMachinesParams struct {
	AccountID string
	Query     *string
	Limit     int32
}

type GetMachineParams struct {
	AccountID string
	MachineID string
}

type CreateMachineParams struct {
	AccountID    string
	Name         string
	SerialNumber string
	Notes        *string
	DepartmentID string
}

type UpdateMachineParams struct {
	AccountID    string
	MachineID    string
	Name         *string
	SerialNumber *string
	Notes        *string
}

type DeleteMachineParams struct {
	AccountID string
	MachineID string
}

// UpsertMachineParams is a single machine in a bulk upsert, matched by name OR serial
// number (case-insensitive) within the account. The department is referenced by name,
// resolved server-side, and confirms matching intent: updates must state the machine's
// current department, and a key matching a machine in a different department is
// rejected as a collision.
type UpsertMachineParams struct {
	Name         string
	SerialNumber string
	Notes        *string
	Department   ObjectIdentifier
}

// BulkUpsertMachinesParams holds the parameters for bulk upserting machines.
type BulkUpsertMachinesParams struct {
	Machines []UpsertMachineParams
}

// ResolvedUpsertMachineRow is a machine upsert row with its department reference resolved to
// an id. No JSON tags: the engine round-trips job_items against this type, an internal column.
type ResolvedUpsertMachineRow struct {
	Name         string
	SerialNumber string
	Notes        *string
	DepartmentID string
}
