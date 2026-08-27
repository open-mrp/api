package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/appctx"
	"github.com/open-mrp/api/shared/audit"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/contracts"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/id"
	"github.com/open-mrp/api/shared/idempotency"
	"github.com/open-mrp/api/shared/messaging"
	"github.com/open-mrp/api/shared/tracing"
)

var pickSvcTracer = tracing.GetTracer("core-service.pick_service")

type pickSvcImpl struct {
	repos           domain.RepoFactory
	mediatorFactory domain.MediatorFactory
	jobSvcFactory   domain.JobSvcFactory
	txManager       TransactionManager
}

type PickSvcConfig struct {
	// Repos (required) is the repository factory.
	Repos domain.RepoFactory

	// MediatorFactory (required) builds the mediators used by this service.
	MediatorFactory domain.MediatorFactory

	// (required) Builds the job service the async pack records on.
	JobSvcFactory domain.JobSvcFactory

	// TxManager (required) wraps multi-step operations in database transactions.
	TxManager TransactionManager
}

func (c *PickSvcConfig) validate() error {
	if c.Repos == nil {
		return fmt.Errorf("pick service: repos is required")
	}
	if c.MediatorFactory == nil {
		return fmt.Errorf("pick service: mediator factory is required")
	}
	if c.JobSvcFactory == nil {
		return fmt.Errorf("pick service: job service factory is required")
	}
	if c.TxManager == nil {
		return fmt.Errorf("pick service: tx manager is required")
	}
	return nil
}

func NewPickSvc(config *PickSvcConfig) domain.PickSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &pickSvcImpl{
		repos:           config.Repos,
		mediatorFactory: config.MediatorFactory,
		jobSvcFactory:   config.JobSvcFactory,
		txManager:       config.TxManager,
	}
}

func (s *pickSvcImpl) mediators() domain.Mediators {
	return s.mediatorFactory.Build(s.repos)
}

func (s *pickSvcImpl) withTx(ctx context.Context, fn func(context.Context, *pickSvcImpl) *apierror.APIError) *apierror.APIError {
	return s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		txSvc := &pickSvcImpl{
			repos:           f,
			mediatorFactory: s.mediatorFactory,
			jobSvcFactory:   s.jobSvcFactory,
			txManager:       s.txManager,
		}
		return fn(txCtx, txSvc)
	})
}

func (s *pickSvcImpl) ListPicks(ctx context.Context, params domain.ListPicksParams) (*domain.ListPicksResult, *apierror.APIError) {
	ctx, span := pickSvcTracer.Start(ctx, "service.pick.list")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsAssignedActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := checkPickReadPermission(identity); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if !identity.IsTargetAccountSet() {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("The OpenMRP-Account-ID header is required."))
	}

	if identity.IsExternalTarget() {
		meds := s.mediators()
		if apiErr := meds.ReadAccess.CheckReadAccess(ctx, *identity.ActorAccountID(), identity.Target.AccountID); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}

	params.AccountID = identity.Target.AccountID

	repo := s.repos.NewPickRepo()
	result, apiErr := repo.List(ctx, params)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	// Lines are the one heavy expansion, so the list pays for them only on request.
	if includesPickLines(params.Includes) {
		for _, pick := range result.Picks {
			lines, apiErr := repo.GetLines(ctx, pick.ID)
			if apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			pick.Lines = lines
		}
	}

	if includesPickShipments(params.Includes) {
		for _, pick := range result.Picks {
			ids, apiErr := repo.GetShipmentIDs(ctx, params.AccountID, pick.ID)
			if apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			pick.ShipmentIDs = ids
		}
	}

	return result, nil
}

