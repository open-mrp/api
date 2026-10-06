package apiendpoint

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/appctx"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

type stubCostRate struct {
	Value string `json:"value"`
}

type stubCostLine struct {
	ID       string        `json:"id"`
	UnitCost *stubCostRate `json:"unit_cost" sensitive:"cost"`
}

type stubCostResponse struct {
	ID       string          `json:"id"`
	UnitCost *stubCostRate   `json:"unit_cost" sensitive:"cost"`
	Lines    []*stubCostLine `json:"lines"`
}

func costIdentity(relation types.IdentityRelationType, roleType constants.RoleType, perms ...string) *types.Identity {
	accountID := "ac_actor"
	granted := map[string]bool{}
	for _, p := range perms {
		granted[p] = true
	}
	role := string(roleType)
	return &types.Identity{
		Type:   types.IdentityActorTypeUser,
		Target: &types.IdentityTarget{AccountID: "ac_actor"},
		Actor:  &types.IdentityActor{RelationType: relation, ID: "us_1", AccountID: &accountID, RoleType: &role, Permissions: granted},
	}
}

func agentCostIdentity(perms ...string) *types.Identity {
	identity := costIdentity(types.IdentityRelationTypeInternal, constants.RoleTypeAgent, perms...)
	identity.Type = types.IdentityActorTypeAgent
	return identity
}

// The gateway clears cost fields on the way out, so whatever a handler or include put there, a caller without costs:read reads null.
func TestExecute_clearsCostFieldsForCallersWithoutCostsRead(t *testing.T) {
	t.Parallel()

	ep := &APIEndpoint[*stubRequest, *stubCostResponse]{
		Method:            http.MethodGet,
		Route:             "/v1/things",
		SuccessStatusCode: http.StatusOK,
		ServiceHandler: func(svc any) func(context.Context, *stubRequest) (*stubCostResponse, *apierror.APIError) {
			return func(context.Context, *stubRequest) (*stubCostResponse, *apierror.APIError) {
				return &stubCostResponse{ID: "th_1", UnitCost: &stubCostRate{Value: "4"}, Lines: []*stubCostLine{{ID: "ln_1", UnitCost: &stubCostRate{Value: "2"}}}}, nil
			}
		},
	}
	bindHandler(ep)

	const shown = `{"id":"th_1","unit_cost":{"value":"4"},"lines":[{"id":"ln_1","unit_cost":{"value":"2"}}]}`
	const hidden = `{"id":"th_1","unit_cost":null,"lines":[{"id":"ln_1","unit_cost":null}]}`
	tests := []struct {
		name     string
		identity *types.Identity
		want     string
	}{
		{"admin", costIdentity(types.IdentityRelationTypeInternal, constants.RoleTypeAdmin), shown},
		{"internal with costs:read", costIdentity(types.IdentityRelationTypeInternal, constants.RoleTypeCustom, "costs:read"), shown},
		{"internal without costs:read", costIdentity(types.IdentityRelationTypeInternal, constants.RoleTypeCustom, "items:read"), hidden},
		{"customer portal actor carrying costs:read", costIdentity(types.IdentityRelationTypeCustomer, "", "costs:read"), hidden},
		{"supplier portal actor carrying costs:read", costIdentity(types.IdentityRelationTypeSupplier, "", "costs:read"), hidden},
		{"agent whose role grants costs:read", agentCostIdentity("costs:read", "items:read"), hidden},
		{"no identity", nil, hidden},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := httptest.NewRequest(http.MethodGet, "/v1/things", nil)
			if tt.identity != nil {
				r = r.WithContext(appctx.WithIdentity(r.Context(), tt.identity))
			}
			w := httptest.NewRecorder()

			ep.Execute(w, r)

			require.Equal(t, http.StatusOK, w.Code, w.Body.String())
			assert.JSONEq(t, tt.want, w.Body.String())
		})
	}
}

// Cost fields are kept out of the request log for everyone, since its readers need not hold costs:read.
func TestExecute_costFieldsAreSensitiveInTheRequestLog(t *testing.T) {
	t.Parallel()

	ep := &APIEndpoint[*stubRequest, *stubCostResponse]{
		Method:            http.MethodGet,
		Route:             "/v1/things",
		SuccessStatusCode: http.StatusOK,
		ServiceHandler: func(svc any) func(context.Context, *stubRequest) (*stubCostResponse, *apierror.APIError) {
			return func(context.Context, *stubRequest) (*stubCostResponse, *apierror.APIError) {
				return &stubCostResponse{ID: "th_1"}, nil
			}
		},
	}
	bindHandler(ep)

	rl := &appctx.RequestLog{ID: "rq_cost"}
	r := httptest.NewRequest(http.MethodGet, "/v1/things", nil)
	r = r.WithContext(appctx.WithRequestLog(r.Context(), rl))
	ep.Execute(httptest.NewRecorder(), r)

	assert.Equal(t, map[string]bool{"unit_cost": true, "lines.unit_cost": true}, rl.SensitiveResponseFields)
}

// An agent's tool results are kept on its run, which any member of the account can read, so a report that is nothing but cost is refused to every agent, whatever its role grants.
func TestExecute_refusesAgentsACostOnlyEndpoint(t *testing.T) {
	t.Parallel()

	ep := &APIEndpoint[*stubRequest, *stubCostResponse]{
		Method:              http.MethodGet,
		Route:               "/v1/things/costs",
		SuccessStatusCode:   http.StatusOK,
		RequiredPermissions: types.AnyOfPermissions{{Domain: types.PermissionDomainCosts, Action: types.ActionRead}},
		ServiceHandler: func(svc any) func(context.Context, *stubRequest) (*stubCostResponse, *apierror.APIError) {
			return func(context.Context, *stubRequest) (*stubCostResponse, *apierror.APIError) {
				return &stubCostResponse{ID: "th_1", UnitCost: &stubCostRate{Value: "4"}}, nil
			}
		},
	}
	bindHandler(ep)

	tests := []struct {
		name     string
		identity *types.Identity
		want     int
	}{
		{"agent whose role grants costs:read", agentCostIdentity("costs:read"), http.StatusForbidden},
		{"agent carrying an admin role type", func() *types.Identity {
			identity := costIdentity(types.IdentityRelationTypeInternal, constants.RoleTypeAdmin, "costs:read")
			identity.Type = types.IdentityActorTypeAgent
			return identity
		}(), http.StatusForbidden},
		{"user whose role grants costs:read", costIdentity(types.IdentityRelationTypeInternal, constants.RoleTypeCustom, "costs:read"), http.StatusOK},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := httptest.NewRequest(http.MethodGet, "/v1/things/costs", nil)
			r = r.WithContext(appctx.WithIdentity(r.Context(), tt.identity))
			w := httptest.NewRecorder()

			ep.Execute(w, r)

			require.Equal(t, tt.want, w.Code, w.Body.String())
			if tt.want == http.StatusForbidden {
				assert.Contains(t, w.Body.String(), "agents never receive cost")
			}
		})
	}
}
