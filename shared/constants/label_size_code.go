package constants

// LabelSizeCode identifies a label size for a scanning station.
type LabelSizeCode string

const (
	// LabelSizeCodeOneByOne indicates a 1x1 label.
	LabelSizeCodeOneByOne LabelSizeCode = "1x1"
	// LabelSizeCodeOneByThree indicates a 1x3 label.
	LabelSizeCodeOneByThree LabelSizeCode = "1x3"
	// LabelSizeCodeOneByFour indicates a 1x4 label.
	LabelSizeCodeOneByFour LabelSizeCode = "1x4"
	// LabelSizeCodeTwoByFour indicates a 2x4 label.
	LabelSizeCodeTwoByFour LabelSizeCode = "2x4"
)

func (c LabelSizeCode) IsValid() bool {
	switch c {
	case LabelSizeCodeOneByOne, LabelSizeCodeOneByThree, LabelSizeCodeOneByFour, LabelSizeCodeTwoByFour:
		return true
	default:
		return false
	}
}

func (c LabelSizeCode) EnumValues() []string {
	return []string{string(LabelSizeCodeOneByOne), string(LabelSizeCodeOneByThree), string(LabelSizeCodeOneByFour), string(LabelSizeCodeTwoByFour)}
}
