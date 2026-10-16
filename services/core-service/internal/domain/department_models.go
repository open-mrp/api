package domain

import (
	"time"

	"github.com/open-mrp/apikit/field"
	"github.com/open-mrp/apikit/pagination"
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
	LaborRate *UnitPrice `audit:"labor_rate"`
	// LaborRateRowID is the legacy rate row LaborRate is also written to, for code that still reads it. CLEAN drops it.
	LaborRateRowID   *string
	ScanningStations []DepartmentScanningStation `audit:"scanning_stations"`
	Machines         []DepartmentMachine         `audit:"machines"`
	// ScanningStationsNextCursor and MachinesNextCursor continue the scanning station and machine lists after the
	// loaded members; nil when every member was loaded.
	ScanningStationsNextCursor *string
	MachinesNextCursor         *string
	AccountID                  string
	CreatedAt                  time.Time
	UpdatedAt                  time.Time
}

// EmbeddedDepartmentMembers is how many machines and scanning stations a loaded department carries: one more than an
// embedded list shows, so a longer list is noticed without counting it.
const EmbeddedDepartmentMembers = 11

// DepartmentReferences blocks deleting a department (a work center in the API) that machines, scanning stations,
// manufacturing method operations or account users still belong to.
var DepartmentReferences = RegisterReferences(References{
	Noun: "work center",
	Probes: []ReferenceProbe{
		{Label: "machines", Table: "machine", Column: "department_id"},
		{Label: "scanning stations", Table: "scanning_station", Column: "department_id"},
		{Label: "manufacturing method operations", Table: "production_step", Column: "department_id"},
		{Label: "account users", Table: "account_user", Column: "department_id"},
	},
})

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
	LaborRate  *UnitPrice
	// LaborRateID is the legacy rate row the service created from LaborRate; the repo only links it.
	LaborRateID        *string
	ScanningStationIDs []string
	MachineIDs         []string
}

type UpdateDepartmentParams struct {
	AccountID    string
	DepartmentID string
	Name         *string
	Notes        field.Clearable[string]
	LocationID   *string
	// LaborRate sets the department's labor rate; a department's labor rate can be changed but not removed.
	LaborRate *UnitPrice
	// LaborRateID is the legacy rate row the service created from LaborRate when the department had none; the repo only
	// links it.
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
