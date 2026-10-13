package apiresource

import (
	"time"

	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/timeutil"
)

// MachineItemScanSuggestion is a scan-history candidate for curating a machine's runnable items.
type MachineItemScanSuggestion struct {
	// The suggested item's ID; used as the row identity for includes.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=machine_item_scan_suggestion"`
	// The item suggested from scan history. Expandable.
	Item *Item `json:"item" expandable:"true"`
	// When this item was last scanned on the machine.
	LastScannedAt time.Time `json:"last_scanned_at" validate:"required"`
	// How many scanned batches of this item were found on the machine in the suggestion window.
	BatchCount int64 `json:"batch_count" validate:"required"`
}

var sampleMachineItemScanSuggestion = &MachineItemScanSuggestion{
	ID:            SampleItemID,
	Object:        constants.ObjectTypeMachineItemScanSuggestion,
	LastScannedAt: timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	BatchCount:    12,
}

func (*MachineItemScanSuggestion) SchemaExample() any {
	return sampleMachineItemScanSuggestion
}
