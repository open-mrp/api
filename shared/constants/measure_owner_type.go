package constants

// MeasureOwnerType names the kind of resource a rate or quantity is attached to.
type MeasureOwnerType string

const (
	// MeasureOwnerTypeItem attaches the measure to an item.
	MeasureOwnerTypeItem MeasureOwnerType = "item"
	// MeasureOwnerTypeProductionStep attaches the measure to a production step.
	MeasureOwnerTypeProductionStep MeasureOwnerType = "production_step"
	// MeasureOwnerTypeDepartment attaches the measure to a department.
	MeasureOwnerTypeDepartment MeasureOwnerType = "department"
)

func (t MeasureOwnerType) IsValid() bool {
	switch t {
	case MeasureOwnerTypeItem, MeasureOwnerTypeProductionStep, MeasureOwnerTypeDepartment:
		return true
	default:
		return false
	}
}

func (t MeasureOwnerType) EnumValues() []string {
	return []string{
		string(MeasureOwnerTypeItem),
		string(MeasureOwnerTypeProductionStep),
		string(MeasureOwnerTypeDepartment),
	}
}
