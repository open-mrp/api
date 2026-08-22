package shippo

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/tracing"
)

const shippoBaseURL = "https://api.goshippo.com"

var shippoTracer = tracing.GetTracer("core-service.shippo_client")

// shippoRequestTimeout bounds a single Shippo call. Live rating is the slow path and
// carriers occasionally stall; without a ceiling a hung call holds the request open.
const shippoRequestTimeout = 15 * time.Second

type clientImpl struct {
	apiKey     string
	httpClient *http.Client
	// Points every call at the Shippo API; tests redirect it at a stub server.
	baseURL string

	// Rate shopping resolves a published-rate account per carrier off the same list. The client is per-request, so one fetch serves them all.
	carrierAccountsOnce sync.Once
	carrierAccounts     []CarrierAccount
}

// ClientFactory creates ShippoClient instances from API keys.
type ClientFactory struct{}

func NewClientFactory() *ClientFactory {
	return &ClientFactory{}
}

func (f *ClientFactory) Build(apiKey string) domain.ShippoClient {
	return &clientImpl{
		apiKey:     apiKey,
		httpClient: &http.Client{Timeout: shippoRequestTimeout},
		baseURL:    shippoBaseURL,
	}
}

func (c *clientImpl) authHeader() string {
	return "ShippoToken " + c.apiKey
}

func (c *clientImpl) doRequest(ctx context.Context, method, path string, body any) (*http.Response, *apierror.APIError) {
	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, apierror.NewInternalError(err, "Failed to marshal Shippo request body.")
		}
		reqBody = bytes.NewReader(b)
	}

	base := c.baseURL
	if base == "" {
		base = shippoBaseURL
	}

	req, err := http.NewRequestWithContext(ctx, method, base+path, reqBody)
	if err != nil {
		return nil, apierror.NewInternalError(err, "Failed to create Shippo HTTP request.")
	}
	req.Header.Set("Authorization", c.authHeader())
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req) // #nosec G704 -- URL from server-configured Shippo API base
	if err != nil {
		return nil, apierror.NewInternalError(err, "Shippo API request failed.")
	}

	return resp, nil
}

func (c *clientImpl) parseErrorResponse(resp *http.Response) *apierror.APIError {
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)

	var apiErr APIError
	_ = json.Unmarshal(b, &apiErr)

	msg := "Shippo API error"
	if apiErr.Detail != "" {
		msg = apiErr.Detail
	} else if apiErr.Message != "" {
		msg = apiErr.Message
	} else if apiErr.Error != "" {
		msg = apiErr.Error
	} else if len(apiErr.NonFieldErrors) > 0 {
		msg = strings.Join(apiErr.NonFieldErrors, "; ")
	} else if len(b) > 0 {
		msg = string(b)
	}

	if resp.StatusCode == http.StatusBadRequest {
		return apierror.NewValidationError("SHIPPO: " + msg)
	}
	return apierror.NewInternalError(fmt.Errorf("shippo status %d: %s", resp.StatusCode, msg), "Shippo API error.")
}

func (c *clientImpl) FindOrRegisterCarrierAccount(ctx context.Context, carrier string) (*domain.ShippoCarrierAccount, *apierror.APIError) {
	ctx, span := shippoTracer.Start(ctx, "shippo.find_or_register_carrier_account")
	defer span.End()

	resp, apiErr := c.doRequest(ctx, http.MethodGet, "/carrier_accounts/?results=100", nil)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, tracing.Trace(span, c.parseErrorResponse(resp))
	}

	var listResp CarrierAccountListResponse
	if err := json.NewDecoder(resp.Body).Decode(&listResp); err != nil {
		return nil, tracing.Trace(span, apierror.NewInternalError(err, "Failed to parse Shippo carrier accounts response."))
	}

	// Find a non-Shippo-default BYOA account for this carrier
	var match *CarrierAccount
	for i := range listResp.Results {
		a := &listResp.Results[i]
		if a.Carrier == carrier && !a.IsShippoAccount {
			if a.Active {
				match = a
				break
			}
			if match == nil {
				match = a
			}
		}
	}

	if match == nil {
		return nil, tracing.Trace(span, apierror.NewValidationError(
			fmt.Sprintf("No %s BYOA account found in Shippo. Please connect your own %s account in the Shippo dashboard first.",
				strings.ToUpper(carrier), strings.ToUpper(carrier))))
	}

	return mapCarrierAccount(match), nil
}