func (s *pickSvcImpl) GetPick(ctx context.Context, pickID string, includes []string) (*domain.Pick, *apierror.APIError) {
	ctx, span := pickSvcTracer.Start(ctx, "service.pick.get")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainPicks, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	accountID := identity.Target.AccountID

	repo := s.repos.NewPickRepo()

	pick, apiErr := repo.Get(ctx, accountID, pickID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if includesPickLines(includes) {
		lines, apiErr := repo.GetLines(ctx, pickID)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		pick.Lines = lines
	}

	if includesPickShipments(includes) {
		ids, apiErr := repo.GetShipmentIDs(ctx, accountID, pickID)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		pick.ShipmentIDs = ids
	}

	return pick, nil
}

func (s *pickSvcImpl) PickAllLines(ctx context.Context, pickID string) (*domain.Pick, *apierror.APIError) {
	ctx, span := pickSvcTracer.Start(ctx, "service.pick.pick_all_lines")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainPicks, types.ActionUpdate); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	accountID := identity.Target.AccountID

	var result *domain.Pick
	apiErr := s.withTx(ctx, func(txCtx context.Context, txSvc *pickSvcImpl) *apierror.APIError {
		txRepo := txSvc.repos.NewPickRepo()

		old, apiErr := txRepo.Get(txCtx, accountID, pickID)
		if apiErr != nil {
			return apiErr
		}

		if apiErr := txRepo.PickAllLines(txCtx, pickID); apiErr != nil {
			return apiErr
		}

		pick, apiErr := txRepo.Get(txCtx, accountID, pickID)
		if apiErr != nil {
			return apiErr
		}

		lines, apiErr := txRepo.GetLines(txCtx, pickID)
		if apiErr != nil {
			return apiErr
		}
		pick.Lines = lines

		result = pick

		changes := audit.ComputeChanges(old, result)

		if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
			ServiceName:      domain.ServiceName,
			Action:           constants.AuditActionUpdate,
			ResourceType:     constants.ObjectTypePick,
			ResourceID:       result.ID,
			RootResourceType: constants.ObjectTypeSalesOrder,
			RootResourceID:   result.SalesOrderID,
			Changes:          changes,
		}); apiErr != nil {
			return apiErr
		}

		return nil
	})

	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return result, nil
}

func (s *pickSvcImpl) VoidPick(ctx context.Context, pickID string) (*domain.Pick, *apierror.APIError) {
	ctx, span := pickSvcTracer.Start(ctx, "service.pick.void")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainPicks, types.ActionUpdate); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	accountID := identity.Target.AccountID

	repo := s.repos.NewPickRepo()

	hasShipped, apiErr := repo.HasShippedItems(ctx, accountID, pickID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if hasShipped {
		return nil, tracing.Trace(span, apierror.NewValidationError("Cannot void a pick with shipped items."))
	}

	old, apiErr := repo.Get(ctx, accountID, pickID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	var result *domain.Pick
	apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *pickSvcImpl) *apierror.APIError {
		txRepo := txSvc.repos.NewPickRepo()

		if apiErr := txRepo.VoidAllLines(txCtx, pickID); apiErr != nil {
			return apiErr
		}

		if apiErr := txRepo.DeleteDuplicatePickLines(txCtx, accountID, pickID); apiErr != nil {
			return apiErr
		}

		if apiErr := txRepo.ClearFinishedAt(txCtx, accountID, pickID); apiErr != nil {
			return apiErr
		}

		pick, apiErr := txRepo.Get(txCtx, accountID, pickID)
		if apiErr != nil {
			return apiErr
		}

		lines, apiErr := txRepo.GetLines(txCtx, pickID)
		if apiErr != nil {
			return apiErr
		}
		pick.Lines = lines

		result = pick

		changes := audit.ComputeChanges(old, result)

		if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
			ServiceName:      domain.ServiceName,
			Action:           constants.AuditActionUpdate,
			ResourceType:     constants.ObjectTypePick,
			ResourceID:       result.ID,
			RootResourceType: constants.ObjectTypeSalesOrder,
			RootResourceID:   result.SalesOrderID,
			Changes:          changes,
		}); apiErr != nil {
			return apiErr
		}

		return nil
	})

	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return result, nil
}

// --- Pack: accept phase ---

