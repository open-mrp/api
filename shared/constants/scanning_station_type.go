package constants

// ScanningStationType represents the type of a scanning station.
type ScanningStationType string

const (
	// ScanningStationTypeInitBatch is the type for initializing a batch.
	ScanningStationTypeInitBatch ScanningStationType = "init_batch"
	// ScanningStationTypeMergeBatch is the type for merging batches.
	ScanningStationTypeMergeBatch ScanningStationType = "merge_batch"
	// ScanningStationTypeMoveBatch is the type for moving a batch.
	ScanningStationTypeMoveBatch ScanningStationType = "move_batch"
	// ScanningStationTypeSplitBatch is the type for splitting a batch.
	ScanningStationTypeSplitBatch ScanningStationType = "split_batch"
)

func (s ScanningStationType) IsValid() bool {
	switch s {
	case ScanningStationTypeInitBatch, ScanningStationTypeMergeBatch, ScanningStationTypeMoveBatch, ScanningStationTypeSplitBatch:
		return true
	default:
		return false
	}
}

func (s ScanningStationType) EnumValues() []string {
	return []string{
		string(ScanningStationTypeInitBatch),
		string(ScanningStationTypeMergeBatch),
		string(ScanningStationTypeMoveBatch),
		string(ScanningStationTypeSplitBatch),
	}
}
