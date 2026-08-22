// Package hubspot implements a thin HTTP client over the HubSpot CRM v3/v4 API, scoped to the company/contact/deal operations the sales-order sync needs.
//
// Idempotency note: deals are keyed on the custom property `augno_sales_order_id`, which EnsureDealProperties creates on first use in the connected HubSpot portal. Contacts dedupe natively on email; companies are matched on domain/name by the caller.
package hubspot

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/tracing"

	"go.opentelemetry.io/otel/trace"
)

const (
	hubspotBaseURL = "https://api.hubapi.com"

	// dealSalesOrderProperty is the custom HubSpot deal property that stores the OpenMRP order id for idempotent upserts.
	dealSalesOrderProperty = "augno_sales_order_id"

	// maxRetries bounds retries on rate-limit (429) and server (5xx) responses. The inbox consumer provides the outer retry loop.
	maxRetries = 3

	// maxRetryAfter caps how long a Retry-After header can hold this goroutine. HubSpot's daily-limit 429 asks for seconds-until-midnight; past this cap it is cheaper to fail and let the queue redeliver than to occupy a consumer slot.
	maxRetryAfter = 60 * time.Second
)

var hubspotTracer = tracing.GetTracer("core-service.hubspot_client")

type clientImpl struct {
	accessToken string
	httpClient  *http.Client
}

// ClientFactory builds HubspotClient instances from a decrypted Private App access token.
type ClientFactory struct{}

func NewClientFactory() *ClientFactory {
	return &ClientFactory{}
}

func (f *ClientFactory) Build(accessToken string) domain.HubspotClient {
	return &clientImpl{
		accessToken: accessToken,
		httpClient:  &http.Client{Timeout: 30 * time.Second},
	}
}

// doRequest sends a request to the HubSpot API, retrying with backoff on 429 and 5xx responses.
func (c *clientImpl) doRequest(ctx context.Context, method, path string, body any) (*http.Response, *apierror.APIError) {
	var bodyBytes []byte
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, apierror.NewInternalError(err, "Failed to marshal HubSpot request body.")
		}
		bodyBytes = b
	}

	var lastResp *http.Response
	for attempt := 0; attempt <= maxRetries; attempt++ {
		var reqBody io.Reader
		if bodyBytes != nil {
			reqBody = bytes.NewReader(bodyBytes)
		}
		req, err := http.NewRequestWithContext(ctx, method, hubspotBaseURL+path, reqBody)
		if err != nil {
			return nil, apierror.NewInternalError(err, "Failed to create HubSpot HTTP request.")
		}
		req.Header.Set("Authorization", "Bearer "+c.accessToken)
		req.Header.Set("Content-Type", "application/json")

		resp, err := c.httpClient.Do(req) // #nosec G704 -- URL from server-configured HubSpot API base
		if err != nil {
			if attempt < maxRetries {
				if sleepErr := backoff(ctx, attempt, 0); sleepErr != nil {
					return nil, apierror.NewInternalError(sleepErr, "HubSpot request canceled.")
				}
				continue
			}
			return nil, apierror.NewInternalError(err, "HubSpot API request failed.")
		}

		if (resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500) && attempt < maxRetries {
			retryAfter := parseRetryAfter(resp)
			_ = resp.Body.Close()
			if sleepErr := backoff(ctx, attempt, retryAfter); sleepErr != nil {
				return nil, apierror.NewInternalError(sleepErr, "HubSpot request canceled.")
			}
			continue
		}

		lastResp = resp
		break
	}
	return lastResp, nil
}

