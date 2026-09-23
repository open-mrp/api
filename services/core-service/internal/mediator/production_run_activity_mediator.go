package mediator

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/appctx"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/contracts"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/messaging"
	"github.com/open-mrp/api/shared/ptrutil"
	"github.com/open-mrp/api/shared/tracing"
)

var productionRunActivityMedTracer = tracing.GetTracer("core-service.production_run_activity_mediator")

type productionRunActivityMedImpl struct {
	repos domain.RepoFactory
}

type ProductionRunActivityMedConfig struct {
	// Repos (required) is the repository factory; alerts join its transaction through the outbox.
	Repos domain.RepoFactory
}

func (c *ProductionRunActivityMedConfig) validate() error {
	if c.Repos == nil {
		return fmt.Errorf("production run activity mediator: repos is required")
	}
	return nil
}

func NewProductionRunActivityMed(config *ProductionRunActivityMedConfig) domain.ProductionRunActivityMed {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &productionRunActivityMedImpl{
		repos: config.Repos,
	}
}

// NotifyBatchesAdded alerts the run's responsible user that someone else added batches to it.
//
//  1. Resolve the responsible user to an active account user; no-op when there is none.
//  2. No-op when the actor is the responsible user.
//  3. Enqueue a bell alert linking to the run, folded into one rolling row per run per day.
func (m *productionRunActivityMedImpl) NotifyBatchesAdded(ctx context.Context, identity *types.Identity, run *domain.ProductionRun, count int) *apierror.APIError {
	ctx, span := productionRunActivityMedTracer.Start(ctx, "mediator.production_run_activity.notify_batches_added")
	defer span.End()

	action := fmt.Sprintf("added %d batches", count)
	if count == 1 {
		action = "added a batch"
	}

	return tracing.Trace(span, m.notify(ctx, identity, run, fmt.Sprintf("Production run %s updated", run.Number), action, true))
}

// NotifyBatchDeleted alerts the responsible user of the batch's run that someone else deleted it. No-op for a batch not on a run.
//
//  1. Load the batch's run.
//  2. Resolve the responsible user to an active account user; no-op when there is none.
//  3. No-op when the actor is the responsible user.
//  4. Enqueue a bell alert linking to the run, folded into one rolling row per run per day.
func (m *productionRunActivityMedImpl) NotifyBatchDeleted(ctx context.Context, identity *types.Identity, accountID string, batch *domain.Batch) *apierror.APIError {
	ctx, span := productionRunActivityMedTracer.Start(ctx, "mediator.production_run_activity.notify_batch_deleted")
	defer span.End()

	if batch.ProductionRun == nil {
		return nil
	}

	run, apiErr := m.repos.NewProductionRunRepo().Get(ctx, domain.GetProductionRunParams{
		ProductionRunID: batch.ProductionRun.ID,
		AccountID:       accountID,
	})
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	action := "deleted a batch"
	if batch.Item.SKU != "" {
		action = "deleted a " + batch.Item.SKU + " batch"
	}

	return tracing.Trace(span, m.notify(ctx, identity, run, fmt.Sprintf("Production run %s updated", run.Number), action, true))
}

// NotifyRunDeleted alerts the run's responsible user that someone else deleted the run and its batches.
//
//  1. Resolve the responsible user to an active account user; no-op when there is none.
//  2. No-op when the actor is the responsible user.
//  3. Enqueue a standalone bell alert with no link.
func (m *productionRunActivityMedImpl) NotifyRunDeleted(ctx context.Context, identity *types.Identity, run *domain.ProductionRun) *apierror.APIError {
	ctx, span := productionRunActivityMedTracer.Start(ctx, "mediator.production_run_activity.notify_run_deleted")
	defer span.End()

	var action string
	switch run.BatchCount {
	case 0:
		action = "deleted the production run"
	case 1:
		action = "deleted the production run and its batch"
	default:
		action = fmt.Sprintf("deleted the production run and its %d batches", run.BatchCount)
	}

	return tracing.Trace(span, m.notify(ctx, identity, run, fmt.Sprintf("Production run %s deleted", run.Number), action, false))
}

func (m *productionRunActivityMedImpl) notify(ctx context.Context, identity *types.Identity, run *domain.ProductionRun, title, action string, linkToRun bool) *apierror.APIError {
	accountUserRepo := m.repos.NewAccountUserRepo()

	recipientID, apiErr := accountUserRepo.ResolveAccountUserID(ctx, run.AccountID, run.ResponsibleUserID)
	if apierror.IsNotFound(apiErr) {
		return nil
	}
	if apiErr != nil {
		return apiErr
	}

	if identity.HasUserActor() {
		actorID, apiErr := accountUserRepo.ResolveAccountUserID(ctx, run.AccountID, identity.Actor.ID)
		if apiErr != nil && !apierror.IsNotFound(apiErr) {
			return apiErr
		}
		if actorID == recipientID {
			return nil
		}
	}

	actorName := "Someone"
	if identity.Actor != nil && ptrutil.Deref(identity.Actor.Name) != "" {
		actorName = *identity.Actor.Name
	}

	data := messaging.AlertFanoutData{
		AccountID:               run.AccountID,
		Category:                string(constants.NotificationCategoryProductionRunUpdated),
		Kind:                    "alert",
		Title:                   title,
		Body:                    actorName + " " + action + ".",
		Priority:                string(constants.NotificationPriorityNormal),
		SenderType:              string(constants.NotificationSenderTypeSystem),
		RecipientAccountUserIDs: []string{recipientID},
	}
	if linkToRun {
		data.LinkResourceType = string(constants.ObjectTypeProductionRun)
		data.LinkResourceID = run.ID
		data.DedupeKey = "runact_" + run.ID + "_" + time.Now().UTC().Format("20060102")
	}

	dataJSON, err := json.Marshal(data)
	if err != nil {
		return apierror.NewInternalError(err, "Failed to marshal production run activity fan-out payload.")
	}

	msg := contracts.AmqpMessage{Data: dataJSON, Identity: identity}
	if requestID, ok := appctx.GetRequestID(ctx); ok {
		msg.RequestID = requestID
	}

	if _, err := m.repos.NewOutboxRepo().Create(ctx, messaging.OutboxMessageInput{
		ServiceName: domain.ServiceName,
		MessageType: string(contracts.NotificationCmdFanout),
		Destination: messaging.ApplicationExchange,
		RoutingKey:  string(contracts.NotificationCmdFanout),
		Payload:     msg,
	}); err != nil {
		return apierror.NewInternalError(err, "Failed to enqueue production run activity fan-out.")
	}

	return nil
}
