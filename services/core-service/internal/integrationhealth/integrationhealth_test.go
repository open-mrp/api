package integrationhealth

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/services/core-service/internal/domain"
	clientmock "github.com/open-mrp/api/services/core-service/internal/domain/mock/client"
	factorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/factory"
	repositorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/repository"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/messaging"
	apierror "github.com/open-mrp/apikit/apierror"
	"github.com/open-mrp/apikit/appctx"
)

type countingOutbox struct{ n int }

func (o *countingOutbox) Create(context.Context, messaging.OutboxMessageInput) (int64, error) {
	o.n++
	return 0, nil
}

func TestStatusOf(t *testing.T) {
	t.Parallel()

	status, ok := StatusOf(nil)
	assert.True(t, ok)
	assert.Equal(t, constants.IntegrationConnectionStatusConnected, status)

	rejected := apierror.NewInternalError(fmt.Errorf("%w: shippo status 401", domain.ErrProviderRejectedCredentials), "Shippo API error.")
	status, ok = StatusOf(rejected)
	assert.True(t, ok)
	assert.Equal(t, constants.IntegrationConnectionStatusInvalidCredentials, status)

	_, ok = StatusOf(apierror.NewInternalError(errors.New("timeout"), "Shippo API error."))
	assert.False(t, ok, "a failure unrelated to the credentials says nothing about them")
}

// Each call through a wrapped client records its outcome; only a change writes, and it is audited under the caller.
func TestShippo_RecordsEachCall(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	repo := repositorymock.NewMockAccountIntegrationRepo(ctrl)
	outbox := &countingOutbox{}
	repos := factorymock.NewMockRepoFactory(ctrl)
	repos.EXPECT().NewAccountIntegrationRepo().Return(repo).AnyTimes()
	repos.EXPECT().NewOutboxRepo().Return(outbox).AnyTimes()
	inner := clientmock.NewMockShippoClient(ctrl)
	client := Shippo(inner, ForAccount(repos, "ac_1", constants.IntegrationCodeShippo))
	ctx := appctx.WithIdentity(context.Background(), &types.Identity{
		Type:   types.IdentityActorTypeUser,
		Target: &types.IdentityTarget{AccountID: "ac_1"},
	})
	integration := func(status constants.IntegrationConnectionStatus) *domain.AccountIntegration {
		return &domain.AccountIntegration{ID: "acig_1", AccountID: "ac_1", IntegrationCode: constants.IntegrationCodeShippo, ConnectionStatus: status}
	}

	// Already connected: a successful call reads and writes nothing more.
	inner.EXPECT().RefundTransaction(gomock.Any(), "tx_1").Return(nil)
	repo.EXPECT().FindByCode(gomock.Any(), "ac_1", constants.IntegrationCodeShippo).Return(integration(constants.IntegrationConnectionStatusConnected), nil)
	assert.Nil(t, client.RefundTransaction(ctx, "tx_1"))
	assert.Equal(t, 0, outbox.n)

	// Rejected credentials flip it and audit the change; the caller sees the provider's error unchanged.
	rejected := apierror.NewInternalError(fmt.Errorf("%w: 401", domain.ErrProviderRejectedCredentials), "Shippo API error.")
	inner.EXPECT().RefundTransaction(gomock.Any(), "tx_2").Return(rejected)
	repo.EXPECT().FindByCode(gomock.Any(), "ac_1", constants.IntegrationCodeShippo).Return(integration(constants.IntegrationConnectionStatusConnected), nil)
	repo.EXPECT().SetConnectionStatus(gomock.Any(), "ac_1", constants.IntegrationCodeShippo, constants.IntegrationConnectionStatusInvalidCredentials).Return(true, nil)
	assert.Equal(t, rejected, client.RefundTransaction(ctx, "tx_2"))
	assert.Equal(t, 1, outbox.n)

	// A failure unrelated to the credentials leaves it alone.
	inner.EXPECT().RefundTransaction(gomock.Any(), "tx_3").Return(apierror.NewInternalError(errors.New("boom"), "Shippo API error."))
	assert.NotNil(t, client.RefundTransaction(ctx, "tx_3"))

	// Without an identity for the account the change is still recorded, just not audited.
	inner.EXPECT().RefundTransaction(gomock.Any(), "tx_4").Return(nil)
	repo.EXPECT().FindByCode(gomock.Any(), "ac_1", constants.IntegrationCodeShippo).Return(integration(constants.IntegrationConnectionStatusInvalidCredentials), nil)
	repo.EXPECT().SetConnectionStatus(gomock.Any(), "ac_1", constants.IntegrationCodeShippo, constants.IntegrationConnectionStatusConnected).Return(true, nil)
	assert.Nil(t, client.RefundTransaction(context.Background(), "tx_4"))
	assert.Equal(t, 1, outbox.n)
}
