package domain

import (
	"time"

	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/apikit/field"
	"github.com/open-mrp/apikit/pagination"
)

// ManufacturingMethod is how an item is made: its operations and the materials they consume. Methods are versioned;
// only drafts are edited, and activating one archives the item's other active standard methods.
type ManufacturingMethod struct {
	ID        string
	AccountID string
	ItemID    string                              `audit:"item_id"`
	Version   int32                               `audit:"version"`
	Status    constants.ManufacturingMethodStatus `audit:"status"`
	// OutputQuantity is how much of the item one run of the method makes; material quantities are per this much.
	OutputQuantity   QuantityValue `audit:"output_quantity"`
	SalesOrderLineID *string       `audit:"sales_order_line_id"`
	ActivatedAt      *time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time

	// The first operations and materials, in order, for the embedded lists: one more than a list shows, so a longer
	// list is noticed. The cursors continue each list after the last item it shows.
	OperationIDs         []string `audit:"-"`
	OperationsNextCursor *string  `audit:"-"`
	MaterialIDs          []string `audit:"-"`
	MaterialsNextCursor  *string  `audit:"-"`
}

// EmbeddedMethodMembers is how many operations and materials a loaded method carries: one more than an embedded list
// shows.
const EmbeddedMethodMembers = 11

// IsEditable reports whether the method's operations, materials and output can change.
func (m *ManufacturingMethod) IsEditable() bool {
	return m.Status == constants.ManufacturingMethodStatusDraft
}