func (c *clientImpl) ConnectCarrierAccount(ctx context.Context, carrier, accountID string, params map[string]string) (*domain.ShippoCarrierAccount, *apierror.APIError) {
	ctx, span := shippoTracer.Start(ctx, "shippo.connect_carrier_account")
	defer span.End()

	createReq := CreateCarrierAccountRequest{
		Carrier:    carrier,
		AccountID:  accountID,
		Active:     true,
		Parameters: params,
	}

	resp, apiErr := c.doRequest(ctx, http.MethodPost, "/carrier_accounts/", createReq)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusBadRequest {
		// Check if the account already exists
		b, _ := io.ReadAll(resp.Body)
		var errBody APIError
		_ = json.Unmarshal(b, &errBody)

		alreadyExists := false
		for _, e := range errBody.NonFieldErrors {
			if strings.Contains(e, "already exists") {
				alreadyExists = true
				break
			}
		}

		if alreadyExists {
			// Find the existing account and reactivate
			existing, apiErr := c.findExistingCarrierAccount(ctx, carrier)
			if apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			if existing != nil {
				updated, apiErr := c.updateCarrierAccount(ctx, existing.ObjectID, carrier, accountID, params)
				if apiErr != nil {
					// Fall back to returning existing as-is
					return existing, nil
				}
				return updated, nil
			}
		}

		return nil, tracing.Trace(span, apierror.NewValidationError("SHIPPO: "+parseAPIErrorMessage(b)))
	}

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return nil, tracing.Trace(span, c.parseErrorResponse(resp))
	}

	var account CarrierAccount
	if err := json.NewDecoder(resp.Body).Decode(&account); err != nil {
		return nil, tracing.Trace(span, apierror.NewInternalError(err, "Failed to parse Shippo carrier account response."))
	}

	if account.IsShippoAccount {
		return nil, tracing.Trace(span, apierror.NewValidationError(
			fmt.Sprintf("The %s account found is a Shippo default account. Please connect your own %s account in the Shippo dashboard.",
				strings.ToUpper(carrier), strings.ToUpper(carrier))))
	}

	return mapCarrierAccount(&account), nil
}

func (c *clientImpl) GetCarrierAccount(ctx context.Context, objectID string) (*domain.ShippoCarrierAccount, *apierror.APIError) {
	ctx, span := shippoTracer.Start(ctx, "shippo.get_carrier_account")
	defer span.End()

	resp, apiErr := c.doRequest(ctx, http.MethodGet, "/carrier_accounts/"+objectID, nil)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, tracing.Trace(span, c.parseErrorResponse(resp))
	}

	var account CarrierAccount
	if err := json.NewDecoder(resp.Body).Decode(&account); err != nil {
		return nil, tracing.Trace(span, apierror.NewInternalError(err, "Failed to parse Shippo carrier account response."))
	}

	return mapCarrierAccount(&account), nil
}

func (c *clientImpl) DeactivateCarrierAccount(ctx context.Context, objectID string) *apierror.APIError {
	ctx, span := shippoTracer.Start(ctx, "shippo.deactivate_carrier_account")
	defer span.End()

	// First get the account to know carrier and accountId
	account, apiErr := c.GetCarrierAccount(ctx, objectID)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	updateReq := UpdateCarrierAccountRequest{
		Carrier:   account.Carrier,
		AccountID: account.AccountID,
		Active:    false,
	}

	resp, apiErr := c.doRequest(ctx, http.MethodPut, "/carrier_accounts/"+objectID, updateReq)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return tracing.Trace(span, c.parseErrorResponse(resp))
	}

	return nil
}

