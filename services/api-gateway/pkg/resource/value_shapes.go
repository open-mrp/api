package apiresource

import (
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/decimalfmt"
	apiexample "github.com/open-mrp/apikit/example"
)

// An amount of money.
type Money struct {
	// Decimal amount, as a string to keep precision.
	Amount string `json:"amount" validate:"required" format:"decimal"`
	// Three-letter ISO 4217 currency code, lowercase. Only `usd` is supported.
	Currency constants.Currency `json:"currency" validate:"required"`
}

// A measured amount: a value and the unit it is measured in.
type Quantity struct {
	// Decimal value, as a string to keep precision.
	Value string `json:"value" validate:"required" format:"decimal"`
	// Unit the value is measured in.
	Unit *Unit `json:"unit" validate:"required"`
}

// A value measured per another unit, such as `100 kg / hr`.
type Rate struct {
	// Decimal value, as a string to keep precision.
	Value string `json:"value" validate:"required" format:"decimal"`
	// Unit the value is measured in, such as kilograms in `100 kg / hr`.
	Unit *Unit `json:"unit" validate:"required"`
	// Unit the rate is measured per, such as hours in `100 kg / hr`.
	PerUnit *Unit `json:"per_unit" validate:"required"`
}

// An amount of money per one unit, such as a price per pair or a labor rate per hour.
type UnitPrice struct {
	// Decimal amount, as a string to keep precision.
	Amount string `json:"amount" validate:"required" format:"decimal"`
	// Three-letter ISO 4217 currency code, lowercase. Only `usd` is supported.
	Currency constants.Currency `json:"currency" validate:"required"`
	// Unit the amount is charged per, such as pairs in `$18.45 / pr`.
	PerUnit *Unit `json:"per_unit" validate:"required"`
}

// Something taken off a price or a total.
type Discount struct {
	// How the discount is expressed.
	//
	// `percent_off`: `percent_off` percent off. `amount_off`: `amount_off` subtracted.
	Type constants.DiscountType `json:"type" validate:"required"`
	// Whole-number percent taken off, from `0` to `100` (`"5"` is 5%).
	//
	// `null` unless `type` is `percent_off`.
	PercentOff *string `json:"percent_off" format:"decimal"`
	// Fixed amount taken off.
	//
	// `null` unless `type` is `amount_off`.
	AmountOff *Money `json:"amount_off"`
}

// UnitStub stands in for a value shape's unit until the gateway loads it: only the ID is set. Every stub reachable from a response is replaced with the full unit before the response is written.
func UnitStub(id string) *Unit {
	if id == "" {
		return nil
	}
	return &Unit{ID: id}
}

// IsUnitStub reports whether u is a stub still waiting for its unit to load.
func IsUnitStub(u *Unit) bool {
	return u != nil && u.Object == "" && u.ID != ""
}

// NewMoney builds a USD amount, keeping the amount's digits as given.
func NewMoney(amount string) *Money {
	return &Money{Amount: amount, Currency: constants.CurrencyUSD}
}

// NewQuantity builds a quantity whose unit loads before the response is written.
func NewQuantity(value, unitID string) *Quantity {
	return &Quantity{Value: value, Unit: UnitStub(unitID)}
}

// NewRate builds a rate whose units load before the response is written.
func NewRate(value, unitID, perUnitID string) *Rate {
	return &Rate{Value: value, Unit: UnitStub(unitID), PerUnit: UnitStub(perUnitID)}
}

// NewUnitPrice builds a USD price per unit whose unit loads before the response is written.
func NewUnitPrice(amount, perUnitID string) *UnitPrice {
	return &UnitPrice{Amount: amount, Currency: constants.CurrencyUSD, PerUnit: UnitStub(perUnitID)}
}

// NormalizeQuantityValue returns a canonical decimal string for API quantity values, trimming database fixed-point padding. Currency units use two fractional digits; other dimensions trim trailing fractional zeros.
func NormalizeQuantityValue(value, unitType string) string {
	return decimalfmt.Normalize(value, unitType == string(constants.UnitTypeCurrency))
}

// NormalizeMonetaryQuantityValue returns a decimal string with exactly two fractional digits.
func NormalizeMonetaryQuantityValue(value string) string {
	return decimalfmt.Normalize(value, true)
}

// NormalizeRateValue returns a canonical decimal string for API rate values. It trims database fixed-point padding while preserving at least two fractional digits when the source value is fractional.
func NormalizeRateValue(value string) string {
	return decimalfmt.Rate(value)
}

var sampleHourUnit = newSampleUnit("Hour", "hr", constants.UnitDimensionTime)

var SampleMoney = &Money{Amount: "1234.56", Currency: constants.CurrencyUSD}

var SampleQuantity = &Quantity{Value: "1200", Unit: SampleUnit}

var SampleRate = &Rate{Value: "100", Unit: SampleUnit, PerUnit: sampleHourUnit}

var SampleUnitPrice = &UnitPrice{Amount: "25.50", Currency: constants.CurrencyUSD, PerUnit: SampleUnit}

var SampleDiscount = &Discount{Type: constants.DiscountTypePercentOff, PercentOff: new("5")}

func (*Money) SchemaExample() any { return apiexample.ValidateAndMarshalToMap(SampleMoney) }

func (*Quantity) SchemaExample() any { return apiexample.ValidateAndMarshalToMap(SampleQuantity) }

func (*Rate) SchemaExample() any { return apiexample.ValidateAndMarshalToMap(SampleRate) }

func (*UnitPrice) SchemaExample() any { return apiexample.ValidateAndMarshalToMap(SampleUnitPrice) }

func (*Discount) SchemaExample() any { return apiexample.ValidateAndMarshalToMap(SampleDiscount) }
