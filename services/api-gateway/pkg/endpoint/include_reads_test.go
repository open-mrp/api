package apiendpoint

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/open-mrp/api/services/api-gateway/pkg/resourcekit"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/appctx"
	apierror "github.com/open-mrp/api/shared/errors"
)

// The handler runs as the caller; only the include loads carry the include-reads flag.
func TestExecute_OnlyIncludeLoadsReadAsTheIncludingRequest(t *testing.T) {
	var ownerLoadedAs *types.Identity
	resourcekit.ResetForTest()
	t.Cleanup(resourcekit.ResetForTest)
	resourcekit.Register(&resourcekit.Definition{
		ObjectType: v2OTCarrier,
		Load: func(context.Context, []string) (map[string]any, *apierror.APIError) {
			t.Error("the root carrier was loaded through the resolver")
			return nil, nil
		},
		Subs: []resourcekit.SubField{{
			Key: "owner", Target: v2OTOwner, Cardinality: resourcekit.CardinalityOnePtr,
			ExtractIDs: func(_ context.Context, p any) []string { return []string{p.(*v2Carrier).OwnerID} },
			Populate: func(_ context.Context, p any, loaded map[string]any) {
				c := p.(*v2Carrier)
				if v, ok := loaded[c.OwnerID]; ok {
					c.Owner = v.(*v2Owner)
				}
			},
		}},
	})
	resourcekit.Register(&resourcekit.Definition{
		ObjectType: v2OTOwner,
		Load: func(ctx context.Context, ids []string) (map[string]any, *apierror.APIError) {
			ownerLoadedAs, _ = appctx.GetIdentityFromContext(ctx)
			out := map[string]any{}
			for _, id := range ids {
				out[id] = &v2Owner{ID: id}
			}
			return out, nil
		},
	})

	var handledAs *types.Identity
	ep := &APIEndpoint[*stubRequest, *v2Carrier]{
		Method:            http.MethodGet,
		Route:             "/v1/v2carriers/:id",
		SuccessStatusCode: http.StatusOK,
		ObjectType:        v2OTCarrier,
		IncludeConfig:     v2OwnerIncludeConfig,
		ServiceHandler: func(any) func(context.Context, *stubRequest) (*v2Carrier, *apierror.APIError) {
			return func(ctx context.Context, _ *stubRequest) (*v2Carrier, *apierror.APIError) {
				handledAs, _ = appctx.GetIdentityFromContext(ctx)
				return &v2Carrier{ID: "c1", OwnerID: "o-c1"}, nil
			}
		},
	}
	bindHandler(ep)

	account := "acct_1"
	caller := &types.Identity{
		Type:   types.IdentityActorTypeUser,
		Target: &types.IdentityTarget{AccountID: account},
		Actor:  &types.IdentityActor{RelationType: types.IdentityRelationTypeInternal, ID: "usr_1", AccountID: &account},
	}
	r := httptest.NewRequest(http.MethodGet, "/v1/v2carriers/c1?include[]=owner", nil)
	r = r.WithContext(appctx.WithIdentity(r.Context(), caller))
	w := httptest.NewRecorder()
	ep.Execute(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	if handledAs != caller || handledAs.IncludeReads {
		t.Errorf("the handler ran as %+v, want the caller unflagged", handledAs)
	}
	if ownerLoadedAs == nil || !ownerLoadedAs.IncludeReads || ownerLoadedAs.Actor != caller.Actor {
		t.Errorf("the include loaded as %+v, want a flagged copy of the caller", ownerLoadedAs)
	}
	if caller.IncludeReads {
		t.Error("the request's identity was flagged")
	}
}
