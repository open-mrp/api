package billingep

import (
	"context"
	"testing"

	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/appctx"
	apierror "github.com/open-mrp/api/shared/errors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The service holds no clients, so a portal that got past the refusal would fail on the nil core client.
func TestGetSpendingCap_PortalsAreRefused(t *testing.T) {
	t.Parallel()

	for name, relation := range map[string]types.IdentityRelationType{
		"customer portal": types.IdentityRelationTypeCustomer,
		"supplier portal": types.IdentityRelationTypeSupplier,
	} {
		actorAccount := "ac_portal"
		ctx := appctx.WithIdentity(context.Background(), &types.Identity{
			Type:   types.IdentityActorTypeAPIKey,
			Target: &types.IdentityTarget{AccountID: "ac_seller"},
			Actor:  &types.IdentityActor{RelationType: relation, ID: "apky_1", AccountID: &actorAccount, Permissions: map[string]bool{"account:read": true}},
		})

		_, apiErr := (&billingSvcImpl{}).GetSpendingCap(ctx, &apiresource.EmptyResource{})
		require.NotNil(t, apiErr, name)
		assert.Equal(t, apierror.ErrorCodeInsufficientPerms, apiErr.Code, name)
	}
}