// parseError reads a non-2xx response and maps it to an APIError. 429 and 5xx are transient (the inbox consumer will retry); other 4xx are permanent.
func (c *clientImpl) parseError(resp *http.Response) *apierror.APIError {
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)

	var er apiErrorResponse
	_ = json.Unmarshal(b, &er)
	msg := er.Message
	if msg == "" {
		msg = string(b)
	}

	switch {
	case resp.StatusCode == http.StatusTooManyRequests:
		return apierror.NewRateLimitExceededError("HubSpot API rate limit exceeded.")
	case resp.StatusCode >= 500:
		return apierror.NewInternalError(fmt.Errorf("hubspot status %d: %s", resp.StatusCode, msg), "HubSpot API server error.")
	case resp.StatusCode == http.StatusUnauthorized:
		// A rejected token is permanent (retrying cannot mint a valid one) but, unlike bad data, it means every sync for this account is dead until someone reconnects. Keep it distinguishable from a validation error so callers can say so rather than reporting an opaque HUBSPOT: message.
		return apierror.NewAuthenticationError("HubSpot rejected the access token. Reconnect the HubSpot integration with a valid Private App token. (" + msg + ")")
	case resp.StatusCode == http.StatusForbidden:
		// 403 means the token is real but under-scoped, which is a different fix from 401's "get a new token". The token is only prefix-checked when saved, so this is the first point an operator learns a scope box was left unticked — name them rather than echoing HubSpot's generic message.
		return apierror.NewAuthenticationError("The HubSpot Private App token is missing a required scope. It needs crm.objects.companies/contacts/deals read+write and crm.schemas.deals read+write. (" + msg + ")")
	default:
		return apierror.NewValidationError("HUBSPOT: " + msg)
	}
}

// EnsureDealProperties creates the custom deal properties the sync depends on if they're missing from the connected portal. It is idempotent: an existing property (200) or a concurrent create (409) is treated as success.
func (c *clientImpl) EnsureDealProperties(ctx context.Context) *apierror.APIError {
	ctx, span := hubspotTracer.Start(ctx, "hubspot.ensure_deal_properties")
	defer span.End()

	resp, apiErr := c.doRequest(ctx, http.MethodGet, "/crm/v3/properties/deals/"+dealSalesOrderProperty, nil)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if resp.StatusCode == http.StatusOK {
		_ = resp.Body.Close()
		return nil
	}
	if resp.StatusCode != http.StatusNotFound {
		return tracing.Trace(span, c.parseError(resp))
	}
	_ = resp.Body.Close()

	def := propertyDefinition{
		Name:           dealSalesOrderProperty,
		Label:          "OpenMRP Sales Order ID",
		Type:           "string",
		FieldType:      "text",
		GroupName:      "dealinformation",
		HasUniqueValue: true,
	}
	resp, apiErr = c.doRequest(ctx, http.MethodPost, "/crm/v3/properties/deals", def)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusConflict {
		return tracing.Trace(span, c.parseError(resp))
	}
	return nil
}

// --- Companies ---

func (c *clientImpl) SearchCompaniesByDomain(ctx context.Context, domainName string) ([]domain.HubspotCompany, *apierror.APIError) {
	ctx, span := hubspotTracer.Start(ctx, "hubspot.search_companies_by_domain")
	defer span.End()
	return c.searchCompanies(ctx, span, "domain", domainName)
}

func (c *clientImpl) SearchCompaniesByName(ctx context.Context, name string) ([]domain.HubspotCompany, *apierror.APIError) {
	ctx, span := hubspotTracer.Start(ctx, "hubspot.search_companies_by_name")
	defer span.End()
	return c.searchCompanies(ctx, span, "name", name)
}

func (c *clientImpl) searchCompanies(ctx context.Context, span trace.Span, property, value string) ([]domain.HubspotCompany, *apierror.APIError) {
	reqBody := searchRequest{
		FilterGroups: []filterGroup{{Filters: []filter{{PropertyName: property, Operator: "EQ", Value: value}}}},
		Properties:   []string{"name", "domain", "lifecyclestage"},
		Limit:        10,
	}
	resp, apiErr := c.doRequest(ctx, http.MethodPost, "/crm/v3/objects/companies/search", reqBody)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, tracing.Trace(span, c.parseError(resp))
	}
	defer resp.Body.Close()

	var result searchResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, tracing.Trace(span, apierror.NewInternalError(err, "Failed to parse HubSpot company search response."))
	}

	companies := make([]domain.HubspotCompany, 0, len(result.Results))
	for _, r := range result.Results {
		companies = append(companies, mapCompany(r))
	}
	return companies, nil
}

