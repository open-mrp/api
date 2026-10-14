package domain

import "github.com/open-mrp/api/shared/constants"

// The functions below read and write money that is still stored as quantity and rate rows in a currency unit. They
// are temporary: once each owning table has its own amount and currency columns, its callers read those columns
// instead and these go away. Every currency unit is a dollar, so every amount is usd.

// MoneyFromCurrencyQuantity reads an amount stored as a quantity in a currency unit.
func MoneyFromCurrencyQuantity(value string) Money {
	return Money{Amount: value, Currency: constants.CurrencyUSD}
}

// MoneyPtrFromCurrencyQuantity is MoneyFromCurrencyQuantity for an amount that may be absent.
func MoneyPtrFromCurrencyQuantity(value *string) *Money {
	if value == nil {
		return nil
	}
	m := MoneyFromCurrencyQuantity(*value)
	return &m
}

// UnitPriceFromCurrencyRate reads a price stored as a rate whose numerator is a currency unit.
func UnitPriceFromCurrencyRate(value, denominatorUnitID string) UnitPrice {
	return UnitPrice{Amount: value, Currency: constants.CurrencyUSD, PerUnitID: denominatorUnitID}
}

// UnitPricePtrFromCurrencyRate is UnitPriceFromCurrencyRate for a price that may be absent.
func UnitPricePtrFromCurrencyRate(value *string, denominatorUnitID string) *UnitPrice {
	if value == nil {
		return nil
	}
	p := UnitPriceFromCurrencyRate(*value, denominatorUnitID)
	return &p
}

// RateParams writes the price as the rate it is stored as: currencyUnitID (the currency's unit) per one of its unit.
func (p UnitPrice) RateParams(currencyUnitID string) CreateRateParams {
	return CreateRateParams{Value: p.Amount, NumeratorUnitID: currencyUnitID, DenominatorUnitID: p.PerUnitID}
}
