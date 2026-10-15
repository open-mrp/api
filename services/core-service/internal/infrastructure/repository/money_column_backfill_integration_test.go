//go:build integration

package repository

import (
	"context"
	gosql "database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/apikit/db/backfill"
)

// TestMoneyColumnBackfill_LevelsColumnsWithLegacyRows runs the backfill over a scratch table with a price and an
// amount, in small batches.
func TestMoneyColumnBackfill_LevelsColumnsWithLegacyRows(t *testing.T) {
	pool := testDB(t)
	suffix := time.Now().UnixNano()
	table := fmt.Sprintf("money_backfill_test_%d", suffix)
	lcExec(t, pool, "CREATE TABLE `"+table+"` (id varchar(191) NOT NULL PRIMARY KEY, "+
		"fee_id varchar(191) NULL, fee_amount decimal(65,30) NULL, fee_currency char(3) NULL, fee_per_unit_id varchar(191) NULL, "+
		"total_id varchar(191) NULL, total_amount decimal(65,30) NULL, total_currency char(3) NULL) "+
		"DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci")
	t.Cleanup(func() { _, _ = pool.Exec("DROP TABLE `" + table + "`") })

	rate := func(id, value, per string) string {
		id = fmt.Sprintf("%s_%d", id, suffix)
		lcExec(t, pool, "INSERT INTO rate (id, value, numerator_unit_id, denominator_unit_id) VALUES (?, ?, 'un_dollar', ?)", id, value, per)
		t.Cleanup(func() { _, _ = pool.Exec("DELETE FROM rate WHERE id = ?", id) })
		return id
	}
	quantity := func(id, value string) string {
		id = fmt.Sprintf("%s_%d", id, suffix)
		lcExec(t, pool, "INSERT INTO quantity (id, value, unit_id) VALUES (?, ?, 'un_dollar')", id, value)
		t.Cleanup(func() { _, _ = pool.Exec("DELETE FROM quantity WHERE id = ?", id) })
		return id
	}

	// row_1 has only legacy rows; row_2 is already level; row_3's price went stale; row_4 has no legacy price.
	lcExec(t, pool, "INSERT INTO `"+table+"` (id, fee_id, total_id) VALUES ('row_1', ?, ?)", rate("rt_1", "25", "un_hr"), quantity("qt_1", "100.5"))
	lcExec(t, pool, "INSERT INTO `"+table+"` (id, fee_id, fee_amount, fee_currency, fee_per_unit_id, total_id, total_amount, total_currency) "+
		"VALUES ('row_2', ?, 12, 'usd', 'un_ea', ?, 7, 'usd')", rate("rt_2", "12", "un_ea"), quantity("qt_2", "7"))
	lcExec(t, pool, "INSERT INTO `"+table+"` (id, fee_id, fee_amount, fee_currency, fee_per_unit_id) VALUES ('row_3', ?, 20, 'usd', 'un_hr')",
		rate("rt_3", "30", "un_min"))
	lcExec(t, pool, "INSERT INTO `"+table+"` (id, total_id) VALUES ('row_4', ?)", quantity("qt_4", "3"))

	spec := MoneyColumnTable{table: table, sources: []moneyColumnSource{
		{legacyID: "fee_id", rate: true, amount: "fee_amount", currency: "fee_currency", perUnit: "fee_per_unit_id"},
		{legacyID: "total_id", amount: "total_amount", currency: "total_currency"},
	}}
	q := sqlc.New(pool)
	b := NewMoneyColumnBackfill(q, q, spec)
	ctx := context.Background()
	cursor, batches := "", 0
	for {
		next, _, done, err := b.Batch(ctx, &backfill.Meter{}, cursor, 2)
		require.NoError(t, err)
		cursor, batches = next, batches+1
		if done {
			break
		}
		require.Less(t, batches, 10)
	}
	assert.Equal(t, "row_4", cursor)

	type money struct{ feeAmount, feeCurrency, feePerUnit, totalAmount, totalCurrency gosql.NullString }
	read := func(id string) money {
		var m money
		require.NoError(t, pool.QueryRow("SELECT CAST(fee_amount AS CHAR), fee_currency, fee_per_unit_id, CAST(total_amount AS CHAR), total_currency FROM `"+table+"` WHERE id = ?", id).
			Scan(&m.feeAmount, &m.feeCurrency, &m.feePerUnit, &m.totalAmount, &m.totalCurrency))
		return m
	}
	var null gosql.NullString
	str := func(s string) gosql.NullString { return gosql.NullString{String: s, Valid: true} }
	dec := func(s string) gosql.NullString { return str(s + ".000000000000000000000000000000") }

	assert.Equal(t, money{dec("25"), str("usd"), str("un_hr"), str("100.500000000000000000000000000000"), str("usd")}, read("row_1"))
	assert.Equal(t, money{dec("12"), str("usd"), str("un_ea"), dec("7"), str("usd")}, read("row_2"))
	assert.Equal(t, money{dec("30"), str("usd"), str("un_min"), null, null}, read("row_3"), "a stale price is brought level with its legacy row")
	assert.Equal(t, money{null, null, null, dec("3"), str("usd")}, read("row_4"), "a row with no legacy price keeps no price")

	// A rerun changes nothing.
	_, _, _, err := b.Batch(ctx, &backfill.Meter{}, "", 10)
	require.NoError(t, err)
	assert.Equal(t, money{dec("25"), str("usd"), str("un_hr"), str("100.500000000000000000000000000000"), str("usd")}, read("row_1"))
}