func (c *clientImpl) ListCompanies(ctx context.Context, cursor string) ([]domain.HubspotCompany, string, *apierror.APIError) {
	ctx, span := hubspotTracer.Start(ctx, "hubspot.list_companies")
	defer span.End()

	path := "/crm/v3/objects/companies?limit=100&properties=name,domain,lifecyclestage"
	if cursor != "" {
		path += "&after=" + url.QueryEscape(cursor)
	}
	resp, apiErr := c.doRequest(ctx, http.MethodGet, path, nil)
	if apiErr != nil {
		return nil, "", tracing.Trace(span, apiErr)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, "", tracing.Trace(span, c.parseError(resp))
	}
	defer resp.Body.Close()

	var result listResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, "", tracing.Trace(span, apierror.NewInternalError(err, "Failed to parse HubSpot company list response."))
	}

	companies := make([]domain.HubspotCompany, 0, len(result.Results))
	for _, r := range result.Results {
		companies = append(companies, mapCompany(r))
	}
	next := ""
	if result.Paging != nil && result.Paging.Next != nil {
		next = result.Paging.Next.After
	}
	return companies, next, nil
}

func (c *clientImpl) CreateCompany(ctx context.Context, company domain.HubspotCompany) (*domain.HubspotCompany, *apierror.APIError) {
	ctx, span := hubspotTracer.Start(ctx, "hubspot.create_company")
	defer span.End()

	props := map[string]string{}
	setIfNotEmpty(props, "name", company.Name)
	setIfNotEmpty(props, "domain", company.Domain)
	setIfNotEmpty(props, "lifecyclestage", company.Lifecycle)

	resp, apiErr := c.doRequest(ctx, http.MethodPost, "/crm/v3/objects/companies", objectInput{Properties: props})
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return nil, tracing.Trace(span, c.parseError(resp))
	}
	defer resp.Body.Close()

	var obj objectResponse
	if err := json.NewDecoder(resp.Body).Decode(&obj); err != nil {
		return nil, tracing.Trace(span, apierror.NewInternalError(err, "Failed to parse HubSpot create-company response."))
	}
	created := mapCompany(obj)
	return &created, nil
}

func (c *clientImpl) UpdateCompany(ctx context.Context, id string, company domain.HubspotCompany) *apierror.APIError {
	ctx, span := hubspotTracer.Start(ctx, "hubspot.update_company")
	defer span.End()

	props := map[string]string{}
	setIfNotEmpty(props, "name", company.Name)
	setIfNotEmpty(props, "domain", company.Domain)
	setIfNotEmpty(props, "lifecyclestage", company.Lifecycle)
	if len(props) == 0 {
		return nil
	}

	resp, apiErr := c.doRequest(ctx, http.MethodPatch, "/crm/v3/objects/companies/"+url.PathEscape(id), objectInput{Properties: props})
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if resp.StatusCode != http.StatusOK {
		return tracing.Trace(span, c.parseError(resp))
	}
	_ = resp.Body.Close()
	return nil
}

// --- Contacts ---