// Accepts a pack: authorizes it, checks synchronously that there is something to pack, and
// records the work on a job. The shipment is created by ExecutePackPick.
func (s *pickSvcImpl) PackPick(ctx context.Context, pickID string, shipmentCaseCount int32) (*domain.Job, *apierror.APIError) {
	ctx, span := pickSvcTracer.Start(ctx, "service.pick.pack.enqueue")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}
	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainPicks, types.ActionUpdate); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if !identity.IsTargetAccountSet() {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("The OpenMRP-Account-ID header is required."))
	}
	accountID := identity.Target.AccountID

	meds := s.mediators()
	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, identity)
	if apiErr != nil {
		return nil, apiErr
	}

	switch domain.RecoveryPoint(idempotencyKey.RecoveryPoint) {
	case domain.RecoveryPointFinished:
		cached, err := idempotency.UnmarshalCachedResponse[domain.Job](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		// Reads the database, so it runs after the key is claimed. A pick nobody can pack is a
		// bad request, and the caller has to learn that now rather than from a failed job.
		repo := s.repos.NewPickRepo()
		if _, apiErr := repo.Get(ctx, accountID, pickID); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		linesToPack, apiErr := repo.FindLinesToPack(ctx, pickID)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		if len(linesToPack) == 0 {
			return nil, tracing.Trace(span, apierror.NewValidationError("No lines to pack."))
		}

		jobItems, err := json.Marshal(domain.PackPickJob{PickID: pickID, ShipmentCaseCount: shipmentCaseCount})
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Failed to marshal pack payload."))
		}

		createdByID := jobCreatedByID(ctx, s.repos, accountID, identity)

		var raisedJob *domain.Job
		apiErr = s.txManager.WithTx(ctx, func(txCtx context.Context, txRepos domain.RepoFactory) *apierror.APIError {
			job, apiErr := s.jobSvcFactory.Build(txRepos).CreateJob(txCtx, domain.CreateJobServiceParams{
				JobItems:     jobItems,
				Type:         constants.JobTypePackPick,
				ResourceType: constants.ObjectTypeShipment,
				CreatedByID:  createdByID,
			})
			if apiErr != nil {
				return apiErr
			}
			raisedJob = job

			// Enqueued through the outbox inside the transaction, so the command is published
			// if and only if the job commits.
			payloadJSON, err := json.Marshal(domain.BulkOperationJobEvent{JobID: job.ID})
			if err != nil {
				return apierror.NewInternalError(err, "Failed to marshal pack job event.")
			}
			msg := contracts.AmqpMessage{Data: payloadJSON, Identity: identity}
			if requestID, ok := appctx.GetRequestID(txCtx); ok {
				msg.RequestID = requestID
			}
			if _, err := txRepos.NewOutboxRepo().Create(txCtx, messaging.OutboxMessageInput{
				ServiceName: domain.ServiceName,
				MessageType: string(messaging.PackPick.RoutingKey()),
				Destination: messaging.ApplicationExchange,
				RoutingKey:  string(messaging.PackPick.RoutingKey()),
				Payload:     msg,
			}); err != nil {
				return apierror.NewInternalError(err, "Failed to create pack outbox message.")
			}

			return s.mediatorFactory.Build(txRepos).Idempotency.CacheSuccessResponse(txCtx, idempotencyKey.TypeID, raisedJob)
		})
		if apiErr != nil {
			return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, apiErr)
		}

		return raisedJob, nil

	default:
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Unexpected recovery point: "+idempotencyKey.RecoveryPoint))
	}
}

// --- Pack: execute phase ---

