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

	"github.com/augno/api/services/core-service/internal/domain"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/tracing"
)

const shippoBaseURL = "https://api.goshippo.com"

var shippoTracer = tracing.GetTracer("core-service.shippo_client")

type clientImpl struct {
	apiKey     string
	httpClient *http.Client
}

// ClientFactory creates ShippoClient instances from API keys.
type ClientFactory struct{}

func NewClientFactory() *ClientFactory {
	return &ClientFactory{}
}

func (f *ClientFactory) Build(apiKey string) domain.ShippoClient {
	return &clientImpl{
		apiKey:     apiKey,
		httpClient: &http.Client{},
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

	req, err := http.NewRequestWithContext(ctx, method, shippoBaseURL+path, reqBody)
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

func (c *clientImpl) GetCarrierServiceLevels(ctx context.Context, objectID string) ([]domain.ShippoServiceLevel, *apierror.APIError) {
	ctx, span := shippoTracer.Start(ctx, "shippo.get_carrier_service_levels")
	defer span.End()

	shipmentReq := CreateShipmentRequest{
		AddressFrom: TestAddress{
			Name: "Test", Street1: "215 Clayton St", City: "San Francisco", State: "CA", Zip: "94117", Country: "US",
		},
		AddressTo: TestAddress{
			Name: "Test", Street1: "185 Berry St", City: "San Francisco", State: "CA", Zip: "94107", Country: "US",
		},
		Parcels: []TestParcel{{
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
	shipmentParcels := make([]TestParcel, len(parcels))
	for i, p := range parcels {
		shipmentParcels[i] = TestParcel{
			Weight:       normalizeShippoDecimal(p.Weight),
			Length:       normalizeShippoDecimal(p.Length),
			Width:        normalizeShippoDecimal(p.Width),
			Height:       normalizeShippoDecimal(p.Height),
			MassUnit:     "lb",
			DistanceUnit: "in",
		}
	}

	shipmentReq := CreateShipmentRequest{
		AddressFrom: TestAddress{
			Name:    from.Name,
			Street1: from.Street1,
			City:    from.City,
			State:   from.State,
			Zip:     from.Zip,
			Country: from.Country,
		},
		AddressTo: TestAddress{
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
	byoa, apiErr := c.GetCarrierAccount(ctx, byoaObjectID)
	if apiErr != nil {
		return byoaObjectID
	}
	defaultAcct, apiErr := c.findDefaultCarrierAccount(ctx, byoa.Carrier)
	if apiErr == nil && defaultAcct != nil && defaultAcct.ObjectID != "" {
		return defaultAcct.ObjectID
	}
	return byoaObjectID
}

// findDefaultCarrierAccount returns Shippo's built-in default account for a carrier type (published/retail rates), preferring an active account. Returns nil when no default account exists for the carrier.
func (c *clientImpl) findDefaultCarrierAccount(ctx context.Context, carrier string) (*domain.ShippoCarrierAccount, *apierror.APIError) {
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
		if a.Carrier == carrier && a.IsShippoAccount {
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