func (c *clientImpl) UpsertContactByEmail(ctx context.Context, contact domain.HubspotContact) (*domain.HubspotContact, *apierror.APIError) {
	ctx, span := hubspotTracer.Start(ctx, "hubspot.upsert_contact_by_email")
	defer span.End()

	existing, apiErr := c.searchContactByEmail(ctx, contact.Email)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	props := map[string]string{}
	setIfNotEmpty(props, "email", contact.Email)
	setIfNotEmpty(props, "firstname", contact.FirstName)
	setIfNotEmpty(props, "lastname", contact.LastName)
	setIfNotEmpty(props, "phone", contact.Phone)
	setIfNotEmpty(props, "lifecyclestage", contact.Lifecycle)

	if existing != nil {
		resp, apiErr := c.doRequest(ctx, http.MethodPatch, "/crm/v3/objects/contacts/"+url.PathEscape(existing.ID), objectInput{Properties: props})
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		if resp.StatusCode != http.StatusOK {
			return nil, tracing.Trace(span, c.parseError(resp))
		}
		_ = resp.Body.Close()
		updated := *existing
		return &updated, nil
	}

	resp, apiErr := c.doRequest(ctx, http.MethodPost, "/crm/v3/objects/contacts", objectInput{Properties: props})
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return nil, tracing.Trace(span, c.parseError(resp))
	}
	defer resp.Body.Close()

	var obj objectResponse
	if err := json.NewDecoder(resp.Body).Decode(&obj); err != nil {
		return nil, tracing.Trace(span, apierror.NewInternalError(err, "Failed to parse HubSpot create-contact response."))
	}
	return &domain.HubspotContact{ID: obj.ID, Email: obj.Properties["email"]}, nil
}

func (c *clientImpl) searchContactByEmail(ctx context.Context, email string) (*domain.HubspotContact, *apierror.APIError) {
	reqBody := searchRequest{
		FilterGroups: []filterGroup{{Filters: []filter{{PropertyName: "email", Operator: "EQ", Value: email}}}},
		Properties:   []string{"email"},
		Limit:        1,
	}
	resp, apiErr := c.doRequest(ctx, http.MethodPost, "/crm/v3/objects/contacts/search", reqBody)
	if apiErr != nil {
		return nil, apiErr
	}
	if resp.StatusCode != http.StatusOK {
		return nil, c.parseError(resp)
	}
	defer resp.Body.Close()

	var result searchResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, apierror.NewInternalError(err, "Failed to parse HubSpot contact search response.")
	}
	if len(result.Results) == 0 {
		return nil, nil
	}
	r := result.Results[0]
	return &domain.HubspotContact{ID: r.ID, Email: r.Properties["email"]}, nil
}

// --- Deals ---

func (c *clientImpl) SearchDealBySalesOrderID(ctx context.Context, salesOrderID string) (*domain.HubspotDeal, *apierror.APIError) {
	ctx, span := hubspotTracer.Start(ctx, "hubspot.search_deal_by_sales_order_id")
	defer span.End()

	reqBody := searchRequest{
		FilterGroups: []filterGroup{{Filters: []filter{{PropertyName: dealSalesOrderProperty, Operator: "EQ", Value: salesOrderID}}}},
		Properties:   []string{"dealname", "amount", dealSalesOrderProperty},
		Limit:        1,
	}
	resp, apiErr := c.doRequest(ctx, http.MethodPost, "/crm/v3/objects/deals/search", reqBody)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, tracing.Trace(span, c.parseError(resp))
	}
	defer resp.Body.Close()

	var result searchResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, tracing.Trace(span, apierror.NewInternalError(err, "Failed to parse HubSpot deal search response."))
	}
	if len(result.Results) == 0 {
		return nil, nil
	}
	r := result.Results[0]
	return &domain.HubspotDeal{ID: r.ID, Name: r.Properties["dealname"], Amount: r.Properties["amount"], SalesOrderID: salesOrderID}, nil
}

func (c *clientImpl) CreateDeal(ctx context.Context, deal domain.HubspotDeal) (*domain.HubspotDeal, *apierror.APIError) {
	ctx, span := hubspotTracer.Start(ctx, "hubspot.create_deal")
	defer span.End()

	resp, apiErr := c.doRequest(ctx, http.MethodPost, "/crm/v3/objects/deals", objectInput{Properties: dealProperties(deal)})
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return nil, tracing.Trace(span, c.parseError(resp))
	}
	defer resp.Body.Close()

	var obj objectResponse
	if err := json.NewDecoder(resp.Body).Decode(&obj); err != nil {
		return nil, tracing.Trace(span, apierror.NewInternalError(err, "Failed to parse HubSpot create-deal response."))
	}
	created := deal
	created.ID = obj.ID
	return &created, nil
}

