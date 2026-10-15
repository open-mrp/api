package repository

import (
	gosql "database/sql"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/constants"
)

func TestUnitPriceFromColumns(t *testing.T) {
	str := func(s string) gosql.NullString { return gosql.NullString{String: s, Valid: true} }
	var null gosql.NullString

	assert.Equal(t, &domain.UnitPrice{Amount: "25.5", Currency: constants.CurrencyUSD, PerUnitID: "un_hr"},
		unitPriceFromColumns(str("25.5"), str("usd"), str("un_hr"), str("99"), str("un_min")), "the columns win over the legacy row")
	assert.Equal(t, &domain.UnitPrice{Amount: "99", Currency: constants.CurrencyUSD, PerUnitID: "un_min"},
		unitPriceFromColumns(null, null, null, str("99"), str("un_min")), "an unfilled row reads its legacy rate as usd")
	assert.Nil(t, unitPriceFromColumns(null, null, null, null, null))
}

func TestMoneyAmountOrLegacy(t *testing.T) {
	str := func(s string) gosql.NullString { return gosql.NullString{String: s, Valid: true} }
	assert.Equal(t, str("1"), moneyAmountOrLegacy(str("1"), str("2")))
	assert.Equal(t, str("2"), moneyAmountOrLegacy(gosql.NullString{}, str("2")))
}

func TestUnitPriceParams(t *testing.T) {
	amount, currency, perUnit := unitPriceParams(&domain.UnitPrice{Amount: "3", Currency: constants.CurrencyUSD, PerUnitID: "un_ea"})
	assert.Equal(t, gosql.NullString{String: "3", Valid: true}, amount)
	assert.Equal(t, gosql.NullString{String: "usd", Valid: true}, currency)
	assert.Equal(t, gosql.NullString{String: "un_ea", Valid: true}, perUnit)

	amount, currency, perUnit = unitPriceParams(nil)
	assert.False(t, amount.Valid || currency.Valid || perUnit.Valid)

	m, c := moneyParams(&domain.Money{Amount: "4", Currency: constants.CurrencyUSD})
	assert.Equal(t, "4", m.String)
	assert.Equal(t, "usd", c.String)
}