// Lists the service levels a carrier account offers, which Shippo exposes only as the rates on a
// shipment — so this rates a throwaway one and keeps the service levels off it.
func (c *clientImpl) GetCarrierServiceLevels(ctx context.Context, objectID string) ([]domain.ShippoServiceLevel, *apierror.APIError) {
	ctx, span := shippoTracer.Start(ctx, "shippo.get_carrier_service_levels")
	defer span.End()

	// Fabricated origin, destination and parcel: nothing is bought, only the rate list is read.
	shipmentReq := CreateShipmentRequest{
		AddressFrom: ShipmentAddress{
			Name: "Test", Street1: "215 Clayton St", City: "San Francisco", State: "CA", Zip: "94117", Country: "US",
		},
		AddressTo: ShipmentAddress{
			Name: "Test", Street1: "185 Berry St", City: "San Francisco", State: "CA", Zip: "94107", Country: "US",
		},
		Parcels: []Parcel{{
			Weight: "5", Length: "10", Width: "10", Height: "10", MassUnit: "lb", DistanceUnit: "in",
		}},
		CarrierAccounts: []string{objectID},
	}

	resp, apiErr := c.doRequest(ctx, http.MethodPost, "/shipments/", shipmentReq)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return nil, tracing.Trace(span, c.parseErrorResponse(resp))
	}

	var shipment ShipmentResponse
	if err := json.NewDecoder(resp.Body).Decode(&shipment); err != nil {
		return nil, tracing.Trace(span, apierror.NewInternalError(err, "Failed to parse Shippo shipment response."))
	}

	seen := make(map[string]bool)
	var levels []domain.ShippoServiceLevel
	for _, r := range shipment.Rates {
		if r.ServiceLevel == nil || r.ServiceLevel.Token == "" || r.ServiceLevel.Name == "" {
			continue
		}
		if seen[r.ServiceLevel.Token] {
			continue
		}
		seen[r.ServiceLevel.Token] = true
		levels = append(levels, domain.ShippoServiceLevel{
			Name:  r.ServiceLevel.Name,
			Token: r.ServiceLevel.Token,
		})
	}

	return levels, nil
}

func (c *clientImpl) InitiateOAuth(ctx context.Context, objectID, redirectURI string, state *string) (string, *apierror.APIError) {
	ctx, span := shippoTracer.Start(ctx, "shippo.initiate_oauth")
	defer span.End()

	stateStr := ""
	if state != nil {
		stateStr = *state
	}

	path := fmt.Sprintf("/carrier_accounts/%s/signin/initiate/?redirect_uri=%s", objectID, url.QueryEscape(redirectURI))
	if stateStr != "" {
		path += "&state=" + url.QueryEscape(stateStr)
	}

	// Don't follow redirects — we want the Location header
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, shippoBaseURL+path, nil)
	if err != nil {
		return "", tracing.Trace(span, apierror.NewInternalError(err, "Failed to create Shippo OAuth request."))
	}
	req.Header.Set("Authorization", c.authHeader())

	resp, err := client.Do(req) // #nosec G704 -- URL from server-configured Shippo API base
	if err != nil {
		return "", tracing.Trace(span, apierror.NewInternalError(err, "Shippo OAuth request failed."))
	}
	defer resp.Body.Close()

	location := resp.Header.Get("Location")
	if location == "" {
		return "", tracing.Trace(span, apierror.NewValidationError("Shippo OAuth did not return a redirect URL."))
	}

	return location, nil
}

func (c *clientImpl) findExistingCarrierAccount(ctx context.Context, carrier string) (*domain.ShippoCarrierAccount, *apierror.APIError) {
	resp, apiErr := c.doRequest(ctx, http.MethodGet, "/carrier_accounts/?results=100", nil)
	if apiErr != nil {
		return nil, apiErr
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, nil
	}

	var listResp CarrierAccountListResponse
	if err := json.NewDecoder(resp.Body).Decode(&listResp); err != nil {
		return nil, nil
	}

	var match *CarrierAccount
	for i := range listResp.Results {
		a := &listResp.Results[i]
		if a.Carrier == carrier && !a.IsShippoAccount {
			if a.Active {
				match = a
				break
			}
			if match == nil {
				match = a
			}
		}
	}

	if match == nil {
		return nil, nil
	}
	return mapCarrierAccount(match), nil
}

