package searchep

import (
	"testing"

	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ent builds a minimal Entity carrying an identifiable name so interleave order is assertable.
func ent(name string) apiresource.Entity {
	return *apiresource.NewEntity("id_"+name, constants.ObjectTypeSalesOrder, &name, nil)
}

func names(ents []apiresource.Entity) []string {
	out := make([]string, len(ents))
	for i, e := range ents {
		out[i] = *e.Name
	}
	return out
}

func TestInterleave(t *testing.T) {
	cases := []struct {
		name    string
		results [][]apiresource.Entity
		limit   int
		want    []string
	}{
		{
			name:    "round-robins across types",
			results: [][]apiresource.Entity{{ent("a1"), ent("a2")}, {ent("b1"), ent("b2")}},
			limit:   10,
			want:    []string{"a1", "b1", "a2", "b2"},
		},
		{
			name:    "caps at limit mid-round",
			results: [][]apiresource.Entity{{ent("a1"), ent("a2")}, {ent("b1"), ent("b2")}},
			limit:   3,
			want:    []string{"a1", "b1", "a2"},
		},
		{
			name:    "drains longer slice once others exhausted",
			results: [][]apiresource.Entity{{ent("a1")}, {ent("b1"), ent("b2"), ent("b3")}},
			limit:   10,
			want:    []string{"a1", "b1", "b2", "b3"},
		},
		{
			name:    "skips empty result slices",
			results: [][]apiresource.Entity{{}, {ent("b1")}, {}},
			limit:   10,
			want:    []string{"b1"},
		},
		{
			name:    "no results yields empty (non-nil) slice",
			results: [][]apiresource.Entity{{}, {}},
			limit:   10,
			want:    []string{},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := interleave(c.results, c.limit)
			assert.Equal(t, c.want, names(got))
		})
	}
}

func TestNonEmpty(t *testing.T) {
	assert.Nil(t, nonEmpty(""))
	if got := nonEmpty("PO-1"); assert.NotNil(t, got) {
		assert.Equal(t, "PO-1", *got)
	}
}

func TestSelectActiveSearchProviders(t *testing.T) {
	providers := []searchProvider{
		{constants.ObjectTypeCustomer, types.Permission{Domain: types.PermissionDomainCustomers, Action: types.ActionRead}, false, nil},
		{constants.ObjectTypeSalesOrder, types.Permission{Domain: types.PermissionDomainSalesOrders, Action: types.ActionRead}, true, nil},
		{constants.ObjectTypeInvoice, types.Permission{Domain: types.PermissionDomainInvoices, Action: types.ActionRead}, true, nil},
	}

	identity := &types.Identity{
		Type: types.IdentityActorTypeUser,
		Actor: &types.IdentityActor{
			ID:          "us_test",
			Permissions: map[string]bool{"customers:read": true, "sales_orders:read": true},
		},
	}

	active := selectActiveSearchProviders(providers, nil, identity, searchScope{})
	require.Len(t, active, 2)
	assert.Equal(t, constants.ObjectTypeCustomer, active[0].objectType)

	filtered := selectActiveSearchProviders(providers, map[constants.ObjectType]bool{constants.ObjectTypeInvoice: true}, identity, searchScope{})
	assert.Empty(t, filtered)

	filtered = selectActiveSearchProviders(providers, map[constants.ObjectType]bool{constants.ObjectTypeCustomer: true}, identity, searchScope{})
	require.Len(t, filtered, 1)
	assert.Equal(t, constants.ObjectTypeCustomer, filtered[0].objectType)

	// A customer-scoped search drops every non-customer-safe provider, even ones the caller can read.
	// Here only sales_order is customer-safe and permitted, so the privileged customer type is excluded.
	scoped := selectActiveSearchProviders(providers, nil, identity, searchScope{customerID: "acc_cust"})
	require.Len(t, scoped, 1)
	assert.Equal(t, constants.ObjectTypeSalesOrder, scoped[0].objectType)
}

func TestSearchReadPermissions(t *testing.T) {
	identity := &types.Identity{
		Type: types.IdentityActorTypeUser,
		Actor: &types.IdentityActor{
			ID:          "us_test",
			Permissions: map[string]bool{"products:read": true},
		},
	}
	assert.NotNil(t, identity.CheckHasAnyPermission(searchReadPermissions...))

	identity.Actor.Permissions = map[string]bool{"units:read": true}
	assert.NotNil(t, identity.CheckHasAnyPermission(searchReadPermissions...))

	identity.Actor.Permissions = map[string]bool{"customers:read": true}
	assert.Nil(t, identity.CheckHasAnyPermission(searchReadPermissions...))
}

func TestSearchQueryFromRequest(t *testing.T) {
	q := "  PO-1001  "
	assert.Equal(t, "PO-1001", searchQueryFromRequest(&SearchRequest{PaginationRequest: apiresource.PaginationRequest{Query: &q}}))

	// A missing or whitespace-only query normalizes to empty; whether that is allowed is decided by the caller (only when scoped with types).
	assert.Equal(t, "", searchQueryFromRequest(&SearchRequest{}))

	empty := "   "
	assert.Equal(t, "", searchQueryFromRequest(&SearchRequest{PaginationRequest: apiresource.PaginationRequest{Query: &empty}}))
}

func TestSearchTypeFilterFromRequest(t *testing.T) {
	filter, apiErr := searchTypeFilterFromRequest(&SearchRequest{})
	require.Nil(t, apiErr)
	assert.Nil(t, filter)

	filter, apiErr = searchTypeFilterFromRequest(&SearchRequest{Types: []constants.ObjectType{constants.ObjectTypeCustomer, constants.ObjectTypeInvoice}})
	require.Nil(t, apiErr)
	assert.True(t, filter[constants.ObjectTypeCustomer])
	assert.True(t, filter[constants.ObjectTypeInvoice])

	_, apiErr = searchTypeFilterFromRequest(&SearchRequest{Types: []constants.ObjectType{constants.ObjectTypeUser}})
	require.NotNil(t, apiErr)
}

func TestIsSearchObjectType(t *testing.T) {
	assert.True(t, isSearchObjectType(constants.ObjectTypeSalesOrder))
	assert.False(t, isSearchObjectType(constants.ObjectTypeUser))
}
