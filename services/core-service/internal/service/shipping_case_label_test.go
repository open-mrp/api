package service

import (
	"context"
	"testing"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/services/core-service/internal/domain"
	factorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/factory"
	repositorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/repository"
	"github.com/open-mrp/api/shared/appctx"
	apierror "github.com/open-mrp/api/shared/errors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// Reports every label as missing from the bucket, as when its copy never landed.
type missingLabelStore struct{ capturingObjectStore }

func (*missingLabelStore) FileExists(context.Context, string, string) (bool, *apierror.APIError) {
	return false, nil
}

func TestGetShippingCaseLabel_FallsBackToTheCarrierHostedLabel(t *testing.T) {
	tests := []struct {
		name   string
		stored *string
		want   *string
	}{
		{"the carrier's url when the copy never landed", strPtr("https://deliver.goshippo.com/label.png"), strPtr("https://deliver.goshippo.com/label.png")},
		{"nothing for the dashboard's relative path", strPtr("files/shipping-labels/shc_1"), nil},
		{"nothing when no label was bought", nil, nil},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			caseRepo := repositorymock.NewMockShippingCaseRepo(ctrl)
			repoFactory := factorymock.NewMockRepoFactory(ctrl)
			repoFactory.EXPECT().NewShippingCaseRepo().Return(caseRepo).AnyTimes()

			caseRepo.EXPECT().GetNumber(gomock.Any(), testLabelAccountID, "shc_1").Return("1001", nil)
			caseRepo.EXPECT().Get(gomock.Any(), testLabelAccountID, "shc_1").
				Return(&domain.ShippingCase{ID: "shc_1", ShippingLabelURL: tc.stored}, nil)

			svc := &shippingCaseSvcImpl{repos: repoFactory, s3Client: &missingLabelStore{}, shippingLabelsBucket: "labels"}
			ctx := appctx.WithIdentity(context.Background(), &types.Identity{
				Type:   types.IdentityActorTypeUser,
				Target: &types.IdentityTarget{AccountID: testLabelAccountID},
				Actor: &types.IdentityActor{
					RelationType: types.IdentityRelationTypeInternal,
					ID:           "usr_internal",
					AccountID:    strPtr(testLabelAccountID),
					Permissions:  map[string]bool{"shipments:read": true},
				},
			})
			got, apiErr := svc.GetShippingCaseLabel(ctx, testLabelAccountID, "shc_1")
			require.Nil(t, apiErr)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestDeleteShippingCase_RefusesACaseWithABoughtLabel(t *testing.T) {
	ctrl := gomock.NewController(t)
	caseRepo := repositorymock.NewMockShippingCaseRepo(ctrl)
	repoFactory := factorymock.NewMockRepoFactory(ctrl)
	repoFactory.EXPECT().NewShippingCaseRepo().Return(caseRepo).AnyTimes()
	caseRepo.EXPECT().Get(gomock.Any(), testLabelAccountID, "shc_1").
		Return(&domain.ShippingCase{ID: "shc_1", ShippoTransactionID: strPtr("txn_1")}, nil)

	ctx := appctx.WithIdentity(context.Background(), &types.Identity{
		Type:   types.IdentityActorTypeUser,
		Target: &types.IdentityTarget{AccountID: testLabelAccountID},
		Actor: &types.IdentityActor{
			RelationType: types.IdentityRelationTypeInternal,
			ID:           "usr_internal",
			AccountID:    strPtr(testLabelAccountID),
			Permissions:  map[string]bool{"shipments:delete": true},
		},
	})

	// No transaction is expected: deleting the case would orphan a label nobody could refund.
	svc := &shippingCaseSvcImpl{repos: repoFactory}
	apiErr := svc.DeleteShippingCase(ctx, testLabelAccountID, "shc_1")
	require.NotNil(t, apiErr)
	assert.Equal(t, apierror.ErrorCodeResourceConflict, apiErr.Code)
}
