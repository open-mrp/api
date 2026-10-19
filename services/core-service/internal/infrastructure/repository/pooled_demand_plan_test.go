//go:build plans

package repository

import (
	"context"
	"fmt"
	"regexp"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
)

// pooledDemandReference is the demand query as it was: every order of the account joined to its lines,
// the window, products and make-to-order customers filtered afterwards.
const pooledDemandReference = `SELECT sol.product_id, YEAR(so.issued_at), MONTH(so.issued_at),
    CAST(COALESCE(SUM(q.value * (u.ratio_numerator / u.ratio_denominator)), 0) AS DECIMAL(65,30))
FROM sales_order_line sol
JOIN sales_order so ON so.id = sol.sales_order_id
JOIN quantity q ON q.id = sol.quantity_id
JOIN unit u ON u.id = q.unit_id
WHERE so.owner_account_id = ? AND so.sales_order_type_code = 'sales_order' AND so.sales_order_status_code <> 'estimate'
  AND so.issued_at IS NOT NULL AND so.issued_at >= ? AND so.issued_at <= ?
  AND sol.product_id IN (%s)
  AND NOT EXISTS (
      SELECT 1 FROM account_relation ar LEFT JOIN account_group ag ON ag.id = ar.account_group_id
      WHERE ar.owner_account_id = so.owner_account_id AND ar.counterparty_account_id = so.buyer_account_id
        AND ar.account_relation_role_code = 'customer'
        AND COALESCE(NULLIF(ar.fulfillment_policy_code, ''), ag.fulfillment_policy_code) = 'make_to_order')
GROUP BY sol.product_id, YEAR(so.issued_at), MONTH(so.issued_at)
ORDER BY sol.product_id, YEAR(so.issued_at), MONTH(so.issued_at)`

var regexpDemand = regexp.MustCompile(`FROM sales_order so FORCE INDEX \(sales_order_owner_type_issued_idx\)`)

// TestPooledOrderDemand_AnswersAsBefore requires the demand read to return what the query it replaced
// returned, from the quantity rows and from the copies the lines carry, with make-to-order customers
// resolved from their own setting and from their group's; and to read the window's orders off the
// issued key, never every order of the account.
func TestPooledOrderDemand_AnswersAsBefore(t *testing.T) {
	ensureDeliveryPerformanceCorpus(t)
	db := planDB(t)
	ctx := context.Background()
	exec := func(query string, args ...any) {
		t.Helper()
		_, err := db.Exec(query, args...)
		require.NoError(t, err)
	}
	// The largest customer is make-to-order by its own setting, the rare one through its group.
	exec("UPDATE account_relation SET fulfillment_policy_code = 'make_to_order' WHERE owner_account_id = ? AND counterparty_account_id = ?",
		planAnaAccount, planAnaBuyerID(0))
	exec("UPDATE account_group SET fulfillment_policy_code = 'make_to_order' WHERE id = 'ag_planana_small'")
	t.Cleanup(func() {
		exec("UPDATE account_relation SET fulfillment_policy_code = NULL WHERE owner_account_id = ?", planAnaAccount)
		exec("UPDATE account_group SET fulfillment_policy_code = NULL WHERE id = 'ag_planana_small'")
	})
	fillCorpusLineCopies(t, db, planAnaAccount)
	prev := lineCopiesDone.Load()
	t.Cleanup(func() { lineCopiesDone.Store(prev) })

	var products []string
	for i := range planAnaItems {
		products = append(products, fmt.Sprintf("pd_planana_%05d", i))
	}
	end := planAnaInvoicedAt(planAnaInvoices - 1)
	for _, tc := range []struct {
		name     string
		start    time.Time
		products []string
	}{
		{"a year of every product", end.AddDate(-1, 0, 0), products},
		{"a year of a few products", end.AddDate(-1, 0, 0), products[:5]},
		{"the whole history", planAnaInvoicedAt(0), products},
	} {
		t.Run(tc.name, func(t *testing.T) {
			args := []any{planAnaAccount, tc.start, end}
			want := []domain.PooledMonthlyDemandRow{}
			rows, err := db.QueryContext(ctx, fmt.Sprintf(pooledDemandReference, placeholders(len(tc.products))), append(args, stringsToAny(tc.products)...)...)
			require.NoError(t, err)
			for rows.Next() {
				var row domain.PooledMonthlyDemandRow
				var quantity string
				require.NoError(t, rows.Scan(&row.ProductID, &row.Year, &row.Month, &quantity))
				row.Quantity = decimalToFloat64(quantity)
				want = append(want, row)
			}
			require.NoError(t, rows.Err())
			require.NotEmpty(t, want, "the case must have demand")

			for _, copies := range []bool{false, true} {
				lineCopiesDone.Store(copies)
				edb := &explainingDB{db: db}
				got, apiErr := NewProductionScheduleInputRepo(sqlc.New(edb)).GetPooledOrderDemandByProduct(ctx, domain.GetPooledOrderDemandParams{
					AccountID: planAnaAccount, WindowStart: tc.start, WindowEnd: end, ProductIDs: tc.products,
				})
				require.Nil(t, apiErr)
				require.Equal(t, want, got, "copies=%v", copies)

				var demand *explainedStatement
				for i := range edb.statements {
					if st := &edb.statements[i]; regexpDemand.MatchString(st.query) {
						demand = st
					}
				}
				require.NotNil(t, demand, "the demand read was explained")
				require.Regexp(t, `Index range scan on so using sales_order_owner_type_issued_idx`, demand.plan)
				if copies {
					require.NotRegexp(t, copyReadRe, demand.plan, "a copy read visits a row it copies\n%s", demand.plan)
				}
			}
		})
	}
}