func (c *clientImpl) UpdateDeal(ctx context.Context, id string, deal domain.HubspotDeal) *apierror.APIError {
	ctx, span := hubspotTracer.Start(ctx, "hubspot.update_deal")
	defer span.End()

	resp, apiErr := c.doRequest(ctx, http.MethodPatch, "/crm/v3/objects/deals/"+url.PathEscape(id), objectInput{Properties: dealProperties(deal)})
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if resp.StatusCode != http.StatusOK {
		return tracing.Trace(span, c.parseError(resp))
	}
	_ = resp.Body.Close()
	return nil
}

// --- Associations ---

func (c *clientImpl) Associate(ctx context.Context, fromType, fromID, toType, toID string) *apierror.APIError {
	ctx, span := hubspotTracer.Start(ctx, "hubspot.associate")
	defer span.End()

	path := fmt.Sprintf("/crm/v4/objects/%s/%s/associations/default/%s/%s",
		url.PathEscape(fromType), url.PathEscape(fromID), url.PathEscape(toType), url.PathEscape(toID))
	resp, apiErr := c.doRequest(ctx, http.MethodPut, path, nil)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return tracing.Trace(span, c.parseError(resp))
	}
	_ = resp.Body.Close()
	return nil
}

// --- helpers ---

func dealProperties(deal domain.HubspotDeal) map[string]string {
	props := map[string]string{}
	setIfNotEmpty(props, "dealname", deal.Name)
	setIfNotEmpty(props, "amount", deal.Amount)
	setIfNotEmpty(props, "pipeline", deal.PipelineID)
	setIfNotEmpty(props, "dealstage", deal.StageID)
	setIfNotEmpty(props, dealSalesOrderProperty, deal.SalesOrderID)
	if !deal.CloseDate.IsZero() {
		props["closedate"] = deal.CloseDate.UTC().Format(time.RFC3339)
	}
	return props
}

func mapCompany(obj objectResponse) domain.HubspotCompany {
	return domain.HubspotCompany{
		ID:        obj.ID,
		Name:      obj.Properties["name"],
		Domain:    obj.Properties["domain"],
		Lifecycle: obj.Properties["lifecyclestage"],
	}
}

func setIfNotEmpty(m map[string]string, key, value string) {
	if value != "" {
		m[key] = value
	}
}

// parseRetryAfter returns the Retry-After header value in seconds, or 0 if absent/unparseable. RFC 7231 permits either a delay in seconds or an HTTP-date; falling back to the default backoff on a date would hammer a server that asked for a long pause.
func parseRetryAfter(resp *http.Response) int {
	v := resp.Header.Get("Retry-After")
	if v == "" {
		return 0
	}
	if secs, err := strconv.Atoi(v); err == nil {
		if secs < 0 {
			return 0
		}
		return secs
	}
	if when, err := http.ParseTime(v); err == nil {
		if secs := int(time.Until(when).Seconds()); secs > 0 {
			return secs
		}
	}
	return 0
}

// backoff sleeps before the next retry, honoring Retry-After when provided, otherwise exponential (0.5s, 1s, 2s, ...). It returns the context error if the context is canceled.
func backoff(ctx context.Context, attempt, retryAfterSecs int) error {
	delay := time.Duration(retryAfterSecs) * time.Second
	if delay == 0 {
		delay = 500 * time.Millisecond * (1 << attempt)
	}
	// A daily-limit 429 carries a Retry-After of seconds-until-midnight, which would park this worker for hours holding its consumer slot. Give up instead and let the queue redeliver, which reschedules the work without pinning a goroutine.
	if delay > maxRetryAfter {
		return fmt.Errorf("hubspot asked to retry after %s, beyond the %s cap", delay, maxRetryAfter)
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
