package stub

import (
	"context"
	"strings"
	"sync"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// PortalDomainProvider is an in-memory portal domain provider for test/dev mode. A domain walks the real lifecycle across successive state reads: read #1 is unverified (pending), read #2 is verified+routing but not yet serving (securing, as if the TLS certificate were still issuing), and reads #3+ are verified and serving. This lets tests exercise the pending, securing, and verified paths of the verify action.
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
		Serving:       false,
		DNSRecords:    stubDNSRecords(domainName),
	}, nil
}

func (p *PortalDomainProvider) GetDomainState(_ context.Context, domainName string) (*domain.PortalDomainProviderState, *apierror.APIError) {
	p.mu.Lock()
	p.stateReads[domainName]++
	reads := p.stateReads[domainName]
	p.mu.Unlock()

	verified := reads > 1
	serving := reads > 2

	return &domain.PortalDomainProviderState{
		Verified:      verified,
		Misconfigured: !verified,
		Serving:       serving,
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
