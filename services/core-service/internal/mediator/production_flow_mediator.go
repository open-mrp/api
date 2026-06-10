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

// LinkFlow recomputes all parent-child production step connections for a step
// based on its current consumptions and productions.
//
//  1. Delegate to the production flow repository to rebuild the step's connections.
func (m *productionFlowMedImpl) LinkFlow(ctx context.Context, productionStepID, accountID string) *apierror.APIError {
	ctx, span := productionFlowMedTracer.Start(ctx, "mediator.production_flow.link_flow")
	defer span.End()

	apiErr := m.repos.NewProductionFlowRepo().LinkFlow(ctx, productionStepID, accountID)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

// DisconnectSteps removes a specific parent-child connection between two steps.
//
//  1. Delegate to the production flow repository to remove the connection.
func (m *productionFlowMedImpl) DisconnectSteps(ctx context.Context, sourceID, targetID string) *apierror.APIError {
	ctx, span := productionFlowMedTracer.Start(ctx, "mediator.production_flow.disconnect_steps")
	defer span.End()

	apiErr := m.repos.NewProductionFlowRepo().DisconnectSteps(ctx, sourceID, targetID)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

// FindSourceStepsByConsumption returns IDs of parent steps that should be disconnected
// when a consumption is deleted.
//
//  1. Delegate to the production flow repository to find the matching parent step IDs.
func (m *productionFlowMedImpl) FindSourceStepsByConsumption(ctx context.Context, productionStepID, consumptionID, accountID string) ([]string, *apierror.APIError) {
	ctx, span := productionFlowMedTracer.Start(ctx, "mediator.production_flow.find_source_steps_by_consumption")
	defer span.End()

	result, apiErr := m.repos.NewProductionFlowRepo().FindSourceStepsByConsumption(ctx, productionStepID, consumptionID, accountID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return result, nil
}

// FindDownstreamStepByItem returns the ID of a downstream step connected via a specific
// consumed item, if one exists.
//
//  1. Delegate to the production flow repository to find the connected downstream step.
func (m *productionFlowMedImpl) FindDownstreamStepByItem(ctx context.Context, productionStepID, itemID, accountID string) (*string, *apierror.APIError) {
	ctx, span := productionFlowMedTracer.Start(ctx, "mediator.production_flow.find_downstream_step_by_item")
	defer span.End()

	result, apiErr := m.repos.NewProductionFlowRepo().FindDownstreamStepByItem(ctx, productionStepID, itemID, accountID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return result, nil
}
