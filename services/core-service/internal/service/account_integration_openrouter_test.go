package service

import (
	"bytes"
	"context"
	"testing"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/services/core-service/internal/domain"
	clientmock "github.com/open-mrp/api/services/core-service/internal/domain/mock/client"
	factorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/factory"
	repositorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/repository"
	"github.com/open-mrp/api/shared/appctx"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/crypto"
	apierror "github.com/open-mrp/api/shared/errors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/mock/gomock"
)

var openRouterTestKey = bytes.Repeat([]byte{7}, crypto.AESGCMKeyBytes)

func TestValidateOpenRouterCredentials(t *testing.T) {
	svc := &accountIntegrationSvcImpl{}
	span := trace.SpanFromContext(context.Background())

	assert.Nil(t, svc.validateOpenRouterCredentials(span, `{"api_key":"sk-or-v1-abc"}`))
	assert.NotNil(t, svc.validateOpenRouterCredentials(span, `{"api_key":"sk-ant-abc"}`))
	assert.NotNil(t, svc.validateOpenRouterCredentials(span, `{"apiKey":"sk-or-v1-abc"}`))
	assert.NotNil(t, svc.validateOpenRouterCredentials(span, `not json`))
}

func TestNormalizeMonthlySpendLimit(t *testing.T) {
	tests := []struct {
		name   string
		code   constants.IntegrationCode
		limit  domain.Money
		want   string
		wantOK bool
	}{
		{"whole dollars", constants.IntegrationCodeOpenRouter, domain.Money{Amount: "100", Currency: "usd"}, "100", true},
		{"trailing zeros are dropped", constants.IntegrationCodeOpenRouter, domain.Money{Amount: "25.5000", Currency: "usd"}, "25.5", true},
		{"four decimal places", constants.IntegrationCodeOpenRouter, domain.Money{Amount: "0.0001", Currency: "usd"}, "0.0001", true},
		{"five decimal places", constants.IntegrationCodeOpenRouter, domain.Money{Amount: "0.00001", Currency: "usd"}, "", false},
		{"zero", constants.IntegrationCodeOpenRouter, domain.Money{Amount: "0", Currency: "usd"}, "", false},
		{"negative", constants.IntegrationCodeOpenRouter, domain.Money{Amount: "-5", Currency: "usd"}, "", false},
		{"too large", constants.IntegrationCodeOpenRouter, domain.Money{Amount: "1000000001", Currency: "usd"}, "", false},
		{"not a number", constants.IntegrationCodeOpenRouter, domain.Money{Amount: "lots", Currency: "usd"}, "", false},
		{"not dollars", constants.IntegrationCodeOpenRouter, domain.Money{Amount: "100", Currency: "eur"}, "", false},
		{"another provider", constants.IntegrationCodeStripe, domain.Money{Amount: "100", Currency: "usd"}, "", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, apiErr := normalizeMonthlySpendLimit(tc.code, &tc.limit)
			if !tc.wantOK {
				assert.NotNil(t, apiErr)
				return
			}
			require.Nil(t, apiErr)
			assert.Equal(t, tc.want, got.Amount)
			assert.Equal(t, "usd", got.Currency)
		})
	}
}

func TestMonthlySpendRemaining(t *testing.T) {
	limit := &domain.Money{Amount: "100", Currency: "usd"}

	assert.Equal(t, "68.1", monthlySpendRemaining(limit, "31.9").Amount)
	assert.Equal(t, "0", monthlySpendRemaining(limit, "140").Amount, "an overspent month has nothing left, not a negative amount")
	assert.Nil(t, monthlySpendRemaining(limit, "garbage"))
}

func agentCtx(accountID string) context.Context {
	return appctx.WithIdentity(context.Background(), &types.Identity{
		Type:   types.IdentityActorTypeAgent,
		Target: &types.IdentityTarget{AccountID: accountID},
		Actor: &types.IdentityActor{
			RelationType: types.IdentityRelationTypeInternal,
			ID:           "agd_test",
			AccountID:    &accountID,
		},
	})
}

type openRouterFixture struct {
	svc          *accountIntegrationSvcImpl
	accounts     *repositorymock.MockAccountRepo
	integrations *repositorymock.MockAccountIntegrationRepo
	openRouter   *clientmock.MockOpenRouterClient
}

func newOpenRouterFixture(t *testing.T) *openRouterFixture {
	ctrl := gomock.NewController(t)
	repos := factorymock.NewMockRepoFactory(ctrl)
	f := &openRouterFixture{
		accounts:     repositorymock.NewMockAccountRepo(ctrl),
		integrations: repositorymock.NewMockAccountIntegrationRepo(ctrl),
		openRouter:   clientmock.NewMockOpenRouterClient(ctrl),
	}
	repos.EXPECT().NewAccountRepo().Return(f.accounts).AnyTimes()
	repos.EXPECT().NewAccountIntegrationRepo().Return(f.integrations).AnyTimes()
	f.svc = &accountIntegrationSvcImpl{repos: repos, encryptionKey: openRouterTestKey, encryptionKeyID: "k1", openRouter: f.openRouter}
	return f
}

