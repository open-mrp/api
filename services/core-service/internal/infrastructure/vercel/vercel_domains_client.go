// Package vercel implements the portal domain provider on top of the Vercel Domains API. Domains are attached to the single Vercel project that serves the dashboard frontend; Vercel issues and renews TLS certificates automatically once the customer's DNS points at it.
package vercel

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/tracing"
)

const (
	vercelBaseURL = "https://api.vercel.com"

	// apexARecordValue is Vercel's anycast A record target for apex domains.
	apexARecordValue = "76.76.21.21"
	// subdomainCNAMEValue is Vercel's CNAME target for subdomains.
	subdomainCNAMEValue = "cname.vercel-dns.com"
)

var vercelTracer = tracing.GetTracer("core-service.vercel_domains_client")

type providerImpl struct {
	token      string
	projectID  string
	teamID     string
	httpClient *http.Client
}

// NewPortalDomainProvider builds the Vercel-backed portal domain provider. teamID may be empty for personal-scope tokens.
func NewPortalDomainProvider(token, projectID, teamID string) domain.PortalDomainProvider {
	return &providerImpl{
		token:      token,
		projectID:  projectID,
		teamID:     teamID,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// projectDomain is the subset of Vercel's project-domain resource we read.
type projectDomain struct {
	Name         string `json:"name"`
	ApexName     string `json:"apexName"`
	Verified     bool   `json:"verified"`
	Verification []struct {
		Type   string `json:"type"`
		Domain string `json:"domain"`
		Value  string `json:"value"`
		Reason string `json:"reason"`
	} `json:"verification"`
}

type domainConfig struct {
	Misconfigured bool `json:"misconfigured"`
}

type vercelError struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func (p *providerImpl) AddDomain(ctx context.Context, domainName string) (*domain.PortalDomainProviderState, *apierror.APIError) {
	ctx, span := vercelTracer.Start(ctx, "vercel.add_domain")
	defer span.End()

	body := map[string]string{"name": domainName}
	resp, apiErr := p.doRequest(ctx, http.MethodPost, fmt.Sprintf("/v10/projects/%s/domains", url.PathEscape(p.projectID)), body)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	defer closeBody(resp)

	if resp.StatusCode >= 400 {
		vErr := readVercelError(resp)
		switch vErr.Error.Code {
		// The domain is already attached to this project — an idempotent retry of a prior attempt. Fall through to reading its current state.
		case "domain_already_exists", "domain_exists":
		case "domain_already_in_use", "not_authorized":
			return nil, tracing.Trace(span, apierror.NewConflictErrorWithParam("This domain is already in use by another project. Remove it there first or contact support.", "domain"))
		case "invalid_domain", "forbidden_domain":
			return nil, tracing.Trace(span, apierror.NewValidationErrorWithParam("This domain cannot be used: "+vErr.Error.Message, "domain"))
		default:
			return nil, tracing.Trace(span, apierror.NewInternalError(fmt.Errorf("vercel add domain: %s (%s)", vErr.Error.Message, vErr.Error.Code), "The domain provider rejected the request."))
		}
	}

	return p.GetDomainState(ctx, domainName)
}

func (p *providerImpl) GetDomainState(ctx context.Context, domainName string) (*domain.PortalDomainProviderState, *apierror.APIError) {
	ctx, span := vercelTracer.Start(ctx, "vercel.get_domain_state")
	defer span.End()

	resp, apiErr := p.doRequest(ctx, http.MethodGet, fmt.Sprintf("/v9/projects/%s/domains/%s", url.PathEscape(p.projectID), url.PathEscape(domainName)), nil)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	defer closeBody(resp)

	if resp.StatusCode == http.StatusNotFound {
		return nil, tracing.Trace(span, apierror.NewResourceNotFoundError("Domain is not registered with the provider."))
	}
	if resp.StatusCode >= 400 {
		vErr := readVercelError(resp)
		return nil, tracing.Trace(span, apierror.NewInternalError(fmt.Errorf("vercel get domain: %s (%s)", vErr.Error.Message, vErr.Error.Code), "Failed to read domain state from the provider."))
	}

	var dom projectDomain
	if err := json.NewDecoder(resp.Body).Decode(&dom); err != nil {
		return nil, tracing.Trace(span, apierror.NewInternalError(err, "Failed to decode provider domain response."))
	}

	// Ownership verification pending (domain claimed by another Vercel account): attempt the TXT-challenge verification so a customer who just published the record flips immediately, then re-read.
	if !dom.Verified && len(dom.Verification) > 0 {
		if verified := p.tryVerify(ctx, domainName); verified != nil {
			dom = *verified
		}
	}

	state := &domain.PortalDomainProviderState{
		Verified:   dom.Verified,
		DNSRecords: requiredDNSRecords(dom),
	}

	// The routing config check is only meaningful once ownership is established.
	if dom.Verified {
		cfg, apiErr := p.getDomainConfig(ctx, domainName)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		state.Misconfigured = cfg.Misconfigured
	} else {
		state.Misconfigured = true
	}

	return state, nil
}

func (p *providerImpl) RemoveDomain(ctx context.Context, domainName string) *apierror.APIError {
	ctx, span := vercelTracer.Start(ctx, "vercel.remove_domain")
	defer span.End()

	resp, apiErr := p.doRequest(ctx, http.MethodDelete, fmt.Sprintf("/v9/projects/%s/domains/%s", url.PathEscape(p.projectID), url.PathEscape(domainName)), nil)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	defer closeBody(resp)

	// Removing a domain that is already gone is a successful idempotent retry.
	if resp.StatusCode >= 400 && resp.StatusCode != http.StatusNotFound {
		vErr := readVercelError(resp)
		return tracing.Trace(span, apierror.NewInternalError(fmt.Errorf("vercel remove domain: %s (%s)", vErr.Error.Message, vErr.Error.Code), "Failed to remove domain from the provider."))
	}

	return nil
}

// tryVerify triggers Vercel's TXT-challenge verification and returns the refreshed domain when the call succeeds, or nil to keep the original state.
func (p *providerImpl) tryVerify(ctx context.Context, domainName string) *projectDomain {
	resp, apiErr := p.doRequest(ctx, http.MethodPost, fmt.Sprintf("/v9/projects/%s/domains/%s/verify", url.PathEscape(p.projectID), url.PathEscape(domainName)), nil)
	if apiErr != nil {
		return nil
	}
	defer closeBody(resp)

	if resp.StatusCode >= 400 {
		return nil
	}

	var dom projectDomain
	if err := json.NewDecoder(resp.Body).Decode(&dom); err != nil {
		return nil
	}
	return &dom
}

func (p *providerImpl) getDomainConfig(ctx context.Context, domainName string) (*domainConfig, *apierror.APIError) {
	path := fmt.Sprintf("/v6/domains/%s/config", url.PathEscape(domainName))
	resp, apiErr := p.doRequest(ctx, http.MethodGet, path, nil)
	if apiErr != nil {
		return nil, apiErr
	}
	defer closeBody(resp)

	if resp.StatusCode >= 400 {
		vErr := readVercelError(resp)
		return nil, apierror.NewInternalError(fmt.Errorf("vercel domain config: %s (%s)", vErr.Error.Message, vErr.Error.Code), "Failed to read domain configuration from the provider.")
	}

	var cfg domainConfig
	if err := json.NewDecoder(resp.Body).Decode(&cfg); err != nil {
		return nil, apierror.NewInternalError(err, "Failed to decode provider domain configuration.")
	}
	return &cfg, nil
}

// requiredDNSRecords derives the records the customer must publish: pending TXT ownership challenges first, then the routing record (A for apex domains, CNAME for subdomains).
func requiredDNSRecords(dom projectDomain) []domain.PortalDNSRecord {
	records := make([]domain.PortalDNSRecord, 0, len(dom.Verification)+1)

	for _, v := range dom.Verification {
		records = append(records, domain.PortalDNSRecord{
			Type:   constants.DNSRecordType(v.Type),
			Name:   v.Domain,
			Value:  v.Value,
			Reason: constants.DNSRecordReasonOwnership,
		})
	}

	if dom.Name == dom.ApexName {
		records = append(records, domain.PortalDNSRecord{
			Type:   constants.DNSRecordTypeA,
			Name:   dom.Name,
			Value:  apexARecordValue,
			Reason: constants.DNSRecordReasonRouting,
		})
	} else {
		records = append(records, domain.PortalDNSRecord{
			Type:   constants.DNSRecordTypeCNAME,
			Name:   dom.Name,
			Value:  subdomainCNAMEValue,
			Reason: constants.DNSRecordReasonRouting,
		})
	}

	return records
}

func (p *providerImpl) doRequest(ctx context.Context, method, path string, body any) (*http.Response, *apierror.APIError) {
	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, apierror.NewInternalError(err, "Failed to marshal provider request body.")
		}
		reqBody = strings.NewReader(string(b))
	}

	fullURL := vercelBaseURL + path
	if p.teamID != "" {
		sep := "?"
		if strings.Contains(path, "?") {
			sep = "&"
		}
		fullURL += sep + "teamId=" + url.QueryEscape(p.teamID)
	}

	req, err := http.NewRequestWithContext(ctx, method, fullURL, reqBody)
	if err != nil {
		return nil, apierror.NewInternalError(err, "Failed to create provider HTTP request.")
	}
	req.Header.Set("Authorization", "Bearer "+p.token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := p.httpClient.Do(req) // #nosec G704 -- URL from server-configured Vercel API base
	if err != nil {
		return nil, apierror.NewInternalError(err, "Failed to reach the domain provider.")
	}
	return resp, nil
}

func readVercelError(resp *http.Response) vercelError {
	var vErr vercelError
	_ = json.NewDecoder(resp.Body).Decode(&vErr)
	if vErr.Error.Message == "" {
		vErr.Error.Message = resp.Status
	}
	return vErr
}

func closeBody(resp *http.Response) {
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()
}
