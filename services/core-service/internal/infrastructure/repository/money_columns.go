package repository

import (
	gosql "database/sql"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/constants"
)

// Money lives on the row that owns it: {prefix}_amount, {prefix}_currency and, for a price, {prefix}_per_unit_id
// (migration 00088). A row written before those columns existed still points at a quantity or rate row in a currency
// unit, and reads fall back to it until MoneyColumnBackfill has filled the row. Every currency unit is a dollar, so a
// fallback amount is usd. The fallback goes once CLEAN drops the old ids.

// unitPriceFromColumns is a row's price from its money columns, or from its legacy rate row (the rate's value and
// denominator unit) when the columns are still empty. Nil when the row has neither.
func unitPriceFromColumns(amount, currency, perUnitID, legacyValue, legacyPerUnitID gosql.NullString) *domain.UnitPrice {
	if amount.Valid {
		return &domain.UnitPrice{Amount: amount.String, Currency: constants.Currency(currency.String), PerUnitID: perUnitID.String}
	}
	if legacyValue.Valid {
		return &domain.UnitPrice{Amount: legacyValue.String, Currency: constants.CurrencyUSD, PerUnitID: legacyPerUnitID.String}
	}
	return nil
}

// moneyAmountOrLegacy is a row's amount column, or its legacy row's value while the column is still empty.
func moneyAmountOrLegacy(amount, legacyValue gosql.NullString) gosql.NullString {
	if amount.Valid {
		return amount
	}
	return legacyValue
}

// unitPriceParams is a price as its three money columns store it; nil is all three NULL.
func unitPriceParams(p *domain.UnitPrice) (amount, currency, perUnitID gosql.NullString) {
	if p == nil {
		return
	}
	return gosql.NullString{String: p.Amount, Valid: true},
		gosql.NullString{String: string(p.Currency), Valid: true},
		gosql.NullString{String: p.PerUnitID, Valid: true}
}

// moneyParams is an amount as its two money columns store it; nil is both NULL.
func moneyParams(m *domain.Money) (amount, currency gosql.NullString) {
	if m == nil {
		return
	}
	return gosql.NullString{String: m.Amount, Valid: true}, gosql.NullString{String: string(m.Currency), Valid: true}
}
