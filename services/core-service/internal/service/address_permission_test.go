package service

import (
	"testing"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"

	"github.com/stretchr/testify/assert"
)

type addressPermissionCase struct {
	relation types.IdentityRelationType
	holds    string
	allowed  bool
}

// roleHolder is a seller's staff member with a custom role granting exactly perm, acting in the account relation names.
func roleHolder(t *testing.T, relation types.IdentityRelationType, perm string) *types.Identity {
	identity := mustIdentity(t, sellerCtx(relation, perm))
	roleType := string(constants.RoleTypeCustom)
	identity.Actor.RoleType = &roleType
	return identity
}

func assertAddressPermission(t *testing.T, what string, check func(*types.Identity) *apierror.APIError, cases []addressPermissionCase) {
	t.Helper()
	for _, tc := range cases {
		apiErr := check(roleHolder(t, tc.relation, tc.holds))
		if tc.allowed {
			assert.Nil(t, apiErr, "%s in a %q account with %s", what, tc.relation, tc.holds)
			continue
		}
		if assert.NotNil(t, apiErr, "%s in a %q account with %s", what, tc.relation, tc.holds) {
			assert.Equal(t, apierror.ErrorCodeInsufficientPerms, apiErr.Code)
		}
	}
}

// Reading addresses takes addresses:read in the seller's own account; customers:read and suppliers:read reach only a customer's and a supplier's.
func TestAddressReadPermission_FollowsTheTarget(t *testing.T) {
	t.Parallel()
	customer, supplier := types.IdentityRelationTypeCustomer, types.IdentityRelationTypeSupplier
	assertAddressPermission(t, "read", checkAddressReadPermission, []addressPermissionCase{
		{"", "addresses:read", true},
		{"", "customers:read", false},
		{"", "suppliers:read", false},
		{customer, "customers:read", true},
		{customer, "addresses:read", false},
		{customer, "suppliers:read", false},
		{supplier, "suppliers:read", true},
		{supplier, "addresses:read", false},
		{supplier, "customers:read", false},
	})
}
