package constants

// SubassemblyFilter controls which items are returned when listing by subassembly scope.
type SubassemblyFilter string

const (
	// SubassemblyFilterAll does not restrict to initial subassemblies only.
	SubassemblyFilterAll SubassemblyFilter = "all"
	// SubassemblyFilterInitialOnly returns only items that are initial subassemblies.
	SubassemblyFilterInitialOnly SubassemblyFilter = "initial_only"
)

func (f SubassemblyFilter) IsValid() bool {
	switch f {
	case SubassemblyFilterAll, SubassemblyFilterInitialOnly:
		return true
	default:
		return false
	}
}

func (f SubassemblyFilter) EnumValues() []string {
	return []string{string(SubassemblyFilterAll), string(SubassemblyFilterInitialOnly)}
}
