package repository

import (
	gosql "database/sql"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/constants"
)

func TestMoneyCopyTables_PairEveryCopyWithItsSource(t *testing.T) {
	for _, tb := range append(moneyCopyTables, orderLineMoneyCopies) {
		require.Len(t, tb.sources, len(tb.copies), tb.table)
		require.NotEmpty(t, tb.uncopied, tb.table)
		for _, u := range tb.uncopied {
			require.Contains(t, tb.copies, u, "%s: an uncopied marker is one of its copies", tb.table)
		}
	}
}

func TestMoneyColumnBackfills_CoverEveryTable(t *testing.T) {
	names := map[string]bool{}
	for _, b := range NewMoneyCopyBackfills(sqlc.New(nil), sqlc.New(nil)) {
		require.True(t, strings.HasPrefix(b.Name, "money_columns_"), b.Name)
		require.False(t, names[b.Name], "backfill names are unique")
		names[b.Name] = true
		require.Equal(t, "money_columns_"+b.t.table, b.Name)
	}
	require.Len(t, names, len(moneyCopyTables))
}

func TestMoneyDriftQuery_ComparesEachCopyNullSafely(t *testing.T) {
	q := moneyDriftQuery(creditLimitCopies)
	require.Contains(t, q, "FROM account_relation ar")
	require.Contains(t, q, "ar.credit_limit_amount <=> cl.value")
	require.Contains(t, q, "ar.credit_limit_currency <=> CASE WHEN cl.id IS NOT NULL THEN 'usd' END")
	require.Contains(t, q, "NOT (ar.credit_limit_currency IS NULL)")
}

func TestStoredMoney_ReadsTheCopiedCurrency(t *testing.T) {
	require.Equal(t, domain.Money{Amount: "5", Currency: constants.CurrencyUSD}, storedMoney("5", gosql.NullString{}),
		"a row the backfill has yet to reach is usd")
	require.Equal(t, constants.Currency("eur"), storedMoney("5", gosql.NullString{String: "eur", Valid: true}).Currency)
	require.Nil(t, storedMoneyPtr(gosql.NullString{}, gosql.NullString{}))
	require.Nil(t, storedRatePrice(nil, gosql.NullString{}))
	require.Equal(t, &domain.UnitPrice{Amount: "1.5", Currency: constants.CurrencyUSD, PerUnitID: "pair"},
		storedRatePrice(&domain.Rate{Value: "1.5", DenominatorUnitID: "pair"}, gosql.NullString{}))
}
