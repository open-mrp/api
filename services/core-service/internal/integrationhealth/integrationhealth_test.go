package integrationhealth

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	clientmock "github.com/open-mrp/api/services/core-service/internal/domain/mock/client"
	repositorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/repository"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
)

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

func TestShippo_RecordsEachCall(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	repo := repositorymock.NewMockAccountIntegrationRepo(ctrl)
	inner := clientmock.NewMockShippoClient(ctrl)
	client := Shippo(inner, ForAccount(repo, "ac_1", constants.IntegrationCodeShippo))

	inner.EXPECT().RefundTransaction(gomock.Any(), "tx_1").Return(nil)
	repo.EXPECT().SetConnectionStatus(gomock.Any(), "ac_1", constants.IntegrationCodeShippo, constants.IntegrationConnectionStatusConnected).Return(nil)
	assert.Nil(t, client.RefundTransaction(context.Background(), "tx_1"))

	rejected := apierror.NewInternalError(fmt.Errorf("%w: 401", domain.ErrProviderRejectedCredentials), "Shippo API error.")
	inner.EXPECT().RefundTransaction(gomock.Any(), "tx_2").Return(rejected)
	repo.EXPECT().SetConnectionStatus(gomock.Any(), "ac_1", constants.IntegrationCodeShippo, constants.IntegrationConnectionStatusInvalidCredentials).Return(nil)
	assert.Equal(t, rejected, client.RefundTransaction(context.Background(), "tx_2"), "the caller sees the provider's error unchanged")

	inner.EXPECT().RefundTransaction(gomock.Any(), "tx_3").Return(apierror.NewInternalError(errors.New("boom"), "Shippo API error."))
	assert.NotNil(t, client.RefundTransaction(context.Background(), "tx_3"))
}
