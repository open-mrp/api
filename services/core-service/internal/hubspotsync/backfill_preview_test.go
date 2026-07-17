package hubspotsync

import (
	"context"
	"testing"
	"time"

	"github.com/augno/api/services/core-service/internal/domain"
	apierror "github.com/augno/api/shared/errors"
)

func TestClassifyCustomer(t *testing.T) {
	acme := CompanyCandidate{HubspotID: "1", Name: "Acme", Domain: "acme.com"}
	acmeDup := CompanyCandidate{HubspotID: "2", Name: "Acme Inc", Domain: "acme.com"}
	byName := CompanyCandidate{HubspotID: "3", Name: "Globex", Domain: ""}

	byDomain := map[string][]CompanyCandidate{
		"acme.com":  {acme},          // unique domain → confident
		"multi.com": {acme, acmeDup}, // two share a domain → ambiguous
	}
	byNameIdx := map[string][]CompanyCandidate{
		"globex": {byName},
	}

	tests := []struct {
		name     string
		customer *domain.Customer
		want     matchTier
	}{
		{
			name:     "unique domain match is confident",
			customer: &domain.Customer{Name: "Whatever", URL: strptr("https://acme.com")},
			want:     matchConfident,
		},
		{
			name:     "multiple domain matches are ambiguous",
			customer: &domain.Customer{Name: "Whatever", URL: strptr("https://multi.com")},
			want:     matchAmbiguous,
		},
		{
			name:     "name-only match is ambiguous (never confident)",
			customer: &domain.Customer{Name: "Globex"},
			want:     matchAmbiguous,
		},
		{
			name:     "domain present but unmatched, with a name match, is ambiguous",
			customer: &domain.Customer{Name: "Globex", URL: strptr("https://nope.com")},
			want:     matchAmbiguous,
		},
		{
			name:     "no domain and no name match is none",
			customer: &domain.Customer{Name: "Unknown Co", URL: strptr("")},
			want:     matchNone,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _ := classifyCustomer(tt.customer, byDomain, byNameIdx)
			if got != tt.want {
				t.Errorf("classifyCustomer() tier = %d, want %d", got, tt.want)
			}
		})
	}
}

// fakeCompanyLister serves canned ListCompanies pages. The embedded interface is nil on purpose: any other method would panic, which keeps the fake honest about what loadCompanyIndex is allowed to touch.
type fakeCompanyLister struct {
	domain.HubspotClient
	pages map[string][]domain.HubspotCompany
	next  map[string]string
}

func (f *fakeCompanyLister) ListCompanies(_ context.Context, cursor string) ([]domain.HubspotCompany, string, *apierror.APIError) {
	return f.pages[cursor], f.next[cursor], nil
}

// TestLoadCompanyIndex_NormalizesDomains pins the normalizer parity between the two sides of the match. HubSpot's domain property is free text, so indexing it verbatim (rather than through deriveDomain) silently sent every www./scheme-qualified company to matchNone, which the backfill then "resolves" by creating a duplicate company.
func TestLoadCompanyIndex_NormalizesDomains(t *testing.T) {
	client := &fakeCompanyLister{
		pages: map[string][]domain.HubspotCompany{
			"": {
				{ID: "1", Name: "Acme", Domain: "www.acme.com"},
				{ID: "2", Name: "Globex", Domain: "https://globex.com"},
				{ID: "3", Name: "Initech", Domain: "  INITECH.COM  "},
			},
		},
		next: map[string]string{"": ""},
	}

	byDomain, _, apiErr := (&service{}).loadCompanyIndex(context.Background(), client)
	if apiErr != nil {
		t.Fatalf("loadCompanyIndex() error = %v", apiErr)
	}

	// Each key is what deriveDomain yields for the matching customer URL, so a customer at "https://www.acme.com" lands on the company HubSpot stored as "www.acme.com".
	for _, want := range []string{"acme.com", "globex.com", "initech.com"} {
		if len(byDomain[want]) != 1 {
			t.Errorf("byDomain[%q] = %d candidates, want 1 (indexed under %v)", want, len(byDomain[want]), keysOf(byDomain))
		}
	}
}

// TestLoadCompanyIndex_StopsOnNonAdvancingCursor guards the paging loop against a portal that echoes the same cursor back, which would otherwise spin forever inside a job.
func TestLoadCompanyIndex_StopsOnNonAdvancingCursor(t *testing.T) {
	client := &fakeCompanyLister{
		pages: map[string][]domain.HubspotCompany{
			"":     {{ID: "1", Name: "Acme", Domain: "acme.com"}},
			"page": {{ID: "2", Name: "Globex", Domain: "globex.com"}},
		},
		next: map[string]string{"": "page", "page": "page"},
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		if _, _, apiErr := (&service{}).loadCompanyIndex(context.Background(), client); apiErr != nil {
			t.Errorf("loadCompanyIndex() error = %v", apiErr)
		}
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("loadCompanyIndex() did not terminate on a non-advancing cursor")
	}
}

func keysOf(m map[string][]CompanyCandidate) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
