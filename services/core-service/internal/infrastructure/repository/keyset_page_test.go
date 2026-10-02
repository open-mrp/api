package repository

import (
	"fmt"
	"strings"
	"testing"
)

func TestKeysetPage_ForcedKeys(t *testing.T) {
	t.Parallel()

	long := make([]string, keysetMaxArms+1)
	for i := range long {
		long[i] = fmt.Sprintf("it_%d", i)
	}
	base := func(item, user []string) keysetPage {
		return keysetPage{
			table: "icl_t", alias: "x", sortColumn: "created_at", createdIndex: "created_idx",
			filters: []keysetFilter{
				{column: "x.item_id", index: "item_idx", values: item},
				{column: "x.user_id", index: "user_idx", values: user},
			},
			where: []string{"x.account_id = ?"}, args: []any{"ac_1"}, desc: true, limit: 26,
		}
	}
	tests := []struct {
		name      string
		page      keysetPage
		longIndex string
		want      string
	}{
		{"unfiltered", base(nil, nil), "", "FORCE INDEX (created_idx)"},
		{"single-valued filter, never beside created", base([]string{"it_1"}, nil), "", "FORCE INDEX (item_idx)"},
		{"long list beside a single-valued filter", base(long, []string{"us_1"}), "", "FORCE INDEX (user_idx, item_idx)"},
		{"long list, unsettled: ranged", base(long, nil), "", "FORCE INDEX (item_idx)"},
		{"long list settled rare: ranged", base(long, nil), "item_idx", "FORCE INDEX (item_idx)"},
		{"long list settled common: created walked alone", base(long, nil), "created_idx", "FORCE INDEX (created_idx)"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			p := tc.page
			p.longIndex = tc.longIndex
			query, _ := p.sql()
			if !strings.Contains(query, tc.want) {
				t.Errorf("sql() = %s\nwant it to contain %s", query, tc.want)
			}
		})
	}
	if got := base(long, []string{"us_1"}).longFilters(); got != nil {
		t.Errorf("longFilters beside a single-valued filter = %v, want none", got)
	}
	if got := base(long, nil).longFilters(); len(got) != 1 || got[0].index != "item_idx" {
		t.Errorf("longFilters = %v, want the item filter", got)
	}
}
