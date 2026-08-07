package productionscheduleep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to quote the earliest date a quantity could ship.
type QuotePromiseDateRequest struct {
	// Item being quoted.
	ItemID string `json:"item_id" validate:"required"`
	// Quantity being quoted, in the item's own unit.
	Quantity float64 `json:"quantity" validate:"required,gt=0"`
}

var sampleQuotePromiseDateRequest = &QuotePromiseDateRequest{
	ItemID:   apiresource.SampleItemID,
	Quantity: 1200,
}

func (*QuotePromiseDateRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleQuotePromiseDateRequest)
}

// Returns the earliest date the published schedule could ship a quantity of an item.
//
// Quoted from the published version — the plan the floor is actually working to — and net of everything already promised to other orders. A date backed by stock somebody else is owed is not a date, so existing commitments are consumed before anything is offered.
//
// The answer allows for finishing after the constraint stage completes, so it is a ship date rather than a production date. When the published horizon cannot supply the quantity at all, `is_promisable` is false and no date is returned: a plan that runs thirteen weeks cannot speak for the fourteenth, and inventing a date beyond it would be the one number a customer actually relies on.
//
// Quoting does not reserve anything. Two quotes taken a minute apart can both come back with the same date, and only issuing an order commits the supply.
type QuotePromiseDateEndpoint struct{}

func (e *QuotePromiseDateEndpoint) Materialize() *apiendpoint.APIEndpoint[*QuotePromiseDateRequest, *apiresource.PromiseDateQuote] {
	return (&apiendpoint.APIEndpoint[*QuotePromiseDateRequest, *apiresource.PromiseDateQuote]{
		Title:             "Quote Promise Date",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/operations/production-schedules/actions/quote-promise-date",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		AgentTool:         true,
		ReadOnly:          true,
		ObjectType:        constants.ObjectTypePromiseDateQuote,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainProductionSchedules, Action: types.ActionRead},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *QuotePromiseDateRequest) (*apiresource.PromiseDateQuote, *apierror.APIError) {
			return svc.(ProductionScheduleSvc).QuotePromiseDate
		},
	})
}
