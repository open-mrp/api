package requestlogep

import (
	"context"
	"net/http"
	"time"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to list request logs.
type ListRequestLogsRequest struct {
	apiresource.PaginationRequest
	// Restricts results to request logs on or after this timestamp.
	StartDate *time.Time `query:"start_date"`
	// Restricts results to request logs on or before this timestamp.
	EndDate *time.Time `query:"end_date"`
	// Filter by the HTTP method.
	Methods []constants.HTTPMethod `query:"methods"`
	// Filter by the HTTP status code.
	StatusCodes []int32 `query:"status_codes"`
	// Filter by the HTTP status class, expressed as the leading digit: `1`–`5` for 1xx–5xx.
	//
	// Combined with `status_codes` using OR — e.g. `status_codes=401` and `status_code_classes=5` matches 401 responses and any 5xx response.
	StatusCodeClasses []int32 `query:"status_code_classes"`
	// Filter by API error code.
	ErrorCodes []apierror.ErrorCode `query:"error_codes"`
	// Exclude request logs whose API error code is in this set.
	//
	// Applied as a negative filter after all other filters. Successful requests (which have no error code) are always kept. The dashboard uses this to hide routine `expired_token` 401s — the noise from short-lived access tokens expiring and clients silently refreshing — while still surfacing genuine auth failures like `invalid_credentials`.
	ExcludeErrorCodes []apierror.ErrorCode `query:"exclude_error_codes"`
	// Filter by the _acting_ account: the account the actor belongs to (the log's `account.id`).
	//
	// Results are always scoped to logs where your account is either the acting account or the target account; this narrows that set to specific acting accounts. For example, pass a customer's account ID to see only requests that customer's actors made against your account.
	ActorAccountIDs []string `query:"actor_account_ids"`
	// Filter by the _target_ account: the account the request acted upon (the log's target account).
	//
	// Results are always scoped to logs where your account is either the acting account or the target account; this narrows that set to specific target accounts. For example, pass a supplier's account ID to see only requests your account made against that supplier.
	TargetAccountIDs []string `query:"target_account_ids"`
	// Filter by the actor identifier.
	//
	// Matches the log's `actor.id`: a user ID for `user` actors or an API key ID for `api_key` actors.
	ActorIDs []string `query:"actor_ids"`
	// Filter by the actor type.
	ActorTypes []constants.ActorType `query:"actor_types"`
	// Filter by the _normalized_ route template.
	//
	// For example `/v1/sales/customers/{id}` matches every request to that route regardless of the specific customer ID. Parameter names inside `{}` are ignored when matching, so `{customer_id}` and `{id}` are equivalent.
	NormalizedRoutes []string `query:"normalized_routes"`
	// Filter by the request host.
	//
	// Typically `api.augno.com`.
	Hosts []string `query:"hosts"`
	// Restricts results to requests that took at least this many microseconds.
	MinLatencyUs *int64 `query:"min_latency_us"`
	// Filter by the user-provided idempotency key.
	IdempotencyKey *string `query:"idempotency_key"`
}

// Returns a paginated list of request logs for the current account.
type ListRequestLogsEndpoint struct{}

func (e *ListRequestLogsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListRequestLogsRequest, *apiresource.List[apiresource.RequestLog]] {
	return (&apiendpoint.APIEndpoint[*ListRequestLogsRequest, *apiresource.List[apiresource.RequestLog]]{
		Title:               "List Request Logs",
		Method:              http.MethodGet,
		ContentType:         "application/json",
		Route:               "/v1/core/request-logs",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		AgentTool:           true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainRequestLogs, Action: types.ActionRead}},
		Preview:             true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListRequestLogsRequest) (*apiresource.List[apiresource.RequestLog], *apierror.APIError) {
			return svc.(RequestLogSvc).ListRequestLogs
		},
		ObjectType: constants.ObjectTypeRequestLog,
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeRequestLog,
			Fields:     []string{"account", "actor", "actor.role"},
		}),
		Extras: apiendpoint.APIEndpointExtras{
			HideFromRequestLog: true,
		},
	})
}
