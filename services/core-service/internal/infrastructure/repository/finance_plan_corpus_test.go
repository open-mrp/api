//go:build plans

package repository

import (
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// The finance corpus settles the transaction corpus's merchant the way production's largest tenant is
// settled: about 1.25 allocations per transaction, each applying it to an invoice, grouped into
// settlements of two or three, with invoices sometimes paid by several allocations. Open transactions
// (planTxCorpus leaves a few dozen not fully allocated) are half untouched, half partly applied.
//
// Customers come in families, which a settle flow lists together: production's largest parents have
// dozens of children and a tenth of the account's transactions between them. planFinParent heads one
// like that (a large child and forty from the tail); planFinSmallParent heads three tail children.
//
// A second, small merchant (planFinSmallAccount) settles a few dozen payments over the same years: a
// list keyed on anything but the account reads the large one's rows to page it.
const (
	planFinMarker       = "ac_planfin_marker"
	planFinVersion      = "Finance Plan Corpus v2 / " + planTxCorpusVersion
	planFinSmallAccount = "ac_planfin_small"
	planFinSmallRows    = 50
	planFinInvoices     = 24_000
	planFinParent       = 1
	planFinSmallParent  = 900
	planFinUnallocated  = 500 // i%1000 == planFinUnallocated is an open transaction in planTxCorpus
	planFinNoInvoice    = "in_planfin_none"
	planFinNoSettlement = "tx_planfin_none"
)

// planFinFamilies maps each parent customer to its children.
var planFinFamilies = func() map[int][]int {
	big := []int{2}
	for c := 21; c <= 60; c++ {
		big = append(big, c)
	}
	return map[int][]int{planFinParent: big, planFinSmallParent: {901, 902, 903}}
}()

func planFinSettlementID(s int) string { return fmt.Sprintf("st_planfin_%016d", s) }
func planFinInvoiceID(n int) string    { return fmt.Sprintf("in_planfin_%016d", n) }
func planFinInvoiceNumber(n int) string {
	return fmt.Sprintf("%d", 50_000+n)
}

// planFinAllocation is one allocation of the corpus.
type planFinAllocation struct {
	seq        int
	tx         int
	amount     string
	createdAt  time.Time
	settlement int
	invoice    int
}

// planFinAllocations lays the allocations out in transaction order, so settlements and invoices, which
// are numbered by allocation, are created in time order too.
func planFinAllocations() []planFinAllocation {
	var out []planFinAllocation
	add := func(tx int, amount string, after time.Duration) {
		k := len(out)
		out = append(out, planFinAllocation{
			seq: k, tx: tx, amount: amount, createdAt: planTxCreatedAt(tx).Add(after),
			settlement: k * 2 / 5, invoice: k * planFinInvoices / 37_500,
		})
	}
	for i := range planTxRows {
		switch {
		case i%2000 == planFinUnallocated:
			// Open and untouched.
		case i%1000 == planFinUnallocated:
			add(i, "40", 2*time.Hour)
		case i%4 == 1:
			add(i, "60", 2*time.Hour)
			add(i, "40", 3*time.Hour)
		default:
			add(i, "100", 2*time.Hour)
		}
	}
	return out
}

var planFinCorpusOnce sync.Once

// ensureFinanceCorpus seeds the finance corpus over the transaction corpus, which it seeds first.
func ensureFinanceCorpus(t *testing.T) {
	t.Helper()
	ensureTransactionCorpus(t)
	db := planDB(t)
	planFinCorpusOnce.Do(func() {
		exec := func(query string, args ...any) {
			_, err := db.Exec(query, args...)
			require.NoError(t, err)
		}
		// The family is re-linked every run: reseeding the transaction corpus recreates the relations.
		exec("UPDATE account_relation SET parent_account_relation_id = NULL WHERE owner_account_id = ?", planTxAccount)
		for parent, children := range planFinFamilies {
			for _, c := range children {
				exec("UPDATE account_relation SET parent_account_relation_id = ? WHERE id = ?",
					fmt.Sprintf("ar_plantx_%04d", parent), fmt.Sprintf("ar_plantx_%04d", c))
			}
		}

		var version string
		_ = db.QueryRow("SELECT name FROM account WHERE id = ?", planFinMarker).Scan(&version)
		if version == planFinVersion {
			return
		}
		allocations := planFinAllocations()
		t.Logf("seeding the finance plan corpus (%d allocations); it is kept for later runs", len(allocations))

		exec("DELETE FROM transaction_allocation WHERE id LIKE 'ta\\_planfin\\_%'")
		exec("DELETE FROM quantity WHERE id LIKE 'qy\\_planfin\\_%'")
		exec("DELETE FROM settlement WHERE account_id IN (?, ?)", planTxAccount, planFinSmallAccount)
		exec("DELETE FROM invoice WHERE account_id IN (?, ?)", planTxAccount, planFinSmallAccount)

		const batch = 1_000
		for start := 0; start < planFinInvoices; start += batch {
			var vals []string
			var args []any
			for n := start; n < min(start+batch, planFinInvoices); n++ {
				at := planTxCreatedAt(n * planTxRows / planFinInvoices).Add(-24 * time.Hour)
				vals = append(vals, "(?, ?, ?, ?, ?, ?, ?)")
				args = append(args, planFinInvoiceID(n), planFinInvoiceNumber(n), fmt.Sprintf("so_planfin_%016d", n),
					fmt.Sprintf("ad_planfin_%016d", n%planTxCustomers), planTxAccount, at, at)
			}
			exec("INSERT INTO invoice (id, number, sales_order_id, billing_address_id, account_id, created_at, updated_at) VALUES "+
				strings.Join(vals, ","), args...)
		}

		settlementAt := map[int]time.Time{}
		for _, a := range allocations {
			if _, ok := settlementAt[a.settlement]; !ok {
				settlementAt[a.settlement] = a.createdAt
			}
		}
		settlements := allocations[len(allocations)-1].settlement + 1
		for start := 0; start < settlements; start += batch {
			var vals []string
			var args []any
			for s := start; s < min(start+batch, settlements); s++ {
				at, ok := settlementAt[s]
				require.True(t, ok, "settlement %d has no allocation", s)
				vals = append(vals, "(?, ?, ?, ?, ?, ?)")
				args = append(args, planFinSettlementID(s), fmt.Sprintf("%d", 1+s), planTxAccount, planTxAccountUserID(s%planTxUsers), at, at)
			}
			exec("INSERT INTO settlement (id, number, account_id, responsible_user_id, created_at, updated_at) VALUES "+
				strings.Join(vals, ","), args...)
		}

		for start := 0; start < len(allocations); start += batch {
			var qVals, aVals []string
			var qArgs, aArgs []any
			for _, a := range allocations[start:min(start+batch, len(allocations))] {
				qID := fmt.Sprintf("qy_planfin_%016d", a.seq)
				qVals = append(qVals, "(?, ?, 'un_plantx', ?, ?)")
				qArgs = append(qArgs, qID, a.amount, a.createdAt, a.createdAt)
				typeCode, _, _ := planTxType(a.tx)
				aVals = append(aVals, "(?, ?, ?, ?, ?, ?, ?, ?, ?)")
				aArgs = append(aArgs, fmt.Sprintf("ta_planfin_%016d", a.seq), planTxID(a.tx), qID,
					planFinInvoiceID(a.invoice), planFinSettlementID(a.settlement), planTxAccount, typeCode, a.createdAt, a.createdAt)
			}
			exec("INSERT INTO quantity (id, value, unit_id, created_at, updated_at) VALUES "+strings.Join(qVals, ","), qArgs...)
			exec("INSERT INTO transaction_allocation (id, transaction_id, amount_id, invoice_id, settlement_id, account_id, transaction_type_code, created_at, updated_at) VALUES "+
				strings.Join(aVals, ","), aArgs...)
		}

		// The small tenant: one settled payment a month or so, interleaved in time with the large one.
		exec("DELETE FROM `transaction` WHERE account_id = ?", planFinSmallAccount)
		exec(`INSERT INTO account (id, name, account_type_code, onboarding_status_code) VALUES (?, 'Plan Small Merchant', 'company', 'active')
		      ON DUPLICATE KEY UPDATE name = VALUES(name)`, planFinSmallAccount)
		for k := range planFinSmallRows {
			at := planTxCreatedAt(k * planTxRows / planFinSmallRows)
			id := func(prefix string) string { return fmt.Sprintf("%s_planfin_s%015d", prefix, k) }
			exec("INSERT INTO quantity (id, value, unit_id, created_at, updated_at) VALUES (?, '100', 'un_plantx', ?, ?), (?, '100', 'un_plantx', ?, ?)",
				id("qy"), at, at, id("qy")+"a", at, at)
			exec("INSERT INTO `transaction` (id, number, account_id, customer_account_id, amount_id, transaction_type_code, transaction_method_code, "+
				"is_fully_allocated, funds_received_at, created_at, updated_at) VALUES (?, ?, ?, ?, ?, 'payment', 'check', 1, ?, ?, ?)",
				id("tx"), fmt.Sprintf("%d", 1+k), planFinSmallAccount, planTxCustomerID(0), id("qy"), at, at, at)
			exec("INSERT INTO invoice (id, number, sales_order_id, billing_address_id, account_id, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
				id("in"), fmt.Sprintf("%d", 1+k), id("so"), id("ad"), planFinSmallAccount, at, at)
			exec("INSERT INTO settlement (id, number, account_id, created_at, updated_at) VALUES (?, ?, ?, ?, ?)",
				id("st"), fmt.Sprintf("%d", 1+k), planFinSmallAccount, at, at)
			exec("INSERT INTO transaction_allocation (id, transaction_id, amount_id, invoice_id, settlement_id, account_id, transaction_type_code, created_at, updated_at) "+
				"VALUES (?, ?, ?, ?, ?, ?, 'payment', ?, ?)", id("ta"), id("tx"), id("qy")+"a", id("in"), id("st"), planFinSmallAccount, at, at)
		}

		exec(`INSERT INTO account (id, name, account_type_code, onboarding_status_code) VALUES (?, ?, 'company', 'active')
		      ON DUPLICATE KEY UPDATE name = VALUES(name)`, planFinMarker, planFinVersion)
		exec("ANALYZE TABLE transaction_allocation, settlement, invoice, `transaction`")
	})
}
