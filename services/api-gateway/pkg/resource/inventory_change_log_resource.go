package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleInventoryChangeLogID = "icl_kb4dlhqx4voe"

// A record of a single change to an item's on-hand inventory.
//
// Every inventory movement — production scans, manual user adjustments, and automatic system actions — produces one entry, forming an audit trail of how on-hand quantities changed over time.
type InventoryChangeLog struct {
	// Inventory change log ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=inventory_change_log"`
	// Action that produced this inventory change.
	//
	// - `scan`: change driven by a scan, typically a production step.
	// - `user_action`: change made manually by a user.
	// - `system_action`: change made automatically by the system.
	// - `user_correction`: manual adjustment a user made to correct an inventory discrepancy.
	ActionTypeCode constants.InventoryActionType `json:"action_type" validate:"required"`
	// Amount of inventory this change applied.
	//
	// The value is signed: positive values increased on-hand inventory, negative values decreased it.
	Quantity *Quantity `json:"quantity" expandable:"true"`
	// Item affected by this change.
	Item *Item `json:"item" expandable:"true"`
	// The user who made this change.
	ResponsibleUser *User `json:"responsible_user" expandable:"true"`
	// The scanning station this change came from.
	//
	// Present only for changes recorded on the production floor, which have an action type of `scan`.
	ResponsibleScanningStation *ScanningStation `json:"responsible_scanning_station" expandable:"true"`
	// Timestamp when this change was recorded.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Timestamp when this record was last updated.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var SampleInventoryChangeLog = &InventoryChangeLog{
	ID:                         SampleInventoryChangeLogID,
	Object:                     constants.ObjectTypeInventoryChangeLog,
	ActionTypeCode:             constants.InventoryActionTypeScan,
	Quantity:                   SampleQuantity,
	Item:                       SampleItem,
	ResponsibleUser:            SampleUser,
	ResponsibleScanningStation: SampleScanningStation,
	CreatedAt:                  timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:                  timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*InventoryChangeLog) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleInventoryChangeLog)
}

// The JSON shape of an inventory change log export.
//
// The export endpoint itself returns an Excel file; this structure documents the equivalent JSON payload.
type ExportInventoryChangeLogsResponse struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=list"`
	// Exported inventory change logs.
	Items []*InventoryChangeLog `json:"items" validate:"required"`
	// Total count of exported items.
	Count int64 `json:"count" validate:"required"`
}

var SampleExportInventoryChangeLogsResponse = &ExportInventoryChangeLogsResponse{
	Object: constants.ObjectTypeList,
	Items:  []*InventoryChangeLog{SampleInventoryChangeLog},
	Count:  1,
}