func (c *clientImpl) updateCarrierAccount(ctx context.Context, objectID, carrier, accountID string, params map[string]string) (*domain.ShippoCarrierAccount, *apierror.APIError) {
	updateReq := UpdateCarrierAccountRequest{
		Carrier:    carrier,
		AccountID:  accountID,
		Active:     true,
		Parameters: params,
	}

	resp, apiErr := c.doRequest(ctx, http.MethodPut, "/carrier_accounts/"+objectID, updateReq)
	if apiErr != nil {
		return nil, apiErr
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseErrorResponse(resp)
	}

	var account CarrierAccount
	if err := json.NewDecoder(resp.Body).Decode(&account); err != nil {
		return nil, apierror.NewInternalError(err, "Failed to parse Shippo carrier account response.")
	}

	if account.IsShippoAccount {
		return nil, apierror.NewValidationError(
			fmt.Sprintf("The %s account found is a Shippo default account. Please connect your own %s account in the Shippo dashboard.",
				strings.ToUpper(carrier), strings.ToUpper(carrier)))
	}

	return mapCarrierAccount(&account), nil
}

func mapCarrierAccount(a *CarrierAccount) *domain.ShippoCarrierAccount {
	return &domain.ShippoCarrierAccount{
		ObjectID:        a.ObjectID,
		Carrier:         a.Carrier,
		AccountID:       a.AccountID,
		Active:          a.Active,
		IsShippoAccount: a.IsShippoAccount,
	}
}

func parseAPIErrorMessage(b []byte) string {
	var apiErr APIError
	if err := json.Unmarshal(b, &apiErr); err != nil {
		if len(b) > 0 {
			return string(b)
		}
		return "Bad request"
	}
	if apiErr.Detail != "" {
		return apiErr.Detail
	}
	if apiErr.Message != "" {
		return apiErr.Message
	}
	if apiErr.Error != "" {
		return apiErr.Error
	}
	if len(apiErr.NonFieldErrors) > 0 {
		return strings.Join(apiErr.NonFieldErrors, "; ")
	}
	return string(b)
}

// shippingRateMarkup is the multiplier applied to all shipping rates (10% markup).
const shippingRateMarkup = 1.1

func applyShippingMarkup(rate float64) float64 {
	if rate == 0 {
		return 0
	}
	return rate * shippingRateMarkup
}

func (c *clientImpl) createShipmentForRates(ctx context.Context, carrierAccountObjectID string, from, to domain.ShippingAddress, parcels []domain.Parcel, billing *domain.ShippingBilling) (*ShipmentResponse, *apierror.APIError) {
	shipmentParcels := make([]Parcel, len(parcels))
	for i, p := range parcels {
		shipmentParcels[i] = Parcel{
			Weight:       normalizeShippoDecimal(p.Weight),
			Length:       normalizeShippoDecimal(p.Length),
			Width:        normalizeShippoDecimal(p.Width),
			Height:       normalizeShippoDecimal(p.Height),
			MassUnit:     "lb",
			DistanceUnit: "in",
		}
	}

	shipmentReq := CreateShipmentRequest{
		AddressFrom: ShipmentAddress{
			Name:    from.Name,
			Street1: from.Street1,
			City:    from.City,
			State:   from.State,
			Zip:     from.Zip,
			Country: from.Country,
		},
		AddressTo: ShipmentAddress{
			Name:    to.Name,
			Street1: to.Street1,
			City:    to.City,
			State:   to.State,
			Zip:     to.Zip,
			Country: to.Country,
		},
		Parcels:         shipmentParcels,
		CarrierAccounts: []string{carrierAccountObjectID},
	}

	// Third-party freight billing: forward the third party's account + address to the
	// carrier via the Shippo shipment `extra.billing` object (matches Dashboard).
	if billing != nil && billing.Type != "" {
		shipmentReq.Extra = &ShipmentExtra{
			Billing: &ShipmentBilling{
				Type:    billing.Type,
				Account: billing.Account,
				Country: billing.Country,
				Zip:     billing.Zip,
			},
		}
	}

	resp, apiErr := c.doRequest(ctx, http.MethodPost, "/shipments/", shipmentReq)
	if apiErr != nil {
		return nil, apiErr
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return nil, c.parseErrorResponse(resp)
	}

	var shipment ShipmentResponse
	if err := json.NewDecoder(resp.Body).Decode(&shipment); err != nil {
		return nil, apierror.NewInternalError(err, "Failed to parse Shippo shipment response.")
	}

	return &shipment, nil
}

