package constants

type UnitType string

const (
	UnitTypeCurrency    UnitType = "currency"
	UnitTypeQuantity    UnitType = "quantity"
	UnitTypeTime        UnitType = "time"
	UnitTypeMass        UnitType = "mass"
	UnitTypeVolume      UnitType = "volume"
	UnitTypeLength      UnitType = "length"
	UnitTypeTemperature UnitType = "temperature"
	UnitTypeArea        UnitType = "area"
)

func (m UnitType) IsValid() bool {
	switch m {
	case UnitTypeCurrency, UnitTypeQuantity, UnitTypeTime, UnitTypeMass, UnitTypeVolume, UnitTypeLength, UnitTypeTemperature, UnitTypeArea:
		return true
	default:
		return false
	}
}

func (m UnitType) EnumValues() []string {
	return []string{
		string(UnitTypeCurrency),
		string(UnitTypeQuantity),
		string(UnitTypeTime),
		string(UnitTypeMass),
		string(UnitTypeVolume),
		string(UnitTypeLength),
		string(UnitTypeTemperature),
		string(UnitTypeArea),
	}
}
