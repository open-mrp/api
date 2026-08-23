package constants

// BatchLotType names what a batch's lot number traces back to.
type BatchLotType string

const (
	// BatchLotTypeMaterial means the lot number traces a raw material consumed by the batch.
	BatchLotTypeMaterial BatchLotType = "material"
	// BatchLotTypeProductionRun means the lot number is the production run number the batch belongs to.
	BatchLotTypeProductionRun BatchLotType = "productionRun"
)

func (t BatchLotType) IsValid() bool {
	switch t {
	case BatchLotTypeMaterial, BatchLotTypeProductionRun:
		return true
	default:
		return false
	}
}

func (t BatchLotType) EnumValues() []string {
	return []string{
		string(BatchLotTypeMaterial),
		string(BatchLotTypeProductionRun),
	}
}
