package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleInventoryChangeLogID = "icl_01jm4r6700f8nwq3v5hx2d9ktp"

// InventoryChangeLog represents a single inventory change log entry.
type InventoryChangeLog struct {
	// The unique identifier for the inventory change log.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=inventory_change_log"`
	// The code indicating the type of inventory change action.
	ActionTypeCode constants.InventoryActionType `json:"action_type" validate:"required"`
	// The quantity associated with this change.
	Quantity *Quantity `json:"quantity" expandable:"true"`
	// The item affected by this change.
	Item *Item `json:"item" expandable:"true"`
	// The user responsible for this change.
	ResponsibleUser *User `json:"responsible_user" expandable:"true"`
	// The scanning station where this change occurred.
	ResponsibleScanningStation *ScanningStation `json:"responsible_scanning_station" expandable:"true"`
	// The timestamp when this change was recorded.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// The timestamp when this record was last updated.
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
// Export endpoints may use arrays; they typically return an Excel file instead.
type ExportInventoryChangeLogsResponse struct {
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=list"`
	// The list of exported inventory change logs.
	Items []*InventoryChangeLog `json:"items" validate:"required"`
	// The total number of items exported.
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