// Runs an accepted pack and settles the job. The inbox de-dupes redeliveries; the terminal
// guard covers a replay that outlives its record.
func (s *pickSvcImpl) ExecutePackPick(ctx context.Context, event domain.BulkOperationJobEvent) *apierror.APIError {
	ctx, span := pickSvcTracer.Start(ctx, "service.pick.pack.execute")
	defer span.End()

	if event.JobID == "" {
		return tracing.Trace(span, apierror.NewValidationError("Pack job event is missing a job."))
	}

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}
	accountID := identity.Target.AccountID

	jobs := s.jobSvcFactory.Build(s.repos)
	job, apiErr := jobs.GetJobForExecution(ctx, event.JobID)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if job.IsTerminal() {
		return nil
	}

	var payload domain.PackPickJob
	if err := json.Unmarshal(job.JobItems, &payload); err != nil {
		apiErr := apierror.NewInternalError(err, "Job items are not a pack payload.")
		jobs.FailJob(ctx, domain.FailJobParams{JobID: job.ID, ApiErr: apiErr})
		return tracing.Trace(span, apiErr)
	}

	if _, apiErr := jobs.StartJob(ctx, domain.StartJobParams{JobID: job.ID}); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *pickSvcImpl) *apierror.APIError {
		var outcome packOutcome
		if apiErr := packPickInTx(txCtx, txSvc, accountID, payload.PickID, payload.ShipmentCaseCount, &outcome); apiErr != nil {
			return apiErr
		}

		// Settled in the same transaction as the writes, so a completed job always has a shipment.
		return s.jobSvcFactory.Build(txSvc.repos).CompleteJob(txCtx, domain.CompleteJobParams{
			JobID: job.ID,
			Results: []domain.RowResult{{
				Index:        0,
				Status:       constants.JobResultStatusCreated,
				ResourceType: constants.ObjectTypeShipment,
				ID:           outcome.ShipmentID,
				Name:         &outcome.ShipmentNumber,
				SubResources: outcome.SubResources,
			}},
		})
	})
	if apiErr != nil {
		jobs.FailJob(ctx, domain.FailJobParams{JobID: job.ID, ApiErr: apiErr})
		return tracing.Trace(span, apiErr)
	}

	return nil
}

// Records what one pack created, so the job it ran for can report it.
type packOutcome struct {
	ShipmentID     string
	ShipmentNumber string
	SubResources   []domain.SubResourceRef
}