func (f *openRouterFixture) storeIntegration(t *testing.T, accountID string, isActive bool, limit *domain.Money) {
	encrypted, err := crypto.EncryptAESGCM([]byte(`{"api_key":"sk-or-v1-secret"}`), openRouterTestKey, []byte(accountID), "k1")
	require.NoError(t, err)
	f.integrations.EXPECT().GetStoredCredentials(gomock.Any(), accountID, constants.IntegrationCodeOpenRouter).
		Return(&domain.StoredIntegrationCredentials{Encrypted: encrypted, IsActive: isActive, MonthlySpendLimit: limit}, nil)
}

func TestGetOpenRouterAccess_NoIntegrationLeavesTheDeploymentKey(t *testing.T) {
	f := newOpenRouterFixture(t)
	f.accounts.EXPECT().GetAccountContext(gomock.Any(), "ac_main").Return(&domain.AccountContext{AccountID: "ac_main"}, nil)
	f.integrations.EXPECT().GetStoredCredentials(gomock.Any(), "ac_main", constants.IntegrationCodeOpenRouter).
		Return(nil, apierror.NewResourceNotFoundError("not found"))

	access, apiErr := f.svc.GetOpenRouterAccess(agentCtx("ac_main"))

	require.Nil(t, apiErr)
	assert.False(t, access.Configured)
	assert.Empty(t, access.APIKey)
}

func TestGetOpenRouterAccess_SandboxBillsToItsOwner(t *testing.T) {
	f := newOpenRouterFixture(t)
	owner := "ac_owner"
	f.accounts.EXPECT().GetAccountContext(gomock.Any(), "ac_sandbox").Return(&domain.AccountContext{AccountID: "ac_sandbox", IsSandbox: true, OwnerAccountID: &owner}, nil)
	f.storeIntegration(t, owner, true, nil)

	access, apiErr := f.svc.GetOpenRouterAccess(agentCtx("ac_sandbox"))

	require.Nil(t, apiErr)
	assert.True(t, access.Configured)
	assert.Equal(t, "sk-or-v1-secret", access.APIKey)
	assert.False(t, access.LimitReached)
}

func TestGetOpenRouterAccess_Limit(t *testing.T) {
	limit := &domain.Money{Amount: "50", Currency: "usd"}
	for name, tc := range map[string]struct {
		spent       string
		wantReached bool
	}{
		"under the limit": {"49.99", false},
		"at the limit":    {"50", true},
		"over the limit":  {"50.01", true},
	} {
		t.Run(name, func(t *testing.T) {
			f := newOpenRouterFixture(t)
			f.accounts.EXPECT().GetAccountContext(gomock.Any(), "ac_main").Return(&domain.AccountContext{AccountID: "ac_main"}, nil)
			f.storeIntegration(t, "ac_main", true, limit)
			f.openRouter.EXPECT().GetKeyUsage(gomock.Any(), "sk-or-v1-secret").Return(&domain.OpenRouterKeyUsage{SpentThisMonth: tc.spent}, nil)

			access, apiErr := f.svc.GetOpenRouterAccess(agentCtx("ac_main"))

			require.Nil(t, apiErr)
			assert.Equal(t, tc.wantReached, access.LimitReached)
			require.NotNil(t, access.SpentThisMonth)
			assert.Equal(t, tc.spent, *access.SpentThisMonth)
		})
	}
}

// OpenRouter's own key limit still guards spend, so an unreadable spend must not stop every agent.
func TestGetOpenRouterAccess_UnreadableSpendLetsTheRunGo(t *testing.T) {
	f := newOpenRouterFixture(t)
	f.accounts.EXPECT().GetAccountContext(gomock.Any(), "ac_main").Return(&domain.AccountContext{AccountID: "ac_main"}, nil)
	f.storeIntegration(t, "ac_main", true, &domain.Money{Amount: "50", Currency: "usd"})
	f.openRouter.EXPECT().GetKeyUsage(gomock.Any(), gomock.Any()).Return(nil, apierror.NewInternalError(nil, "down"))

	access, apiErr := f.svc.GetOpenRouterAccess(agentCtx("ac_main"))

	require.Nil(t, apiErr)
	assert.False(t, access.LimitReached)
	assert.Nil(t, access.SpentThisMonth)
}

func TestGetOpenRouterAccess_InactiveSkipsTheSpendCheck(t *testing.T) {
	f := newOpenRouterFixture(t)
	f.accounts.EXPECT().GetAccountContext(gomock.Any(), "ac_main").Return(&domain.AccountContext{AccountID: "ac_main"}, nil)
	f.storeIntegration(t, "ac_main", false, &domain.Money{Amount: "50", Currency: "usd"})

	access, apiErr := f.svc.GetOpenRouterAccess(agentCtx("ac_main"))

	require.Nil(t, apiErr)
	assert.True(t, access.Configured)
	assert.False(t, access.IsActive)
}

func TestGetOpenRouterAccess_RefusesUsers(t *testing.T) {
	f := newOpenRouterFixture(t)
	accountID := "ac_main"
	ctx := appctx.WithIdentity(context.Background(), &types.Identity{
		Type:   types.IdentityActorTypeUser,
		Target: &types.IdentityTarget{AccountID: accountID},
		Actor:  &types.IdentityActor{RelationType: types.IdentityRelationTypeInternal, ID: "usr_1", AccountID: &accountID},
	})

	_, apiErr := f.svc.GetOpenRouterAccess(ctx)

	assert.NotNil(t, apiErr)
}
