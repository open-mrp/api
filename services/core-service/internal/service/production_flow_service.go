package service

import (
	"context"
	"fmt"

	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/shared/appctx"
	"github.com/augno/api/shared/audit"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/idempotency"
	"github.com/augno/api/shared/tracing"
)

var productionFlowSvcTracer = tracing.GetTracer("core-service.production_flow_service")

type productionFlowSvcImpl struct {
	repos           domain.RepoFactory
	mediatorFactory domain.MediatorFactory
	txManager       TransactionManager
}

type ProductionFlowSvcConfig struct {
	Repos           domain.RepoFactory
	MediatorFactory domain.MediatorFactory
	TxManager       TransactionManager
}

func (c *ProductionFlowSvcConfig) validate() error {
	if c.Repos == nil {
		return fmt.Errorf("production flow service: repos is required")
	}
	if c.MediatorFactory == nil {
		return fmt.Errorf("production flow service: mediator factory is required")
	}
	if c.TxManager == nil {
		return fmt.Errorf("production flow service: tx manager is required")
	}
	return nil
}

func NewProductionFlowSvc(config *ProductionFlowSvcConfig) domain.ProductionFlowSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}
	return &productionFlowSvcImpl{
		repos:           config.Repos,
		mediatorFactory: config.MediatorFactory,
		txManager:       config.TxManager,
	}
}

func (s *productionFlowSvcImpl) mediators() domain.Mediators {
	return s.mediatorFactory.Build(s.repos)
}

func (s *productionFlowSvcImpl) withTx(ctx context.Context, fn func(context.Context, *productionFlowSvcImpl) *apierror.APIError) *apierror.APIError {
	if s.txManager == nil {
		panic("txManager is nil")
	}

	return s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		txSvc := &productionFlowSvcImpl{
			repos:           f,
			mediatorFactory: s.mediatorFactory,
			txManager:       s.txManager,
		}
		return fn(txCtx, txSvc)
	})
}

