package constants

// LabelSizeCode identifies a label size for a scanning station.
type LabelSizeCode string

const (
	LabelSizeCodeOneByOne   LabelSizeCode = "1x1"
	LabelSizeCodeOneByThree LabelSizeCode = "1x3"
	LabelSizeCodeOneByFour  LabelSizeCode = "1x4"
	LabelSizeCodeTwoByFour  LabelSizeCode = "2x4"
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
