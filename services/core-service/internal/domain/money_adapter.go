package domain

import "github.com/open-mrp/api/shared/constants"

// The functions below read and write money that is still stored as quantity and rate rows in a currency unit. They
// are temporary: once each owning table has its own amount and currency columns, its callers read those columns
// instead and these go away. Every currency unit is a dollar, so every amount is usd.

// MoneyFromCurrencyQuantity reads an amount stored as a quantity in a currency unit.
func MoneyFromCurrencyQuantity(value string) Money {
	return Money{Amount: value, Currency: constants.CurrencyUSD}
}

// UnitPriceFromCurrencyRate reads a price stored as a rate whose numerator is a currency unit.
func UnitPriceFromCurrencyRate(value, denominatorUnitID string) UnitPrice {
	return UnitPrice{Amount: value, Currency: constants.CurrencyUSD, PerUnitID: denominatorUnitID}
}

// RateParams writes the price as the rate it is stored as: currencyUnitID (the currency's unit) per one of its unit.
func (p UnitPrice) RateParams(currencyUnitID string) CreateRateParams {
	return CreateRateParams{Value: p.Amount, NumeratorUnitID: currencyUnitID, DenominatorUnitID: p.PerUnitID}
}
