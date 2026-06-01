package apiendpoint

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/augno/api/services/api-gateway/pkg/resourcekit"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// V2 integration tests validate the path: parseIncludeTree → ResolveIncludes →
// direct JSON response. They use a fake resource registered just for the
// duration of each test to avoid polluting the global registry.

const (
	v2OTCarrier constants.ObjectType = "v2_test_carrier"
	v2OTOwner   constants.ObjectType = "v2_test_owner"
)

// v2OwnerIncludeConfig is the per-endpoint allow-list every V2 test uses.
// Real endpoints construct this via apiendpoint.IncludesFor; tests bypass
// that helper to avoid requiring registration in the legacy include_registry.
var v2OwnerIncludeConfig = &IncludeConfig{
	Fields: []IncludeField{{Key: "owner", ObjectType: v2OTOwner, JSONPaths: []string{"owner"}}},
}

type v2Carrier struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Name    string   `json:"name"`
	OwnerID string   `json:"-"`
	Owner   *v2Owner `json:"owner"`
}

type v2Owner struct {
	ID     string `json:"id"`
	Object string `json:"object"`
	Name   string `json:"name"`
}

func registerV2Fixture(t *testing.T) {
	t.Helper()
	resourcekit.ResetForTest()
	resourcekit.Register(&resourcekit.Definition{
		ObjectType: v2OTCarrier,
		Load: func(_ context.Context, ids []string) (map[string]any, *apierror.APIError) {
			out := map[string]any{}
			for _, id := range ids {
				out[id] = &v2Carrier{ID: id, Object: string(v2OTCarrier), Name: "Carrier " + id, OwnerID: "o-" + id}
			}
			return out, nil
		},
		Subs: []resourcekit.SubField{
			{
				Key: "owner", Target: v2OTOwner, Cardinality: resourcekit.CardinalityOnePtr,
				ExtractIDs: func(_ context.Context, p any) []string {
					c := p.(*v2Carrier)
					if c.OwnerID == "" {
						return nil
					}
					return []string{c.OwnerID}
				},
				Populate: func(_ context.Context, p any, loaded map[string]any) {
					c := p.(*v2Carrier)
					if v, ok := loaded[c.OwnerID]; ok {
						c.Owner = v.(*v2Owner)
					}
				},
			},
		},
	})
	resourcekit.Register(&resourcekit.Definition{
		ObjectType: v2OTOwner,
		Load: func(_ context.Context, ids []string) (map[string]any, *apierror.APIError) {
			out := map[string]any{}
			for _, id := range ids {
				out[id] = &v2Owner{ID: id, Object: string(v2OTOwner), Name: "Owner " + id}
			}
			return out, nil
		},
	})
	t.Cleanup(resourcekit.ResetForTest)
}

func TestExecuteV2_NoIncludes_LeavesSubResourceNull(t *testing.T) {
	registerV2Fixture(t)
	ep := &APIEndpoint[*stubRequest, *v2Carrier]{
		Method:            http.MethodGet,
		Route:             "/v1/v2carriers/:id",
		SuccessStatusCode: http.StatusOK,
		ObjectType:        v2OTCarrier,
		IncludeConfig:     v2OwnerIncludeConfig,
		ServiceHandler: func(svc any) func(context.Context, *stubRequest) (*v2Carrier, *apierror.APIError) {
			return func(context.Context, *stubRequest) (*v2Carrier, *apierror.APIError) {
				return &v2Carrier{ID: "c1", Object: string(v2OTCarrier), Name: "Carrier c1", OwnerID: "o-c1"}, nil
			}
		},
	}
	bindHandler(ep)

	r := httptest.NewRequest(http.MethodGet, "/v1/v2carriers/c1", nil)
	w := httptest.NewRecorder()
	ep.Execute(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	var got map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v body=%s", err, w.Body.String())
	}
	if got["owner"] != nil {
		t.Errorf("owner should be null when not included, got %v", got["owner"])
	}
}