// resolvePublishedRateCarrierAccountID maps a BYOA (bring-your-own-account) carrier account object ID to the carrier's Shippo default (published/retail) account object ID, mirroring Dashboard's resolvePublishedRateCarrierAccountId. BYOA accounts return the account's negotiated rates; customer-facing estimates should quote published rates instead. Best-effort: if the BYOA account cannot be read or no Shippo default account exists for that carrier, the original BYOA object ID is returned so a quote is still produced.
func (c *clientImpl) resolvePublishedRateCarrierAccountID(ctx context.Context, byoaObjectID string) string {
	accounts := c.listCarrierAccounts(ctx)

	carrier := carrierOf(accounts, byoaObjectID)
	if carrier == "" {
		// The account is past the page fetched above, so read it directly.
		byoa, apiErr := c.GetCarrierAccount(ctx, byoaObjectID)
		if apiErr != nil {
			return byoaObjectID
		}
		carrier = byoa.Carrier
	}

	if match := findDefaultCarrierAccount(accounts, carrier); match != nil && match.ObjectID != "" {
		return match.ObjectID
	}
	return byoaObjectID
}

// listCarrierAccounts returns the carrier accounts on this token, fetched once per client. Returns nil when the fetch fails; callers fall back to the account they were given.
func (c *clientImpl) listCarrierAccounts(ctx context.Context) []CarrierAccount {
	c.carrierAccountsOnce.Do(func() {
		resp, apiErr := c.doRequest(ctx, http.MethodGet, "/carrier_accounts/?results=100", nil)
		if apiErr != nil {
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return
		}

		var listResp CarrierAccountListResponse
		if err := json.NewDecoder(resp.Body).Decode(&listResp); err != nil {
			return
		}
		c.carrierAccounts = listResp.Results
	})
	return c.carrierAccounts
}

// carrierOf returns the carrier type of the named account, or "" when it is not in the list.
func carrierOf(accounts []CarrierAccount, objectID string) string {
	for i := range accounts {
		if accounts[i].ObjectID == objectID {
			return accounts[i].Carrier
		}
	}
	return ""
}

// findDefaultCarrierAccount returns Shippo's built-in account for a carrier type (published/retail rates), preferring an active one. Returns nil when the carrier has no default account.
func findDefaultCarrierAccount(accounts []CarrierAccount, carrier string) *CarrierAccount {
	var match *CarrierAccount
	for i := range accounts {
		a := &accounts[i]
		if a.Carrier == carrier && a.IsShippoAccount {
			if a.Active {
				return a
			}
			if match == nil {
				match = a
			}
		}
	}
	return match
}

