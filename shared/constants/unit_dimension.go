package constants

// UnitDimension is what a unit measures. Values convert only between units of the same dimension. Money is not a
// dimension: amounts carry a currency instead of a unit.
type UnitDimension string

const (
	// UnitDimensionQuantity is for countable things, such as each, pair or case.
	UnitDimensionQuantity UnitDimension = "quantity"
	// UnitDimensionTime measures durations, such as hours or minutes.
	UnitDimensionTime UnitDimension = "time"
	// UnitDimensionMass measures weight, such as kilograms or pounds.
	UnitDimensionMass UnitDimension = "mass"
	// UnitDimensionVolume measures volume, such as liters or gallons.
	UnitDimensionVolume UnitDimension = "volume"
	// UnitDimensionLength measures distance, such as meters or feet.
	UnitDimensionLength UnitDimension = "length"
	// UnitDimensionTemperature measures temperature, such as Celsius or Fahrenheit.
	UnitDimensionTemperature UnitDimension = "temperature"
	// UnitDimensionArea measures area, such as square meters or acres.
	UnitDimensionArea UnitDimension = "area"
)

func (d UnitDimension) IsValid() bool {
	switch d {
	case UnitDimensionQuantity, UnitDimensionTime, UnitDimensionMass, UnitDimensionVolume, UnitDimensionLength,
		UnitDimensionTemperature, UnitDimensionArea:
		return true
	default:
		return false
	}
}

func (d UnitDimension) EnumValues() []string {
	return []string{
		string(UnitDimensionQuantity),
		string(UnitDimensionTime),
		string(UnitDimensionMass),
		string(UnitDimensionVolume),
		string(UnitDimensionLength),
		string(UnitDimensionTemperature),
		string(UnitDimensionArea),
	}
}