// MethodOperation is one step of work in a method. It is stored as a production step, so its ID is the step's.
type MethodOperation struct {
	ID                  string
	MethodID            string                      `audit:"manufacturing_method_id"`
	Position            int32                       `audit:"position"`
	Sequence            constants.OperationSequence `audit:"sequence"`
	Type                constants.OperationType     `audit:"type"`
	WorkCenterID        *string                     `audit:"work_center_id"`
	MachineIDs          []string                    `audit:"machine_ids"`
	ScanningStationID   *string                     `audit:"scanning_station_id"`
	SetupTime           *QuantityValue              `audit:"setup_time"`
	LaborTime           *RateValue                  `audit:"labor_time"`
	MachineTime         *RateValue                  `audit:"machine_time"`
	LaborRate           *UnitPrice                  `audit:"labor_rate"`
	OverheadRate        *UnitPrice                  `audit:"overhead_rate"`
	LevelingPercent     string                      `audit:"leveling_percent"`
	AllowancePercent    string                      `audit:"allowance_percent"`
	WorkInstructions    *string                     `audit:"work_instructions"`
	LaborRateID         string                      `audit:"-"`
	LaborTimeID         string                      `audit:"-"`
	OverheadRateID      string                      `audit:"-"`
	SetupTimeQuantityID *string                     `audit:"-"`
	MachineTimeID       *string                     `audit:"-"`
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

// MethodMaterial is an item an operation consumes, per the method's output quantity. It is stored as a consumption.
type MethodMaterial struct {
	ID               string
	MethodID         string           `audit:"manufacturing_method_id"`
	ItemID           string           `audit:"item_id"`
	OperationID      *string          `audit:"operation_id"`
	Quantity         QuantityValue    `audit:"quantity"`
	ScrapQuantity    QuantityValue    `audit:"scrap_quantity"`
	Supply           constants.Supply `audit:"supply"`
	WorkInstructions *string          `audit:"work_instructions"`
	QuantityID       string           `audit:"-"`
	ScrapQuantityID  string           `audit:"-"`
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type ListManufacturingMethodsParams struct {
	AccountID     string
	Cursor        *string
	Limit         int32
	ItemIDs       []string
	Statuses      []constants.ManufacturingMethodStatus
	WorkCenterIDs []string
	Created       CreatedRange
}

type ListManufacturingMethodsResult struct {
	Methods  []*ManufacturingMethod
	PageInfo pagination.PageInfo
}

type CreateManufacturingMethodParams struct {
	AccountID string
	ItemID    string
	// OutputQuantity defaults to one of the item's stocking unit.
	OutputQuantity *QuantityValue
}

type UpdateManufacturingMethodParams struct {
	AccountID      string
	MethodID       string
	OutputQuantity *QuantityValue
}

// ListMethodMembersParams pages a method's operations or materials.
type ListMethodMembersParams struct {
	AccountID string
	MethodID  string
	Cursor    *string
	Limit     int32
}

type ListMethodOperationsResult struct {
	Operations []*MethodOperation
	PageInfo   pagination.PageInfo
}

type ListMethodMaterialsResult struct {
	Materials []*MethodMaterial
	PageInfo  pagination.PageInfo
}

// MethodOperationInput is an operation write. On create, unset fields take their defaults; on update they keep their
// values.
type MethodOperationInput struct {
	Position          *int32
	Sequence          *constants.OperationSequence
	WorkCenterID      *string
	MachineIDs        *[]string
	ScanningStationID field.Clearable[string]
	SetupTime         field.Clearable[QuantityValue]
	LaborTime         *RateValue
	MachineTime       field.Clearable[RateValue]
	LaborRate         *UnitPrice
	OverheadRate      *UnitPrice
	LevelingPercent   *string
	AllowancePercent  *string
	WorkInstructions  field.Clearable[string]
}

type CreateMethodOperationParams struct {
	AccountID string
	MethodID  string
	MethodOperationInput
}

type UpdateMethodOperationParams struct {
	AccountID   string
	MethodID    string
	OperationID string
	MethodOperationInput
}

// MethodMaterialInput is a material write. On create, unset fields take their defaults; on update they keep their
// values.
type MethodMaterialInput struct {
	ItemID           *string
	OperationID      *string
	Quantity         *QuantityValue
	ScrapQuantity    *QuantityValue
	Supply           *constants.Supply
	WorkInstructions field.Clearable[string]
}

type CreateMethodMaterialParams struct {
	AccountID string
	MethodID  string
	MethodMaterialInput
}

type UpdateMethodMaterialParams struct {
	AccountID  string
	MethodID   string
	MaterialID string
	MethodMaterialInput
}

// ProductionFlow is every active method behind an item: the item's own default method first, then the methods of the
// materials it makes, level by level down to purchased and stocked materials.
type ProductionFlow struct {
	ItemID    string
	MethodIDs []string
}

// BulkUpsertManufacturingMethodRow is one method in a bulk upsert: a new version for the item, activated unless the row
// asks for a draft. Items and units are resolved before the rows are stored on the job.
type BulkUpsertManufacturingMethodRow struct {
	Item           ItemIdentifier
	OutputQuantity *UpsertQuantity
	Status         *constants.ManufacturingMethodStatus
	Operations     []BulkUpsertMethodOperationRow
	Materials      []BulkUpsertMethodMaterialRow
}

// UpsertQuantity is a quantity whose unit is referenced fuzzily.
type UpsertQuantity struct {
	Value string
	Unit  UnitIdentifier
}

// UpsertRate is a rate whose units are referenced fuzzily.
type UpsertRate struct {
	Value   string
	Unit    UnitIdentifier
	PerUnit UnitIdentifier
}

// UpsertUnitPrice is a price whose per-unit is referenced fuzzily.
type UpsertUnitPrice struct {
	Amount   string
	Currency string
	PerUnit  UnitIdentifier
}

type BulkUpsertMethodOperationRow struct {
	Sequence         *constants.OperationSequence
	WorkCenter       *ObjectIdentifier
	Machines         []ObjectIdentifier
	ScanningStation  *ObjectIdentifier
	SetupTime        *UpsertQuantity
	LaborTime        UpsertRate
	MachineTime      *UpsertRate
	LaborRate        *UpsertUnitPrice
	OverheadRate     *UpsertUnitPrice
	LevelingPercent  *string
	AllowancePercent *string
	WorkInstructions *string
}

type BulkUpsertMethodMaterialRow struct {
	Item ItemIdentifier
	// Operation is the 1-based position of the consuming operation in the row's operations; the first when unset.
	Operation        *int32
	Quantity         UpsertQuantity
	ScrapQuantity    *UpsertQuantity
	Supply           *constants.Supply
	WorkInstructions *string
}

// ResolvedMethodRow is a bulk upsert row with every reference resolved to an ID, as stored on the job.
type ResolvedMethodRow struct {
	ItemID         string
	OutputQuantity *QuantityValue
	Status         constants.ManufacturingMethodStatus
	Operations     []MethodOperationInput
	Materials      []ResolvedMethodMaterial
}

// ResolvedMethodMaterial is a bulk material with its consuming operation as an index into the row's operations.
type ResolvedMethodMaterial struct {
	OperationIndex int
	MethodMaterialInput
}
