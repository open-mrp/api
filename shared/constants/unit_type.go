package constants

// UnitType represents the category of a unit of measure.
type UnitType string

const (
	// UnitTypeCurrency represents monetary units such as dollars or euros.
	UnitTypeCurrency UnitType = "currency"
	// UnitTypeQuantity represents discrete countable units.
	UnitTypeQuantity UnitType = "quantity"
	// UnitTypeTime represents time-based units such as hours or minutes.
	UnitTypeTime UnitType = "time"
	// UnitTypeMass represents weight-based units such as kilograms or pounds.
	UnitTypeMass UnitType = "mass"
	// UnitTypeVolume represents volumetric units such as liters or gallons.
	UnitTypeVolume UnitType = "volume"
	// UnitTypeLength represents distance-based units such as meters or feet.
	UnitTypeLength UnitType = "length"
	// UnitTypeTemperature represents temperature units such as Celsius or Fahrenheit.
	UnitTypeTemperature UnitType = "temperature"
	// UnitTypeArea represents area-based units such as square meters or acres.
	UnitTypeArea UnitType = "area"
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