func TestExecuteV2_IncludeOwner_PopulatesSubResource(t *testing.T) {
	registerV2Fixture(t)
	ep := &APIEndpoint[*stubRequest, *v2Carrier]{
		Method:            http.MethodGet,
		Route:             "/v1/v2carriers/:id",
		SuccessStatusCode: http.StatusOK,
		ObjectType:        v2OTCarrier,
		IncludeConfig:     v2OwnerIncludeConfig,
		ServiceHandler: func(svc any) func(context.Context, *stubRequest) (*v2Carrier, *apierror.APIError) {
			return func(context.Context, *stubRequest) (*v2Carrier, *apierror.APIError) {
				return &v2Carrier{ID: "c1", Object: string(v2OTCarrier), Name: "Carrier c1", OwnerID: "o-c1"}, nil
			}
		},
	}
	bindHandler(ep)

	r := httptest.NewRequest(http.MethodGet, "/v1/v2carriers/c1?include[]=owner", nil)
	w := httptest.NewRecorder()
	ep.Execute(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	var got map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v body=%s", err, w.Body.String())
	}
	ownerMap, ok := got["owner"].(map[string]any)
	if !ok {
		t.Fatalf("expected owner object, got %T (%v)", got["owner"], got["owner"])
	}
	if ownerMap["id"] != "o-c1" {
		t.Errorf("expected owner.id=o-c1, got %v", ownerMap["id"])
	}
	if ownerMap["name"] != "Owner o-c1" {
		t.Errorf("expected owner.name to come from loader (no hallucination), got %v", ownerMap["name"])
	}
}

// TestExecuteV2_OutOfScopeInclude_Rejected is the keystone safety property:
// the resolver knows how to traverse arbitrary registered paths, but the
// per-endpoint IncludeConfig.Fields allow-list MUST veto anything outside
// the explicit whitelist. Without this guard, a client could request
// `?include[]=owner.account.something_else` and walk the whole graph.
//
// The fixture registers `owner` AND a hypothetical sibling sub-resource on
// the carrier; the endpoint allow-list only lists `owner`. Requesting the
// sibling must fail with 400.
func TestExecuteV2_OutOfScopeInclude_Rejected(t *testing.T) {
	registerV2Fixture(t) // registers v2OTCarrier with `owner` only
	ep := &APIEndpoint[*stubRequest, *v2Carrier]{
		Method:            http.MethodGet,
		Route:             "/v1/v2carriers/:id",
		SuccessStatusCode: http.StatusOK,
		ObjectType:        v2OTCarrier,
		// Allow-list intentionally empty — even `owner` (which IS registered
		// in the resourcekit graph) cannot be requested by clients of this
		// particular endpoint.
		IncludeConfig: &IncludeConfig{Fields: nil},
		ServiceHandler: func(svc any) func(context.Context, *stubRequest) (*v2Carrier, *apierror.APIError) {
			return func(context.Context, *stubRequest) (*v2Carrier, *apierror.APIError) {
				t.Fatal("handler should not run when include validation fails")
				return nil, nil
			}
		},
	}
	bindHandler(ep)

	r := httptest.NewRequest(http.MethodGet, "/v1/v2carriers/c1?include[]=owner", nil)
	w := httptest.NewRecorder()
	ep.Execute(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for include outside endpoint allow-list, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestExecuteV2_InvalidInclude_Rejected(t *testing.T) {
	registerV2Fixture(t)
	ep := &APIEndpoint[*stubRequest, *v2Carrier]{
		Method:            http.MethodGet,
		Route:             "/v1/v2carriers/:id",
		SuccessStatusCode: http.StatusOK,
		ObjectType:        v2OTCarrier,
		IncludeConfig:     v2OwnerIncludeConfig,
		ServiceHandler: func(svc any) func(context.Context, *stubRequest) (*v2Carrier, *apierror.APIError) {
			return func(context.Context, *stubRequest) (*v2Carrier, *apierror.APIError) {
				t.Fatal("handler should not run when include validation fails")
				return nil, nil
			}
		},
	}
	bindHandler(ep)

	r := httptest.NewRequest(http.MethodGet, "/v1/v2carriers/c1?include[]=not_a_field", nil)
	w := httptest.NewRecorder()
	ep.Execute(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid include, got %d body=%s", w.Code, w.Body.String())
	}
}