// TestMoneyColumnBackfill_KeepsAConcurrentWrite checks a row whose columns changed after the batch read them is left
// alone.
func TestMoneyColumnBackfill_KeepsAConcurrentWrite(t *testing.T) {
	pool := testDB(t)
	suffix := time.Now().UnixNano()
	table := fmt.Sprintf("money_backfill_race_%d", suffix)
	lcExec(t, pool, "CREATE TABLE `"+table+"` (id varchar(191) NOT NULL PRIMARY KEY, "+
		"amount_id varchar(191) NULL, amount decimal(65,30) NULL, currency char(3) NULL) DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci")
	t.Cleanup(func() { _, _ = pool.Exec("DROP TABLE `" + table + "`") })
	qid := fmt.Sprintf("qt_race_%d", suffix)
	lcExec(t, pool, "INSERT INTO quantity (id, value, unit_id) VALUES (?, 5, 'un_dollar')", qid)
	t.Cleanup(func() { _, _ = pool.Exec("DELETE FROM quantity WHERE id = ?", qid) })
	lcExec(t, pool, "INSERT INTO `"+table+"` (id, amount_id) VALUES ('row_1', ?)", qid)

	q := sqlc.New(pool)
	b := NewMoneyColumnBackfill(q, q, MoneyColumnTable{table: table, sources: []moneyColumnSource{{legacyID: "amount_id", amount: "amount", currency: "currency"}}})
	rows, err := b.readRows(context.Background(), "", 10)
	require.NoError(t, err)
	require.Len(t, rows, 1)

	// A writer fills the row between the batch's read and its write.
	lcExec(t, pool, "UPDATE `"+table+"` SET amount = 9, currency = 'usd' WHERE id = 'row_1'")
	args := []any{}
	for _, v := range rows[0].want {
		args = append(args, v)
	}
	args = append(args, rows[0].id)
	for _, v := range rows[0].current {
		args = append(args, v)
	}
	_, err = pool.Exec(b.update, args...)
	require.NoError(t, err)

	var amount string
	require.NoError(t, pool.QueryRow("SELECT CAST(amount AS CHAR) FROM `"+table+"` WHERE id = 'row_1'").Scan(&amount))
	assert.Equal(t, "9.000000000000000000000000000000", amount)
}

// TestMoneyColumnTables_StatementsRunOnTheSchema reads a row of each real table through its backfill and fills an
// account with no rows, so a column the spec names wrongly fails here rather than in a deploy.
func TestMoneyColumnTables_StatementsRunOnTheSchema(t *testing.T) {
	pool := testDB(t)
	q := sqlc.New(pool)
	for _, table := range MoneyColumnTables {
		_, err := NewMoneyColumnBackfill(q, q, table).readRows(context.Background(), "", 1)
		require.NoError(t, err, table.table)
	}
	require.NoError(t, syncAccountMoneyColumns(context.Background(), pool, "ac_money_columns_none"))
}
