package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleInventoryChangeLogID = "icl_01jm4r6700f8nwq3v5hx2d9ktp"

// InventoryChangeLog is an inventory change log entry.
type InventoryChangeLog struct {
	// Inventory change log ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=inventory_change_log"`
	// Inventory action type code.
	ActionTypeCode constants.InventoryActionType `json:"action_type" validate:"required"`
	// Quantity for this change.
	Quantity *Quantity `json:"quantity" expandable:"true"`
	// Item affected by this change.
	Item *Item `json:"item" expandable:"true"`
	// User responsible for this change.
	ResponsibleUser *User `json:"responsible_user" expandable:"true"`
	// Scanning station where this change occurred.
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

// ExportInventoryChangeLogsResponse is the response for the export endpoint when returning JSON.
// Export endpoints typically return an Excel file; this struct is used for JSON output.
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

func (*ExportInventoryChangeLogsResponse) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleExportInventoryChangeLogsResponse)
}
