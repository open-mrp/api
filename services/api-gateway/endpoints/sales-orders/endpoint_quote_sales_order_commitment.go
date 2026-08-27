package salesorderep

import (
	"context"
	"net/http"
	"time"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/field"
)

// Request to preview the ship-by date a set of commitment inputs would produce.
type QuoteSalesOrderCommitmentRequest struct {
	// An existing order to preview against. Its customer, ship-to address, carrier, and service level are used, and the commitment fields below replace whatever it currently carries.
	//
	// Omit it to preview an order that has not been created yet, supplying the parts directly.
	SalesOrderID field.Optional[string] `json:"sales_order_id,omitzero" validate:"omitempty"`
	// The buying account, used to resolve its lead time and receiving days.
	BuyerAccountID field.Optional[string] `json:"buyer_account_id,omitzero" validate:"omitempty"`
	// The ship-to address, which decides the destination timezone and the lane transit is quoted on.
	ShipToAddressID field.Optional[string] `json:"ship_to_address_id,omitzero" validate:"omitempty"`
	// Carrier for the shipment.
	CarrierID field.Optional[string] `json:"carrier_id,omitzero" validate:"omitempty"`
	// Service level for the shipment, which the lane's transit estimate is keyed on.
	ServiceLevelID field.Optional[string] `json:"service_level_id,omitzero" validate:"omitempty"`
	// When the order would be issued. Defaults to now, since a lead time is measured from issue and an order built today but issued next week commits to next week's date.
	IssuedAt field.Optional[time.Time] `json:"issued_at,omitzero"`
	// Date delivery would be promised to the customer.
	PromisedAt field.Optional[time.Time] `json:"promised_at,omitzero"`
	// Days between issue and the order being due to ship, in place of the customer's standing lead time.
	LeadTimeOverrideDays field.Optional[int32] `json:"lead_time_override_days,omitzero" validate:"omitempty,gte=0,lte=3650"`
	// The exact date the order would be due to ship.
	ShipByOverrideDate field.Optional[time.Time] `json:"ship_by_override_date,omitzero"`
}

// The ship-by date a set of commitment inputs would produce, and how it was reached.
type QuoteSalesOrderCommitmentResponse struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=sales_order_commitment_quote"`
	// The commitment the inputs would produce — the same object an order carries once issued, so a preview and the stamped result cannot drift.
	//
	// Its `ship_by_date` is null when no rule resolves one, and `transit_days` is null when the lane has never been quoted and the service level carries no default, or when no service level was supplied to quote one on.
	Commitment *apiresource.Commitment `json:"commitment"`
	// The derivation in order, one entry per rule that moved the date.
	Steps []apiresource.CommitmentQuoteStep `json:"steps"`
}

// The sample previews the case the feature exists for: a Saturday delivery promised to a customer who only receives on weekdays.
var sampleQuoteSalesOrderCommitmentRequest = &QuoteSalesOrderCommitmentRequest{
	BuyerAccountID:  field.Some(apiresource.SampleAccountID),
	ShipToAddressID: field.Some(apiresource.SampleAddressID),
	ServiceLevelID:  field.Some(apiresource.SampleServiceLevelID),
	PromisedAt:      field.Some(time.Date(2026, time.August, 22, 0, 0, 0, 0, time.UTC)),
}

func (*QuoteSalesOrderCommitmentRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleQuoteSalesOrderCommitmentRequest)
}

var sampleQuoteSalesOrderCommitmentResponse = &QuoteSalesOrderCommitmentResponse{
	Object:     constants.ObjectTypeSalesOrderCommitmentQuote,
	Commitment: apiresource.SampleQuotedCommitment,
	Steps:      apiresource.SampleCommitmentQuoteSteps(),
}

func (*QuoteSalesOrderCommitmentResponse) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleQuoteSalesOrderCommitmentResponse)
}

// Previews the ship-by date a set of commitment inputs would produce, without creating or changing anything.
//
// Runs the same resolution an order runs when it is issued: a promised delivery date has the customer's receiving days, the carrier's transit, and the plant's shipping days worked back through it, while a lead time or a pinned ship date is snapped onto the next earlier day the plant ships. The returned steps are that derivation in order, so a caller can show why a date is what it is rather than restating the rules.
//
// At most one of `promised_at`, `lead_time_override_days`, and `ship_by_override_date` may be set; they are alternative answers to the same question.
//
// Advisory rather than binding. Carrier transit comes from a lane cache warmed in the background, so a lane nobody has shipped yet quotes against the service level's default or against no transit at all, and the date stamped at issue may differ once the lane has been rated.
type QuoteSalesOrderCommitmentEndpoint struct{}

func (e *QuoteSalesOrderCommitmentEndpoint) Materialize() *apiendpoint.APIEndpoint[*QuoteSalesOrderCommitmentRequest, *QuoteSalesOrderCommitmentResponse] {
	return (&apiendpoint.APIEndpoint[*QuoteSalesOrderCommitmentRequest, *QuoteSalesOrderCommitmentResponse]{
		Title:             "Quote Sales Order Commitment",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/sales/sales-orders/actions/quote-commitment",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		ReadOnly:          true,
		Preview:           true,
		Extras:            apiendpoint.APIEndpointExtras{HideFromRequestLog: true},
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainSalesOrders, Action: types.ActionRead},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *QuoteSalesOrderCommitmentRequest) (*QuoteSalesOrderCommitmentResponse, *apierror.APIError) {
			return svc.(SalesOrderSvc).QuoteSalesOrderCommitment
		},
	})
}