func (c *clientImpl) FetchShippingRate(ctx context.Context, params domain.FetchShippingRateParams) (float64, *apierror.APIError) {
	ctx, span := shippoTracer.Start(ctx, "shippo.fetch_shipping_rate")
	defer span.End()

	// Customer-facing estimates quote the carrier's Shippo default (published/retail) account rather than the BYOA account's negotiated rates.
	carrierAccountObjectID := c.resolvePublishedRateCarrierAccountID(ctx, params.CarrierAccountObjectID)
	shipment, apiErr := c.createShipmentForRates(ctx, carrierAccountObjectID, params.FromAddress, params.ToAddress, params.Parcels, params.Billing)
	if apiErr != nil {
		return 0, tracing.Trace(span, apiErr)
	}

	if len(shipment.Rates) == 0 {
		return 0, nil
	}

	// If a specific service level token is requested, find that rate
	if params.ServiceLevelToken != "" {
		for _, r := range shipment.Rates {
			if r.ServiceLevel != nil && r.ServiceLevel.Token == params.ServiceLevelToken {
				amount, err := parseFloat(r.Amount)
				if err != nil {
					return 0, tracing.Trace(span, apierror.NewInternalError(err, "Failed to parse Shippo rate amount."))
				}
				return applyShippingMarkup(amount), nil
			}
		}
		return 0, nil
	}

	// Prefer BESTVALUE attribute
	for _, r := range shipment.Rates {
		for _, attr := range r.Attributes {
			if attr == "BESTVALUE" {
				amount, err := parseFloat(r.Amount)
				if err != nil {
					return 0, tracing.Trace(span, apierror.NewInternalError(err, "Failed to parse Shippo rate amount."))
				}
				return applyShippingMarkup(amount), nil
			}
		}
	}

	// Fallback to cheapest rate
	var cheapest float64 = -1
	for _, r := range shipment.Rates {
		amount, err := parseFloat(r.Amount)
		if err != nil {
			continue
		}
		if cheapest < 0 || amount < cheapest {
			cheapest = amount
		}
	}

	if cheapest < 0 {
		return 0, nil
	}
	return applyShippingMarkup(cheapest), nil
}

func (c *clientImpl) FetchAllShippingRates(ctx context.Context, params domain.FetchAllShippingRatesParams) ([]domain.ShippoRateOption, *apierror.APIError) {
	ctx, span := shippoTracer.Start(ctx, "shippo.fetch_all_shipping_rates")
	defer span.End()

	// Customer-facing estimates quote the carrier's Shippo default (published/retail) account rather than the BYOA account's negotiated rates.
	carrierAccountObjectID := c.resolvePublishedRateCarrierAccountID(ctx, params.CarrierAccountObjectID)
	shipment, apiErr := c.createShipmentForRates(ctx, carrierAccountObjectID, params.FromAddress, params.ToAddress, params.Parcels, nil)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	var options []domain.ShippoRateOption
	for _, r := range shipment.Rates {
		if r.ServiceLevel == nil || r.ServiceLevel.Token == "" || r.ServiceLevel.Name == "" {
			continue
		}
		amount, err := parseFloat(r.Amount)
		if err != nil || amount == 0 {
			continue
		}
		options = append(options, domain.ShippoRateOption{
			ServiceLevelName:  r.ServiceLevel.Name,
			ServiceLevelToken: r.ServiceLevel.Token,
			Amount:            applyShippingMarkup(amount),
			EstimatedDays:     r.EstimatedDays,
		})
	}

	return options, nil
}

func parseFloat(s string) (float64, error) {
	var f float64
	_, err := fmt.Sscanf(s, "%f", &f)
	return f, err
}

// shippoMaxDecimalPlaces bounds the fractional digits sent to Shippo for parcel
// weight and dimensions.
const shippoMaxDecimalPlaces = 4

// normalizeShippoDecimal formats a numeric string with at most shippoMaxDecimalPlaces
// fractional digits, trimming trailing zeros. Shippo's parcel fields are
// DecimalField(max_digits=10); raw float formatting (e.g. strconv.FormatFloat(v,
// 'f', -1, 64) producing "1.0499999999999998" from accumulated float arithmetic)
// is otherwise rejected with "Ensure that there are no more than 10 digits in
// total." Unparseable values are returned unchanged for Shippo to validate.
func normalizeShippoDecimal(s string) string {
	v, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil {
		return s
	}
	out := strconv.FormatFloat(v, 'f', shippoMaxDecimalPlaces, 64)
	if strings.Contains(out, ".") {
		out = strings.TrimRight(out, "0")
		out = strings.TrimRight(out, ".")
	}
	return out
}

