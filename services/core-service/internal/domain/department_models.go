package domain

import (
	"time"

	"github.com/augno/api/shared/pagination"
)

// DepartmentScanningStation is a scanning station sub-resource attached to a department.
// carries an export's filters — the list params without pagination, plus the cap
// that keeps one request from building an unbounded workbook
type ExportDepartmentsParams struct {
	AccountID string
	Query     *string
	Limit     int32
}

type DepartmentScanningStation struct {
	ID                  string
	Name                string
	Type                string
	OperatorRequirement string
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

// DepartmentMachine is a machine sub-resource attached to a department.
type DepartmentMachine struct {
	ID           string
	Name         string
	SerialNumber string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type Department struct {
	ID               string
	Name             string  `audit:"name"`
	Notes            *string `audit:"notes"`
	LocationID       *string `audit:"location_id"`
	LocationName     *string `audit:"location_name"`
	LocationTypeCode *string `audit:"location_type_code"`
	// LaborRate is the hourly cost of work done in this department (e.g. a changeover tech), used by production scheduling to cost changeovers. Nil when the department has none.
	LaborRate        *ProductionStepRate         `audit:"labor_rate"`
	ScanningStations []DepartmentScanningStation `audit:"scanning_stations"`
	Machines         []DepartmentMachine         `audit:"machines"`
	AccountID        string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type ListDepartmentsParams struct {
	AccountID string
	Cursor    *string
	Limit     int32
	Query     *string
}

type ListDepartmentsResult struct {
	Departments []*Department
	PageInfo    pagination.PageInfo
}

type GetDepartmentParams struct {
	AccountID    string
	DepartmentID string
}

type CreateDepartmentParams struct {
	AccountID  string
	Name       string
	Notes      *string
	LocationID *string
	LaborRate  *CreateRateParams
	// LaborRateID is the rate row the service created from LaborRate; the repo only links it.
	LaborRateID        *string
	ScanningStationIDs []string
	MachineIDs         []string
}

type UpdateDepartmentParams struct {
	AccountID    string
	DepartmentID string
	Name         *string
	Notes        *string
	LocationID   *string
	// LaborRate creates the department's rate when it has none, or rewrites the existing rate row in place.
	LaborRate *CreateRateParams
	// LaborRateID is the rate row the service created from LaborRate; the repo only links it.
	LaborRateID        *string
	ScanningStationIDs []string
	MachineIDs         []string
}

type DeleteDepartmentParams struct {
	AccountID    string
	DepartmentID string
}

// UpsertDepartmentParams is a single department in a bulk upsert, matched by name
// (case-insensitive) within the account. The location is referenced by name and
// resolved server-side. Machine / scanning-station attachment is not part of bulk
// upsert — machines and stations reference their department at creation.
type UpsertDepartmentParams struct {
	Name     string
	Notes    *string
	Location *ObjectIdentifier
}

// BulkUpsertDepartmentsParams holds the parameters for bulk upserting departments.
type BulkUpsertDepartmentsParams struct {
	Departments []UpsertDepartmentParams
}

// ResolvedUpsertDepartmentRow is a department upsert row with its location reference resolved
// to an id. No JSON tags: the engine round-trips job_items against this type, an internal column.
type ResolvedUpsertDepartmentRow struct {
	Name       string
	Notes      *string
	LocationID *string
}
