package constants

// PickStatus filters a list of picks by whether the pick has been finished.
type PickStatus string

const (
	// PickStatusOpen returns picks that have not been finished.
	PickStatusOpen PickStatus = "open"
	// PickStatusClosed returns picks that have been finished.
	PickStatusClosed PickStatus = "closed"
)

func (s PickStatus) IsValid() bool {
	switch s {
	case PickStatusOpen, PickStatusClosed:
		return true
	default:
		return false
	}
}

func (s PickStatus) EnumValues() []string {
	return []string{
		string(PickStatusOpen),
		string(PickStatusClosed),
	}
}