// Selects the format Shippo renders purchased labels in, matching what the label viewer expects.
const labelFileType = "PNG"

// Buys carrier labels for a shipment's cases in one instant-label transaction, then reads the
// per-parcel transactions off the purchased rate for each case's tracking number and label URL.
func (c *clientImpl) CreateTransactionInstantLabel(ctx context.Context, params domain.CreateLabelParams) (*domain.LabelResult, *apierror.APIError) {
	ctx, span := shippoTracer.Start(ctx, "shippo.create_transaction_instant_label")
	defer span.End()

	if len(params.Parcels) == 0 {
		return nil, tracing.Trace(span, apierror.NewValidationError("SHIPPO: A label purchase requires at least one parcel."))
	}

	parcels := make([]Parcel, len(params.Parcels))
	for i, parcel := range params.Parcels {
		parcels[i] = Parcel{
			Weight:       normalizeShippoDecimal(parcel.Weight),
			Length:       normalizeShippoDecimal(parcel.Length),
			Width:        normalizeShippoDecimal(parcel.Width),
			Height:       normalizeShippoDecimal(parcel.Height),
			MassUnit:     "lb",
			DistanceUnit: "in",
		}
	}

	req := CreateTransactionRequest{
		CarrierAccount:    params.CarrierAccountObjectID,
		ServiceLevelToken: params.ServiceLevelToken,
		LabelFileType:     labelFileType,
		Shipment: LabelShipment{
			AddressFrom: toLabelAddress(params.FromAddress),
			AddressTo:   toLabelAddress(params.ToAddress),
			Parcels:     parcels,
		},
	}
	if params.Billing != nil && params.Billing.Type != "" {
		req.Shipment.Extra = &ShipmentExtra{
			Billing: &ShipmentBilling{
				Type:    params.Billing.Type,
				Account: params.Billing.Account,
				Country: params.Billing.Country,
				Zip:     params.Billing.Zip,
			},
		}
	}

	resp, apiErr := c.doRequest(ctx, http.MethodPost, "/transactions", req)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return nil, tracing.Trace(span, c.parseErrorResponse(resp))
	}

	var transaction TransactionResponse
	if err := json.NewDecoder(resp.Body).Decode(&transaction); err != nil {
		return nil, tracing.Trace(span, apierror.NewInternalError(err, "Failed to parse Shippo transaction response."))
	}

	rateObjectID, rateAmount := parseTransactionRate(transaction.Rate)
	if transaction.Status != "SUCCESS" || rateObjectID == "" {
		return nil, tracing.Trace(span, apierror.NewValidationError(
			fmt.Sprintf("SHIPPO: Transaction status is %s - %s", transaction.Status, transactionMessages(transaction.Messages))))
	}

	// The instant-label call returns one transaction; the per-parcel labels hang off the rate it bought.
	parcelTransactions, apiErr := c.listTransactionsByRate(ctx, rateObjectID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if len(parcelTransactions) < len(params.Parcels) {
		return nil, tracing.Trace(span, apierror.NewValidationError("SHIPPO: Parcel transactions not found."))
	}

	masterTracking := transaction.TrackingNumber
	packages := make([]domain.LabelPackage, len(params.Parcels))
	for i := range params.Parcels {
		// Shippo lists a rate's transactions newest-first, so the parcels come back reversed.
		pt := parcelTransactions[len(parcelTransactions)-1-i]
		if pt.ObjectID == "" || pt.TrackingNumber == "" || pt.LabelURL == "" {
			return nil, tracing.Trace(span, apierror.NewValidationError("SHIPPO: Parcel transaction not found."))
		}
		if masterTracking == "" {
			masterTracking = pt.TrackingNumber
		}
		packages[i] = domain.LabelPackage{
			TrackingNumber:      pt.TrackingNumber,
			LabelURL:            pt.LabelURL,
			ShippoTransactionID: pt.ObjectID,
		}
	}

	// The negotiated rate is what the account is actually billed, so it is passed through unmarked-up.
	amount, err := parseFloat(rateAmount)
	if err != nil {
		amount = 0
	}

	return &domain.LabelResult{
		MasterTrackingNumber: masterTracking,
		NegotiatedRate:       amount,
		Packages:             packages,
	}, nil
}