// GetProductionFlow returns the production flow graph for a given item.
//
// 1. Extract and validate the caller's identity, actor type, and production_steps:read permission.
// 2. Find the production step(s) that produce the given item.
// 3. Load the full edge graph for the account.
// 4. BFS backward from the initial step through parent edges, collecting relevant step IDs.
// 5. For each step, fetch full data including consumptions.
// 6. Compute in/out step IDs from the edge data.
func (s *productionFlowSvcImpl) GetProductionFlow(ctx context.Context, itemID string) ([]*domain.ProductionFlowStep, *apierror.APIError) {
	ctx, span := productionFlowSvcTracer.Start(ctx, "service.production_flow.get")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainProductionSteps, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	accountID := identity.Target.AccountID
	flowRepo := s.repos.NewProductionFlowRepo()

	// Find the step(s) that produce this item.
	initialStepIDs, apiErr := flowRepo.FindStepsByProducedItem(ctx, accountID, itemID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if len(initialStepIDs) == 0 {
		return []*domain.ProductionFlowStep{}, nil
	}

	// Get the full edge graph for the account.
	edges, apiErr := flowRepo.GetAllStepEdgesForAccount(ctx, accountID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	// Build adjacency map: childID → []parentIDs
	parentMap := make(map[string][]string)
	// Also build childMap: parentID → []childIDs for computing OutStepIDs
	childMap := make(map[string][]string)
	for _, edge := range edges {
		parentMap[edge.ChildStepID] = append(parentMap[edge.ChildStepID], edge.ParentStepID)
		childMap[edge.ParentStepID] = append(childMap[edge.ParentStepID], edge.ChildStepID)
	}

	// BFS backward from initial steps through parent edges.
	relevantStepIDs := make(map[string]bool)
	queue := make([]string, 0, len(initialStepIDs))
	queue = append(queue, initialStepIDs...)

	for len(queue) > 0 {
		stepID := queue[0]
		queue = queue[1:]

		if relevantStepIDs[stepID] {
			continue
		}
		relevantStepIDs[stepID] = true

		for _, parentID := range parentMap[stepID] {
			if !relevantStepIDs[parentID] {
				queue = append(queue, parentID)
			}
		}
	}

	// Fetch each relevant step's full data.
	stepQueryRepo := s.repos.NewProductionStepQueryRepo()
	steps := make([]*domain.ProductionFlowStep, 0, len(relevantStepIDs))

	for stepID := range relevantStepIDs {
		step, apiErr := flowRepo.GetFlowStep(ctx, accountID, stepID)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}

		// Fetch consumptions for this step.
		stepDetail, apiErr := stepQueryRepo.Find(ctx, accountID, stepID)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		step.Consumptions = stepDetail.Consumptions

		// Compute InStepIDs and OutStepIDs from edge data, filtered to relevant steps only.
		inIDs := make([]string, 0)
		for _, parentID := range parentMap[stepID] {
			if relevantStepIDs[parentID] {
				inIDs = append(inIDs, parentID)
			}
		}
		step.InStepIDs = inIDs

		outIDs := make([]string, 0)
		for _, childID := range childMap[stepID] {
			if relevantStepIDs[childID] {
				outIDs = append(outIDs, childID)
			}
		}
		step.OutStepIDs = outIDs

		steps = append(steps, step)
	}

	return steps, nil
}

// ConnectSteps links two production steps in the flow DAG.
//
// 1. Extract and validate the caller's identity, actor type, and production_steps:update permission.
// 2. Validate both steps belong to the account.
// 3. With idempotency, insert the connection.
func (s *productionFlowSvcImpl) ConnectSteps(ctx context.Context, sourceStepID, targetStepID string) *apierror.APIError {
	ctx, span := productionFlowSvcTracer.Start(ctx, "service.production_flow.connect_steps")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainProductionSteps, types.ActionUpdate); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	accountID := identity.Target.AccountID
	stepQueryRepo := s.repos.NewProductionStepQueryRepo()

	// Validate both steps are in the account.
	sourceInAccount, apiErr := stepQueryRepo.IsInAccount(ctx, accountID, sourceStepID)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if !sourceInAccount {
		return tracing.Trace(span, apierror.NewResourceNotFoundError("Source production step not found."))
	}

	targetInAccount, apiErr := stepQueryRepo.IsInAccount(ctx, accountID, targetStepID)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if !targetInAccount {
		return tracing.Trace(span, apierror.NewResourceNotFoundError("Target production step not found."))
	}

	meds := s.mediators()

	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, identity)
	if apiErr != nil {
		return apiErr
	}

	switch domain.RecoveryPoint(idempotencyKey.RecoveryPoint) {
	case domain.RecoveryPointFinished:
		cached, err := idempotency.UnmarshalCachedResponse[struct{}](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Error

	case domain.RecoveryPointStarted:
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *productionFlowSvcImpl) *apierror.APIError {
			if apiErr := txSvc.repos.NewProductionFlowRepo().ConnectStepsIdempotent(txCtx, sourceStepID, targetStepID); apiErr != nil {
				return apiErr
			}

			edge := &domain.StepEdge{ParentStepID: sourceStepID, ChildStepID: targetStepID}
			changes := audit.ComputeChanges(nil, edge)

			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionCreate,
				ResourceType: constants.ObjectTypeProductionFlow,
				ResourceID:   sourceStepID,
				Changes:      changes,
			}); apiErr != nil {
				return apiErr
			}

			return txSvc.mediators().Idempotency.CacheSuccessResponse(txCtx, idempotencyKey.TypeID, &struct{}{})
		})

		if apiErr != nil {
			return meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, apiErr)
		}

		return nil

	default:
		return tracing.Trace(span, apierror.NewInvariantViolationError("Unexpected recovery point: "+idempotencyKey.RecoveryPoint))
	}
}
