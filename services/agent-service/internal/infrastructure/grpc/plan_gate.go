package grpc

import (
	"context"

	"github.com/augno/api/services/agent-service/internal/domain"
	"github.com/augno/api/shared/constants"
)

type PlanGateAdapter struct {
	coreClient domain.CoreClient
}

func NewPlanGateAdapter(coreClient domain.CoreClient) *PlanGateAdapter {
	return &PlanGateAdapter{coreClient: coreClient}
}

func (a *PlanGateAdapter) CanUseAgents(ctx context.Context, accountID string) (bool, error) {
	acctCtx, err := a.coreClient.GetAccountContext(ctx, accountID)
	if err != nil {
		return false, err
	}

	if acctCtx.PlanCode == string(constants.PlanCodeFree) {
		return false, nil
	}

	return true, nil
}
