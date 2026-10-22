package service

import (
	"context"
	"errors"
	"testing"

	"github.com/open-mrp/api/services/agent-service/internal/domain"
	clientmock "github.com/open-mrp/api/services/agent-service/internal/domain/mock/client"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func runnerWithOpenRouter(t *testing.T, access *domain.OpenRouterAccess, err error) *runnerSvc {
	core := clientmock.NewMockCoreClient(gomock.NewController(t))
	core.EXPECT().GetOpenRouterAccess(gomock.Any()).Return(access, err)
	return &runnerSvc{coreClient: core}
}

func TestWithOpenRouterAccess_NoIntegrationKeepsTheDeploymentKey(t *testing.T) {
	ctx := context.Background()
	s := runnerWithOpenRouter(t, &domain.OpenRouterAccess{}, nil)

	got, refusal, err := s.withOpenRouterAccess(ctx)

	require.NoError(t, err)
	assert.Empty(t, refusal)
	assert.Equal(t, ctx, got)
}

func TestWithOpenRouterAccess_ActiveIntegrationCarriesItsKey(t *testing.T) {
	s := runnerWithOpenRouter(t, &domain.OpenRouterAccess{Configured: true, IsActive: true, APIKey: "sk-or-v1-account"}, nil)

	got, refusal, err := s.withOpenRouterAccess(context.Background())

	require.NoError(t, err)
	assert.Empty(t, refusal)
	assert.NotEqual(t, context.Background(), got)
}

func TestWithOpenRouterAccess_InactiveIntegrationRefuses(t *testing.T) {
	s := runnerWithOpenRouter(t, &domain.OpenRouterAccess{Configured: true, APIKey: "sk-or-v1-account"}, nil)

	_, refusal, err := s.withOpenRouterAccess(context.Background())

	require.NoError(t, err)
	assert.Contains(t, refusal, "inactive")
}

func TestWithOpenRouterAccess_SpentLimitRefuses(t *testing.T) {
	limit, spent := "50", "50.12"
	s := runnerWithOpenRouter(t, &domain.OpenRouterAccess{Configured: true, IsActive: true, APIKey: "sk-or-v1-account", MonthlySpendLimit: &limit, SpentThisMonth: &spent, LimitReached: true}, nil)

	_, refusal, err := s.withOpenRouterAccess(context.Background())

	require.NoError(t, err)
	assert.Contains(t, refusal, "$50.12")
	assert.Contains(t, refusal, "$50 monthly limit")
}

func TestWithOpenRouterAccess_LookupFailureFailsTheRun(t *testing.T) {
	s := runnerWithOpenRouter(t, nil, errors.New("core unavailable"))

	_, _, err := s.withOpenRouterAccess(context.Background())

	assert.Error(t, err)
}
