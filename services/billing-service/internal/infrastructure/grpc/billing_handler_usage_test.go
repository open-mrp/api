package grpc

import (
	"context"
	"testing"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/services/billing-service/internal/domain"
	servicemock "github.com/open-mrp/api/services/billing-service/internal/domain/mock/service"
	"github.com/open-mrp/api/shared/appctx"
	"github.com/open-mrp/api/shared/contracts"
	apierror "github.com/open-mrp/api/shared/errors"
	pb "github.com/open-mrp/api/shared/proto/billing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

const usageSellerID = "ac_seller"

func usageCtx(relation types.IdentityRelationType, actorAccount string) context.Context {
	return appctx.WithIdentity(context.Background(), &types.Identity{
		Type:   types.IdentityActorTypeAPIKey,
		Target: &types.IdentityTarget{AccountID: usageSellerID},
		Actor:  &types.IdentityActor{RelationType: relation, ID: "apky_1", AccountID: &actorAccount, Permissions: map[string]bool{"account:read": true}},
	})
}

// A portal key may hold account:read for its own account; the seller's usage is still not its to read. The service mock has no expectation, so reaching it fails the test.
func TestGetAccountUsage_PortalsAreRefused(t *testing.T) {
	t.Parallel()

	for name, ctx := range map[string]context.Context{
		"customer portal": usageCtx(types.IdentityRelationTypeCustomer, "ac_customer"),
		"supplier portal": usageCtx(types.IdentityRelationTypeSupplier, "ac_supplier"),
	} {
		handler := &billingHandler{billingSvc: servicemock.NewMockBillingSvc(gomock.NewController(t))}

		_, err := handler.GetAccountUsage(ctx, &pb.GetAccountUsageRequest{})
		apiErr := contracts.ConvertGRPCError(ctx, err, "billing-service")
		require.NotNil(t, apiErr, name)
		assert.Equal(t, apierror.ErrorCodeInsufficientPerms, apiErr.Code, name)
	}
}

func TestGetAccountUsage_SellerStaffReadIt(t *testing.T) {
	t.Parallel()

	svc := servicemock.NewMockBillingSvc(gomock.NewController(t))
	svc.EXPECT().GetAccountUsage(gomock.Any(), usageSellerID).Return(&domain.AccountUsage{Seats: domain.UsageItem{Current: 3}}, nil)
	handler := &billingHandler{billingSvc: svc}

	resp, err := handler.GetAccountUsage(usageCtx(types.IdentityRelationTypeInternal, usageSellerID), &pb.GetAccountUsageRequest{})
	require.NoError(t, err)
	assert.Equal(t, int32(3), resp.GetSeats().GetCurrent())
}
