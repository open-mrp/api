package constants

// LabelTypeCode identifies a label type for a scanning station.
type LabelTypeCode string

const (
	LabelTypeCodeTag      LabelTypeCode = "tag"
	LabelTypeCodeTraveler LabelTypeCode = "traveler"
)

func (c LabelTypeCode) IsValid() bool {
	switch c {
	case LabelTypeCodeTag, LabelTypeCodeTraveler:
		return true
	default:
		return false
	}
}

func (c LabelTypeCode) EnumValues() []string {
	return []string{string(LabelTypeCodeTag), string(LabelTypeCodeTraveler)}
}
