// Package lifecycle builds the endpoints that move a record through its workflow: one `POST …/actions/{verb}` per
// transition, named for the status it produces (`issue`, `release`, `cancel`). Status itself is read-only.
//
// An action endpoint answers 200 with the updated resource, so the caller sees the new status without another read.
// Like every POST it is safe to retry with an Idempotency-Key. When the record's current status doesn't allow the
// action, the service returns 409 `invalid_state_transition` (core-service domain.RequireTransition).
//
// The endpoint's own type still carries its docstring, which the reference publishes. Write it as:
//
//	// Releases a job, moving it from `planned` to `released`.
//	//
//	// Only a `planned` job can be released; any other status is a 409 `invalid_state_transition`.
//	type ReleaseJobEndpoint struct{}
//
//	func (*ReleaseJobEndpoint) Materialize() *apiendpoint.APIEndpoint[*lifecycle.Request, *apiresource.Job] {
//		return lifecycle.Action("Release Job", "/v1/operations/jobs/{id}/actions/release", constants.ObjectTypeJob, policy,
//			func(svc any) apiendpoint.ServiceHandler[*lifecycle.Request, *apiresource.Job] { return svc.(JobSvc).ReleaseJob })
//	}
package lifecycle

import (
	"fmt"
	"net/http"
	"regexp"

	apiendpoint "github.com/open-mrp/apikit/endpoint"
	"github.com/open-mrp/apikit/object"
)

// Request is the request of an action that takes nothing but the record's ID. Embed it in an action's own request when the action takes a body.
type Request struct {
	// ID of the record.
	ID string `path:"id" validate:"required"`
}

var actionRoute = regexp.MustCompile(`^/v1/[a-z-]+/.+/\{id\}/actions/[a-z]+(-[a-z]+)*$`)

// Action is a lifecycle action endpoint: POST to route, 200 with the updated resource, public, and offered to agents. route must be a record's `…/{id}/actions/{verb}`; anything else panics at startup, since it would ship a route that breaks the convention.
func Action[TReq, TResp any](title, route string, objectType object.Type, auth any, handler func(svc any) apiendpoint.ServiceHandler[TReq, TResp]) *apiendpoint.APIEndpoint[TReq, TResp] {
	if !actionRoute.MatchString(route) {
		panic(fmt.Sprintf("lifecycle: %q is not a record action route (/v1/{namespace}/{resource}/{id}/actions/{verb})", route))
	}
	if objectType == "" {
		panic(fmt.Sprintf("lifecycle: %s needs the object type it returns", route))
	}
	return &apiendpoint.APIEndpoint[TReq, TResp]{
		Title:             title,
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             route,
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		Auth:              auth,
		ObjectType:        objectType,
		ServiceHandler:    handler,
	}
}
