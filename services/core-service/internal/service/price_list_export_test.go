package service

import (
	"strings"
	"testing"
	"time"
)

// The browser names a downloaded export from the last segment of its object key, so an export that is not a spreadsheet has to say so in the key. Getting this wrong ships a PDF named .xlsx, which refuses to open.
func TestExportObjectKey_ExtensionFollowsTheFormat(t *testing.T) {
	at := time.Date(2026, 8, 14, 0, 0, 0, 0, time.UTC)

	cases := []struct {
		name    string
		slug    string
		ext     string
		wantEnd string
	}{
		{"price list is a pdf", "price_list", "pdf", "price_list_export_08-14-2026.pdf"},
		{"spreadsheet exports are unchanged", "departments", "", "departments_export_08-14-2026.xlsx"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			key := exportObjectKey("ac_1", tc.slug, "job_1", at, tc.ext)
			if !strings.HasSuffix(key, tc.wantEnd) {
				t.Errorf("key = %q, want it to end in %q", key, tc.wantEnd)
			}
			if !strings.HasPrefix(key, "exports/ac_1/"+tc.slug+"/job_1/") {
				t.Errorf("key = %q, want it scoped to the account, resource and job", key)
			}
		})
	}
}

// The price list registers itself as a PDF; an empty Ext here would silently produce a spreadsheet name.
func TestPriceListExportSpec_DeclaresPDF(t *testing.T) {
	spec := (&accountPriceSvcImpl{}).priceListExportSpec()

	if spec.Ext != "pdf" {
		t.Errorf("Ext = %q, want pdf", spec.Ext)
	}
	if spec.Slug != "price_list" {
		t.Errorf("Slug = %q, want price_list", spec.Slug)
	}
}
