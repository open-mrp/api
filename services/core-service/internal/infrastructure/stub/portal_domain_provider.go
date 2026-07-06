package stub

import (
	"context"
	"strings"
	"sync"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// PortalDomainProvider is an in-memory portal domain provider for test/dev mode. A domain reports unverified on its first state read and verified from the second read onward, so tests can exercise both the pending and verified paths of the verify action.
type PortalDomainProvider struct {
	mu         sync.Mutex
	stateReads map[string]int
}

func NewPortalDomainProvider() *PortalDomainProvider {
	return &PortalDomainProvider{stateReads: map[string]int{}}
}

func (p *PortalDomainProvider) AddDomain(_ context.Context, domainName string) (*domain.PortalDomainProviderState, *apierror.APIError) {
	return &domain.PortalDomainProviderState{
		Verified:      false,
		Misconfigured: true,
		DNSRecords:    stubDNSRecords(domainName),
	}, nil
}

func (p *PortalDomainProvider) GetDomainState(_ context.Context, domainName string) (*domain.PortalDomainProviderState, *apierror.APIError) {
	p.mu.Lock()
	p.stateReads[domainName]++
	verified := p.stateReads[domainName] > 1
	p.mu.Unlock()

	return &domain.PortalDomainProviderState{
		Verified:      verified,
		Misconfigured: !verified,
		DNSRecords:    stubDNSRecords(domainName),
	}, nil
}

func (p *PortalDomainProvider) RemoveDomain(_ context.Context, domainName string) *apierror.APIError {
	p.mu.Lock()
	delete(p.stateReads, domainName)
	p.mu.Unlock()
	return nil
}

func stubDNSRecords(domainName string) []domain.PortalDNSRecord {
	// Treat anything with more than two labels as a subdomain, mirroring the real provider's apex/subdomain split.
	if strings.Count(domainName, ".") >= 2 {
		return []domain.PortalDNSRecord{{Type: constants.DNSRecordTypeCNAME, Name: domainName, Value: "cname.vercel-dns.com", Reason: constants.DNSRecordReasonRouting}}
	}
	return []domain.PortalDNSRecord{{Type: constants.DNSRecordTypeA, Name: domainName, Value: "76.76.21.21", Reason: constants.DNSRecordReasonRouting}}
}
