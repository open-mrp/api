package constants

// OperatorRequirement represents operator requirements for a scanning station.
type OperatorRequirement string

const (
	// OperatorRequirementNone means no special operator requirement.
	OperatorRequirementNone OperatorRequirement = "none"
	// OperatorRequirementMaterialCheck means material check is required.
	OperatorRequirementMaterialCheck OperatorRequirement = "material_check"
)

func (o OperatorRequirement) IsValid() bool {
	switch o {
	case OperatorRequirementNone, OperatorRequirementMaterialCheck:
		return true
	default:
		return false
	}
}

func (o OperatorRequirement) EnumValues() []string {
	return []string{
		string(OperatorRequirementNone),
		string(OperatorRequirementMaterialCheck),
	}
}
