package mediator

import (
	"context"

	"github.com/augno/api/services/core-service/internal/domain"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/tracing"
)

var productionFlowMedTracer = tracing.GetTracer("core-service.production_flow_mediator")

type productionFlowMedImpl struct {
	repos domain.RepoFactory
}

func NewProductionFlowMed(repos domain.RepoFactory) domain.ProductionFlowMed {
	return &productionFlowMedImpl{repos: repos}
}

func (m *productionFlowMedImpl) LinkFlow(ctx context.Context, productionStepID, accountID string) *apierror.APIError {
	ctx, span := productionFlowMedTracer.Start(ctx, "mediator.production_flow.link_flow")
	defer span.End()

	apiErr := m.repos.NewProductionFlowRepo().LinkFlow(ctx, productionStepID, accountID)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (m *productionFlowMedImpl) DisconnectSteps(ctx context.Context, sourceID, targetID string) *apierror.APIError {
	ctx, span := productionFlowMedTracer.Start(ctx, "mediator.production_flow.disconnect_steps")
	defer span.End()

	apiErr := m.repos.NewProductionFlowRepo().DisconnectSteps(ctx, sourceID, targetID)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (m *productionFlowMedImpl) FindSourceStepsByConsumption(ctx context.Context, productionStepID, consumptionID, accountID string) ([]string, *apierror.APIError) {
	ctx, span := productionFlowMedTracer.Start(ctx, "mediator.production_flow.find_source_steps_by_consumption")
	defer span.End()

	result, apiErr := m.repos.NewProductionFlowRepo().FindSourceStepsByConsumption(ctx, productionStepID, consumptionID, accountID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return result, nil
}

func (m *productionFlowMedImpl) FindDownstreamStepByItem(ctx context.Context, productionStepID, itemID, accountID string) (*string, *apierror.APIError) {
	ctx, span := productionFlowMedTracer.Start(ctx, "mediator.production_flow.find_downstream_step_by_item")
	defer span.End()

	result, apiErr := m.repos.NewProductionFlowRepo().FindDownstreamStepByItem(ctx, productionStepID, itemID, accountID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return result, nil
}
