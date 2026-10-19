package repository

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/open-mrp/api/services/core-service/internal/domain"
)

// The demand read ranges the window's orders on the issued key; reading every order of the account and
// filtering afterwards is what made schedule previews take seconds.
func TestPooledOrderDemandQuery_ReadsTheWindowsOrders(t *testing.T) {
	t.Parallel()

	start, end := time.Date(2025, 10, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	params := domain.GetPooledOrderDemandParams{AccountID: "ac_1", WindowStart: start, WindowEnd: end, ProductIDs: []string{"pd_1", "pd_2"}}

	for _, tc := range []struct {
		name     string
		excluded []string
		copies   bool
		want     []any
	}{
		{"rows", nil, false, []any{"ac_1", start, end, "pd_1", "pd_2"}},
		{"copies", nil, true, []any{"ac_1", start, end, "pd_1", "pd_2"}},
		{"make-to-order customers", []string{"ac_c1", "ac_c2"}, true, []any{"ac_1", start, end, "ac_c1", "ac_c2", "pd_1", "pd_2"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			query, args := pooledOrderDemandQuery(params, tc.excluded, tc.copies)
			if !reflect.DeepEqual(args, tc.want) {
				t.Errorf("args = %v\nwant %v", args, tc.want)
			}
			if got := strings.Count(query, "?"); got != len(args) {
				t.Errorf("%d placeholders but %d args", got, len(args))
			}
			if !strings.Contains(query, "FROM sales_order so FORCE INDEX (sales_order_owner_type_issued_idx)") {
				t.Errorf("orders are not read off the issued key:\n%s", query)
			}
			if got := strings.Contains(query, "buyer_account_id NOT IN"); got != (len(tc.excluded) > 0) {
				t.Errorf("customer exclusion present = %v with %d excluded:\n%s", got, len(tc.excluded), query)
			}
			if got := strings.Contains(query, "JOIN quantity q"); got == tc.copies {
				t.Errorf("quantity rows read = %v with copies = %v:\n%s", got, tc.copies, query)
			}
			if strings.Contains(query, "NOT EXISTS") {
				t.Errorf("make-to-order customers are resolved per order:\n%s", query)
			}
		})
	}
}
