package hubspotsync

import (
	"testing"

	"github.com/augno/api/services/core-service/internal/domain"
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