// Runs a pack inside the caller's transaction: marks the picked lines packed, opens remainder
// lines for whatever is still outstanding, and creates the shipment with its lines and cases.
func packPickInTx(txCtx context.Context, txSvc *pickSvcImpl, accountID, pickID string, shipmentCaseCount int32, out *packOutcome) *apierror.APIError {
	txRepo := txSvc.repos.NewPickRepo()
	pickLineRepo := txSvc.repos.NewPickLineRepo()
	var subResources []domain.SubResourceRef

	// Find lines eligible for packing
	linesToPack, apiErr := txRepo.FindLinesToPack(txCtx, pickID)
	if apiErr != nil {
		return apiErr
	}
	if len(linesToPack) == 0 {
		return apierror.NewValidationError("No lines to pack.")
	}

	// Mark lines as packed
	if apiErr := txRepo.PackLines(txCtx, pickID); apiErr != nil {
		return apiErr
	}

	// For each packed line's order line (deduplicated), calculate remaining and create new pick lines if needed
	var remainingPickLineIDs []string
	processedOrderLines := make(map[string]bool)
	for _, line := range linesToPack {
		if processedOrderLines[line.SalesOrderLineID] {
			continue
		}
		processedOrderLines[line.SalesOrderLineID] = true

		remainingValue, unitID, apiErr := pickLineRepo.CalculateRemainingForOrderLine(txCtx, line.SalesOrderLineID)
		if apiErr != nil {
			return apiErr
		}

		remainingFloat, _ := strconv.ParseFloat(remainingValue, 64)
		if remainingFloat > 0 {
			// Only create a remaining pick line if there isn't already an unpacked one for this order line
			hasUnpacked, apiErr := pickLineRepo.HasUnpackedPickLineForOrderLine(txCtx, line.SalesOrderLineID)
			if apiErr != nil {
				return apiErr
			}
			if hasUnpacked {
				continue
			}

			newPickLineID, apiErr := id.GenID(id.PickLineIDPrefix, nil)
			if apiErr != nil {
				return apiErr
			}
			newQuantityID, apiErr := id.GenID(id.QuantityIDPrefix, nil)
			if apiErr != nil {
				return apiErr
			}
			// Create with quantity 0 — remaining lines are placeholders that need explicit quantity assignment
			if apiErr := txRepo.CreateQuantity(txCtx, newQuantityID, "0", unitID); apiErr != nil {
				return apiErr
			}
			if apiErr := pickLineRepo.CreateForRemaining(txCtx, newPickLineID, newQuantityID, pickID, line.SalesOrderLineID); apiErr != nil {
				return apiErr
			}
			remainingPickLineIDs = append(remainingPickLineIDs, newPickLineID)
		}
	}

	// Get sales order info for shipment creation
	salesOrder, apiErr := txRepo.GetSalesOrderForPick(txCtx, accountID, pickID)
	if apiErr != nil {
		return apiErr
	}

	// Deferred to here because the pick lines are synthesized before the order is in hand, and
	// the root stamp is what puts them in that order's history.
	for _, lineID := range remainingPickLineIDs {
		if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
			ServiceName:      domain.ServiceName,
			Action:           constants.AuditActionCreate,
			ResourceType:     constants.ObjectTypePickLine,
			ResourceID:       lineID,
			RootResourceType: constants.ObjectTypeSalesOrder,
			RootResourceID:   salesOrder.ID,
		}); apiErr != nil {
			return apiErr
		}
	}

	// Count existing shipments to determine shipment number
	count, apiErr := txRepo.CountShipmentsByOrder(txCtx, salesOrder.ID)
	if apiErr != nil {
		return apiErr
	}

	var shipmentNumber string
	if count == 0 {
		shipmentNumber = salesOrder.Number
	} else {
		shipmentNumber = fmt.Sprintf("%s-%d", salesOrder.Number, count+1)
	}

	// Generate shipment ID
	shipmentID, apiErr := id.GenID(id.ShipmentIDPrefix, nil)
	if apiErr != nil {
		return apiErr
	}

	// Create shipment
	if apiErr := txRepo.CreateShipment(txCtx, domain.CreateShipmentFromPickParams{
		ID:                shipmentID,
		Number:            shipmentNumber,
		SalesOrderID:      salesOrder.ID,
		CarrierID:         salesOrder.CarrierID,
		ServiceLevelID:    salesOrder.ServiceLevelID,
		ShippingAddressID: salesOrder.ShippingAddressID,
		StatusCode:        "packed",
		AccountID:         accountID,
	}); apiErr != nil {
		return apiErr
	}

	// Packing is where the shipment, its lines and its cases come into existence; without their
	// own create events the order's history shows only "pick updated".
	if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
		ServiceName:      domain.ServiceName,
		Action:           constants.AuditActionCreate,
		ResourceType:     constants.ObjectTypeShipment,
		ResourceID:       shipmentID,
		RootResourceType: constants.ObjectTypeSalesOrder,
		RootResourceID:   salesOrder.ID,
	}); apiErr != nil {
		return apiErr
	}

	// Create shipment lines for each packed line
	for _, line := range linesToPack {
		shipmentLineID, apiErr := id.GenID(id.ShipmentLineIDPrefix, nil)
		if apiErr != nil {
			return apiErr
		}
		shipmentLineQtyID, apiErr := id.GenID(id.QuantityIDPrefix, nil)
		if apiErr != nil {
			return apiErr
		}
		if apiErr := txRepo.CreateQuantity(txCtx, shipmentLineQtyID, line.QuantityValue, line.QuantityUnitID); apiErr != nil {
			return apiErr
		}
		if apiErr := txRepo.CreateShipmentLine(txCtx, domain.CreateShipmentLineParams{
			ID:               shipmentLineID,
			ShipmentID:       shipmentID,
			SalesOrderLineID: line.SalesOrderLineID,
			QuantityID:       shipmentLineQtyID,
		}); apiErr != nil {
			return apiErr
		}

		if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
			ServiceName:      domain.ServiceName,
			Action:           constants.AuditActionCreate,
			ResourceType:     constants.ObjectTypeShipmentLine,
			ResourceID:       shipmentLineID,
			RootResourceType: constants.ObjectTypeSalesOrder,
			RootResourceID:   salesOrder.ID,
		}); apiErr != nil {
			return apiErr
		}

		subResources = append(subResources, domain.SubResourceRef{
			ResourceType: constants.ObjectTypeShipmentLine,
			ID:           shipmentLineID,
		})
	}

	// Both unit ids are resolved rather than written in: a stranded unit ref drops the case from
	// the shipping_cases expansion (which inner-joins the unit) while case_count still counts it.
	unitRepo := txSvc.repos.NewUnitRepo()
	freightAmountUnitID, apiErr := unitRepo.GetCurrencyBaseUnitID(txCtx)
	if apiErr != nil {
		return apiErr
	}
	freightWeightUnitID, apiErr := unitRepo.GetFreightWeightUnitID(txCtx)
	if apiErr != nil {
		return apiErr
	}

	// Create shipping cases
	for i := 0; i < int(shipmentCaseCount); i++ {
		caseID, apiErr := id.GenID(id.ShippingCaseIDPrefix, nil)
		if apiErr != nil {
			return apiErr
		}
		caseNumber := fmt.Sprintf("%s-%d", shipmentNumber, i+1)

		freightAmountID, apiErr := id.GenID(id.QuantityIDPrefix, nil)
		if apiErr != nil {
			return apiErr
		}
		freightWeightID, apiErr := id.GenID(id.QuantityIDPrefix, nil)
		if apiErr != nil {
			return apiErr
		}

		if apiErr := txRepo.CreateQuantity(txCtx, freightAmountID, "0", freightAmountUnitID); apiErr != nil {
			return apiErr
		}
		if apiErr := txRepo.CreateQuantity(txCtx, freightWeightID, "0", freightWeightUnitID); apiErr != nil {
			return apiErr
		}

		if apiErr := txRepo.CreateShippingCase(txCtx, domain.CreateShippingCaseParams{
			ID:              caseID,
			Number:          caseNumber,
			FreightAmountID: freightAmountID,
			FreightWeightID: freightWeightID,
			ShipmentID:      shipmentID,
			CarrierID:       salesOrder.CarrierID,
			AccountID:       accountID,
		}); apiErr != nil {
			return apiErr
		}

		if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
			ServiceName:      domain.ServiceName,
			Action:           constants.AuditActionCreate,
			ResourceType:     constants.ObjectTypeShippingCase,
			ResourceID:       caseID,
			RootResourceType: constants.ObjectTypeSalesOrder,
			RootResourceID:   salesOrder.ID,
		}); apiErr != nil {
			return apiErr
		}

		subResources = append(subResources, domain.SubResourceRef{
			ResourceType: constants.ObjectTypeShippingCase,
			ID:           caseID,
			Name:         &caseNumber,
		})
	}

	// Mark pick as finished if all lines are now packed
	if apiErr := txRepo.MarkFinishedIfAllPacked(txCtx, pickID); apiErr != nil {
		return apiErr
	}

	pick, apiErr := txRepo.Get(txCtx, accountID, pickID)
	if apiErr != nil {
		return apiErr
	}

	if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
		ServiceName:      domain.ServiceName,
		Action:           constants.AuditActionUpdate,
		ResourceType:     constants.ObjectTypePick,
		ResourceID:       pick.ID,
		RootResourceType: constants.ObjectTypeSalesOrder,
		RootResourceID:   pick.SalesOrderID,
		Metadata:         map[string]any{"packed": true},
	}); apiErr != nil {
		return apiErr
	}

	*out = packOutcome{
		ShipmentID:     shipmentID,
		ShipmentNumber: shipmentNumber,
		SubResources:   subResources,
	}
	return nil
}

// checkPickReadPermission checks the appropriate read permission based on the identity context. Internal actors need picks:read for their own account, or customers:read / suppliers:read for external accounts.
func checkPickReadPermission(identity *types.Identity) *apierror.APIError {
	if !identity.IsInternalActor() {
		return nil
	}
	if identity.IsTargetCustomerAccount() {
		return identity.CheckHasPermission(types.PermissionDomainCustomers, types.ActionRead)
	}
	if identity.IsTargetSupplierAccount() {
		return identity.CheckHasPermission(types.PermissionDomainSuppliers, types.ActionRead)
	}
	return identity.CheckHasPermission(types.PermissionDomainPicks, types.ActionRead)
}

// Reports whether the caller asked for the pick's related shipments, which cost their own query.
func includesPickShipments(includes []string) bool {
	for _, inc := range includes {
		if inc == "related.shipments" {
			return true
		}
	}
	return false
}

func includesPickLines(includes []string) bool {
	for _, inc := range includes {
		if inc == "lines" || strings.HasPrefix(inc, "lines.") {
			return true
		}
	}
	return false
}
