package constants

// TransitEstimateSource names how a cached lane estimate was obtained. It decides what a refresh is allowed to overwrite: a harvested row is disposable and can be replaced whenever a newer quote arrives, where an operator's row is the only transit the system will ever have for a lane no carrier will rate.
type TransitEstimateSource string

const (
	// TransitEstimateSourceShippo is harvested from a rate quote the system already made, and is free to be refreshed.
	TransitEstimateSourceShippo TransitEstimateSource = "shippo"
	// TransitEstimateSourceManual was entered by an operator and must survive refreshes.
	TransitEstimateSourceManual TransitEstimateSource = "manual"
)

func (m TransitEstimateSource) IsValid() bool {
	switch m {
	case TransitEstimateSourceShippo, TransitEstimateSourceManual:
		return true
	default:
		return false
	}
}

func (m TransitEstimateSource) EnumValues() []string {
	return []string{
		string(TransitEstimateSourceShippo),
		string(TransitEstimateSourceManual),
	}
}

func (m *TransitEstimateSource) StringPtr() *string {
	if m == nil {
		return nil
	}
	s := string(*m)
	return &s
}