// Lists the transactions Shippo created for one purchased rate — one per parcel.
func (c *clientImpl) listTransactionsByRate(ctx context.Context, rateObjectID string) ([]TransactionResponse, *apierror.APIError) {
	resp, apiErr := c.doRequest(ctx, http.MethodGet, "/transactions?rate="+url.QueryEscape(rateObjectID), nil)
	if apiErr != nil {
		return nil, apiErr
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseErrorResponse(resp)
	}

	var listResp TransactionListResponse
	if err := json.NewDecoder(resp.Body).Decode(&listResp); err != nil {
		return nil, apierror.NewInternalError(err, "Failed to parse Shippo transaction list response.")
	}
	return listResp.Results, nil
}

// Refunds a purchased Shippo label transaction. An already-refunded transaction is treated as
// success so a retried void is not blocked by work a previous attempt completed.
func (c *clientImpl) RefundTransaction(ctx context.Context, transactionID string) *apierror.APIError {
	ctx, span := shippoTracer.Start(ctx, "shippo.refund_transaction")
	defer span.End()

	resp, apiErr := c.doRequest(ctx, http.MethodPost, "/refunds", CreateRefundRequest{Transaction: transactionID})
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusBadRequest {
		b, _ := io.ReadAll(resp.Body)
		if isAlreadyRefunded(parseAPIErrorMessage(b)) {
			return nil
		}
		return tracing.Trace(span, apierror.NewValidationError("SHIPPO: "+parseAPIErrorMessage(b)))
	}
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return tracing.Trace(span, c.parseErrorResponse(resp))
	}

	var refund RefundResponse
	if err := json.NewDecoder(resp.Body).Decode(&refund); err != nil {
		return tracing.Trace(span, apierror.NewInternalError(err, "Failed to parse Shippo refund response."))
	}
	if refund.Status == "ERROR" {
		return tracing.Trace(span, apierror.NewValidationError("SHIPPO: Refund failed for transaction "+transactionID+"."))
	}
	return nil
}

// Reports whether a Shippo 400 means the label was already refunded or voided rather than a real failure.
func isAlreadyRefunded(message string) bool {
	m := strings.ToLower(message)
	return strings.Contains(m, "already") || strings.Contains(m, "refunded") || strings.Contains(m, "voided")
}

// Reads a transaction's rate, which Shippo returns either as the rate's object id or as the rate object.
func parseTransactionRate(raw json.RawMessage) (objectID, amount string) {
	if len(raw) == 0 {
		return "", ""
	}
	var rate TransactionRate
	if err := json.Unmarshal(raw, &rate); err == nil && rate.ObjectID != "" {
		return rate.ObjectID, rate.Amount
	}
	var id string
	if err := json.Unmarshal(raw, &id); err == nil {
		return id, ""
	}
	return "", ""
}

// Joins a transaction's messages into one explanation for the error surfaced to the user.
func transactionMessages(messages []TransactionMessage) string {
	if len(messages) == 0 {
		return "Unknown error"
	}
	texts := make([]string, 0, len(messages))
	for _, m := range messages {
		if m.Text != "" {
			texts = append(texts, m.Text)
		}
	}
	if len(texts) == 0 {
		return "Unknown error"
	}
	return strings.Join(texts, ", ")
}

// Maps a domain address onto the fuller address a printed label needs.
func toLabelAddress(a domain.ShippingAddress) LabelAddress {
	out := LabelAddress{
		Name:    a.Name,
		Street1: a.Street1,
		City:    a.City,
		State:   a.State,
		Zip:     a.Zip,
		Country: a.Country,
	}
	if a.Company != nil {
		out.Company = *a.Company
	}
	if a.Street2 != nil {
		out.Street2 = *a.Street2
	}
	if a.Phone != nil {
		out.Phone = *a.Phone
	}
	if a.Email != nil {
		out.Email = *a.Email
	}
	return out
}
