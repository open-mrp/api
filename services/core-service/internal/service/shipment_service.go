package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/shopspring/decimal"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/event"
	"github.com/open-mrp/api/services/core-service/internal/mediator"
	"github.com/open-mrp/api/shared/appctx"
	"github.com/open-mrp/api/shared/audit"
	s3client "github.com/open-mrp/api/shared/cloud/s3"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/crypto"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/id"
	"github.com/open-mrp/api/shared/idempotency"
	"github.com/open-mrp/api/shared/messaging"
	"github.com/open-mrp/api/shared/ptrutil"
	"github.com/open-mrp/api/shared/textutil"
	"github.com/open-mrp/api/shared/timeutil"
	"github.com/open-mrp/api/shared/tracing"
)

// decryptShippoAPIKey decrypts and unwraps a stored Shippo integration credential blob, returning the plaintext API key to hand to the Shippo client factory. The credential is sealed with the account ID as additional authenticated data (see account_integration_service.go), so the same accountID must be supplied here.
func decryptShippoAPIKey(encryptedCreds string, encryptionKey []byte, accountID string) (string, *apierror.APIError) {
	plaintext, err := crypto.DecryptAESGCM(encryptedCreds, encryptionKey, []byte(accountID))
	if err != nil {
		return "", apierror.NewInternalError(err, "Failed to decrypt Shippo credentials.")
	}
	var creds domain.ShippoCredentials
	if err := json.Unmarshal(plaintext, &creds); err != nil {
		return "", apierror.NewInternalError(err, "Failed to parse Shippo credentials.")
	}
	return creds.APIKey, nil
}

var shipmentSvcTracer = tracing.GetTracer("core-service.shipment_service")

type shipmentSvcImpl struct {
	repos                domain.RepoFactory
	mediatorFactory      domain.MediatorFactory
	txManager            TransactionManager
	shippoFactory        domain.ShippoClientFactory
	encryptionKey        []byte
	notificationPub      domain.NotificationPublisher
	billingPub           domain.BillingPublisher
	s3Client             s3client.ObjectStore
	shippingLabelsBucket string
	frontendURL          string
	branding             BrandingAssets
	outboxNotifier       messaging.OutboxNotifier
}

type ShipmentSvcConfig struct {
	// Repos (required) is the repository factory.
	Repos domain.RepoFactory

	// MediatorFactory (required) builds the mediators used by this service.
	MediatorFactory domain.MediatorFactory

	// TxManager (required) wraps multi-step operations in database transactions.
	TxManager TransactionManager

	// ShippoFactory (optional; default: nil) builds Shippo shipping clients. It is not validated at construction; shipping code paths panic at runtime if it is unset.
	ShippoFactory domain.ShippoClientFactory

	// EncryptionKey (optional; default: nil) decrypts stored integration credentials (e.g. the Shippo API key). It is not validated at construction; live-rate code paths fail at runtime if it is unset while a Shippo integration is configured.
	EncryptionKey []byte

	// NotificationPub (optional; default: nil) publishes notification messages to the outbox. It is not validated
	// at construction.
	NotificationPub domain.NotificationPublisher

	// Meters the invoice a ship creates (optional; default: nil). Not validated; a nil publisher skips
	// metering rather than failing the ship.
	BillingPub domain.BillingPublisher

	// Stores a shipped label and removes a voided one (optional; default: nil). Not validated; a nil client skips both.
	S3Client s3client.ObjectStore

	// Names the S3 bucket holding shipping labels (optional; default: ""). Not validated; an empty bucket skips both.
	ShippingLabelsBucket string

	// FrontendURL (optional; default: "") is the dashboard base URL behind the invoice email's
	// order-online link. Not validated; an empty URL drops the link.
	FrontendURL string

	// Branding (optional) resolves the merchant logo for the invoice PDF letterhead. Omitted, it falls back to a text-only letterhead.
	Branding BrandingAssets

	// OutboxNotifier (optional; default: nil) wakes the outbox enqueuer the instant a void's allocation requests commit, so released stock is offered to open demand on the next moment rather than on the enqueuer's next idle poll. When nil, the requests are still picked up on the next poll.
	OutboxNotifier messaging.OutboxNotifier
}

func (c *ShipmentSvcConfig) validate() error {
	if c.Repos == nil {
		return fmt.Errorf("shipment service: repos is required")
	}
	if c.MediatorFactory == nil {
		return fmt.Errorf("shipment service: mediator factory is required")
	}
	if c.TxManager == nil {
		return fmt.Errorf("shipment service: tx manager is required")
	}
	return nil
}

func NewShipmentSvc(config *ShipmentSvcConfig) domain.ShipmentSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &shipmentSvcImpl{
		repos:                config.Repos,
		mediatorFactory:      config.MediatorFactory,
		txManager:            config.TxManager,
		shippoFactory:        config.ShippoFactory,
		encryptionKey:        config.EncryptionKey,
		notificationPub:      config.NotificationPub,
		billingPub:           config.BillingPub,
		s3Client:             config.S3Client,
		shippingLabelsBucket: config.ShippingLabelsBucket,
		frontendURL:          config.FrontendURL,
		branding:             config.Branding,
		outboxNotifier:       config.OutboxNotifier,
	}
}

// kickOutbox wakes the outbox enqueuer so a just-committed allocation request is picked up
// immediately rather than on the enqueuer's next idle poll, which can be up to MaxPollInterval away.
// No-op when no notifier was injected. Call only after the writing transaction has committed —
// kicking while it is still open races the poll against a row it cannot yet see.
func (s *shipmentSvcImpl) kickOutbox() {
	if s.outboxNotifier != nil {
		s.outboxNotifier.Notify()
	}
}

func (s *shipmentSvcImpl) mediators() domain.Mediators {
	return s.mediatorFactory.Build(s.repos)
}

func (s *shipmentSvcImpl) withTx(ctx context.Context, fn func(context.Context, *shipmentSvcImpl) *apierror.APIError) *apierror.APIError {
	return s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		txSvc := &shipmentSvcImpl{
			repos:                f,
			mediatorFactory:      s.mediatorFactory,
			txManager:            s.txManager,
			shippoFactory:        s.shippoFactory,
			encryptionKey:        s.encryptionKey,
			notificationPub:      s.notificationPub,
			billingPub:           s.billingPub,
			s3Client:             s.s3Client,
			shippingLabelsBucket: s.shippingLabelsBucket,
			frontendURL:          s.frontendURL,
		}
		return fn(txCtx, txSvc)
	})
}

func (s *shipmentSvcImpl) ListShipments(ctx context.Context, params domain.ListShipmentsParams) (*domain.ListShipmentsResult, *apierror.APIError) {
	ctx, span := shipmentSvcTracer.Start(ctx, "service.shipment.list")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainShipments, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	params.AccountID = identity.Target.AccountID

	result, apiErr := s.repos.NewShipmentRepo().List(ctx, params)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	// Expand lines per shipment only when requested (so the list can serve the lines.item array filter).
	for _, include := range params.Includes {
		if include == "lines" {
			lineRepo := s.repos.NewShipmentLineRepo()
			for _, shp := range result.Shipments {
				lines, apiErr := lineRepo.ListByShipment(ctx, shp.ID)
				if apiErr != nil {
					return nil, tracing.Trace(span, apiErr)
				}
				shp.Lines = lines
			}
			break
		}
	}

	return result, nil
}

func (s *shipmentSvcImpl) GetShipment(ctx context.Context, params domain.GetShipmentParams) (*domain.Shipment, *apierror.APIError) {
	ctx, span := shipmentSvcTracer.Start(ctx, "service.shipment.get")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsAssignedActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := checkShipmentReadPermission(identity); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if !identity.IsTargetAccountSet() {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("The OpenMRP-Account header is required."))
	}

	if identity.IsExternalTarget() {
		meds := s.mediators()
		// Counterparty-aware: a customer-portal relation actor may read shipments on
		// orders they bought. Data stays scoped to Target.AccountID; the owner-side
		// CheckReadAccess only allows the actor->target direction and wrongly rejects them.
		if apiErr := meds.ReadAccess.CheckCounterpartyReadAccess(ctx, *identity.ActorAccountID(), identity.Target.AccountID); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}

	params.AccountID = identity.Target.AccountID

	shipment, apiErr := s.repos.NewShipmentRepo().Get(ctx, params)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	// Buyer scope: customer actors may only retrieve shipments for orders they bought.
	if identity.IsCustomerUser() {
		actorAccountID := identity.ActorAccountID()
		if actorAccountID == nil || shipment.CustomerID != *actorAccountID {
			return nil, tracing.Trace(span, apierror.NewResourceNotFoundError("Shipment not found."))
		}
	}

	// Load includes
	for _, inc := range params.Includes {
		switch inc {
		case "lines":
			lines, apiErr := s.repos.NewShipmentLineRepo().ListByShipment(ctx, params.ShipmentID)
			if apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			shipment.Lines = lines
		case "shipping_cases":
			cases, apiErr := s.repos.NewShippingCaseRepo().ListByShipment(ctx, params.ShipmentID)
			if apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			shipment.ShippingCases = cases
		}
	}

	return shipment, nil
}

func (s *shipmentSvcImpl) UpdateShipment(ctx context.Context, params domain.UpdateShipmentParams) (*domain.Shipment, *apierror.APIError) {
	ctx, span := shipmentSvcTracer.Start(ctx, "service.shipment.update")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainShipments, types.ActionUpdate); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	params.AccountID = identity.Target.AccountID

	meds := s.mediators()

	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, identity)
	if apiErr != nil {
		return nil, apiErr
	}

	switch domain.RecoveryPoint(idempotencyKey.RecoveryPoint) {
	case domain.RecoveryPointFinished:
		cached, err := idempotency.UnmarshalCachedResponse[domain.Shipment](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.Shipment
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *shipmentSvcImpl) *apierror.APIError {
			txRepo := txSvc.repos.NewShipmentRepo()

			old, apiErr := txRepo.Get(txCtx, domain.GetShipmentParams{
				AccountID:  params.AccountID,
				ShipmentID: params.ShipmentID,
			})
			if apiErr != nil {
				return apiErr
			}

			if apiErr := checkShipmentRoutingStillMutable(old, params); apiErr != nil {
				return apiErr
			}

			// The SQL assigns the service level outright rather than COALESCE-ing it, so an omitted
			// field has to carry the current value forward; an explicit null falls through and clears.
			params.ServiceLevelID = params.ServiceLevelID.BackfillUnsetPtr(old.ServiceLevelID)

			if apiErr := cascadeCarrierToShippingCases(txCtx, txSvc.repos, params.AccountID, params.ShipmentID, params.CarrierID); apiErr != nil {
				return apiErr
			}

			updated, apiErr := txRepo.Update(txCtx, params)
			if apiErr != nil {
				return apiErr
			}
			result = updated

			if slices.Contains(params.Includes, "lines") {
				lines, apiErr := txSvc.repos.NewShipmentLineRepo().ListByShipment(txCtx, params.ShipmentID)
				if apiErr != nil {
					return apiErr
				}
				result.Lines = lines
			}
			if slices.Contains(params.Includes, "shipping_cases") {
				cases, apiErr := txSvc.repos.NewShippingCaseRepo().ListByShipment(txCtx, params.ShipmentID)
				if apiErr != nil {
					return apiErr
				}
				result.ShippingCases = cases
			}

			changes := audit.ComputeChanges(old, updated)

			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:      domain.ServiceName,
				Action:           constants.AuditActionUpdate,
				ResourceType:     constants.ObjectTypeShipment,
				ResourceID:       updated.ID,
				RootResourceType: constants.ObjectTypeSalesOrder,
				RootResourceID:   updated.SalesOrderID,
				Changes:          changes,
			}); apiErr != nil {
				return apiErr
			}

			return txSvc.mediators().Idempotency.CacheSuccessResponse(txCtx, idempotencyKey.TypeID, result)
		})

		if apiErr != nil {
			return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, apiErr)
		}

		return result, nil

	default:
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Unexpected recovery point: "+idempotencyKey.RecoveryPoint))
	}
}

// Moves every case on the shipment onto the incoming carrier. Each case builds its own tracking
// deep-link from its carrier's code, so cases left behind link to a carrier that never carried them.
func cascadeCarrierToShippingCases(txCtx context.Context, repos domain.RepoFactory, accountID, shipmentID string, carrierID *string) *apierror.APIError {
	if carrierID == nil {
		return nil
	}
	return repos.NewShippingCaseRepo().RepointToCarrier(txCtx, accountID, shipmentID, *carrierID)
}

// Overrides the routing of a shipment that has already left, which the ordinary update refuses.
// Reserved for admins recovering a mis-routed dispatch, so it deliberately skips that guard.
func (s *shipmentSvcImpl) AdminUpdateShipmentTracking(ctx context.Context, params domain.AdminUpdateShipmentTrackingParams) (*domain.Shipment, *apierror.APIError) {
	ctx, span := shipmentSvcTracer.Start(ctx, "service.shipment.admin_update_tracking")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckIsAdmin(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainShipments, types.ActionUpdate); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	params.AccountID = identity.Target.AccountID

	meds := s.mediators()

	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, identity)
	if apiErr != nil {
		return nil, apiErr
	}

	switch domain.RecoveryPoint(idempotencyKey.RecoveryPoint) {
	case domain.RecoveryPointFinished:
		cached, err := idempotency.UnmarshalCachedResponse[domain.Shipment](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.Shipment
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *shipmentSvcImpl) *apierror.APIError {
			txRepo := txSvc.repos.NewShipmentRepo()

			old, apiErr := txRepo.Get(txCtx, domain.GetShipmentParams{
				AccountID:  params.AccountID,
				ShipmentID: params.ShipmentID,
			})
			if apiErr != nil {
				return apiErr
			}

			if old.ShippedAt == nil {
				return apierror.NewValidationError("Shipment has not been shipped yet. Use the regular update endpoint.")
			}

			if apiErr := txSvc.checkAdminTrackingRouting(txCtx, params); apiErr != nil {
				return apiErr
			}

			if apiErr := cascadeCarrierToShippingCases(txCtx, txSvc.repos, params.AccountID, params.ShipmentID, params.CarrierID); apiErr != nil {
				return apiErr
			}

			updated, apiErr := txRepo.Update(txCtx, domain.UpdateShipmentParams{
				AccountID:            params.AccountID,
				ShipmentID:           params.ShipmentID,
				MasterTrackingNumber: params.MasterTrackingNumber,
				CarrierID:            params.CarrierID,
				// The column is assigned outright rather than coalesced, so an unsent field has to carry the current value forward.
				ServiceLevelID: params.ServiceLevelID.BackfillUnsetPtr(old.ServiceLevelID),
				Includes:       params.Includes,
			})
			if apiErr != nil {
				return apiErr
			}
			result = updated

			if slices.Contains(params.Includes, "lines") {
				lines, apiErr := txSvc.repos.NewShipmentLineRepo().ListByShipment(txCtx, params.ShipmentID)
				if apiErr != nil {
					return apiErr
				}
				result.Lines = lines
			}
			if slices.Contains(params.Includes, "shipping_cases") {
				cases, apiErr := txSvc.repos.NewShippingCaseRepo().ListByShipment(txCtx, params.ShipmentID)
				if apiErr != nil {
					return apiErr
				}
				result.ShippingCases = cases
			}

			changes := audit.ComputeChanges(old, updated)

			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:      domain.ServiceName,
				Action:           constants.AuditActionUpdate,
				ResourceType:     constants.ObjectTypeShipment,
				ResourceID:       updated.ID,
				RootResourceType: constants.ObjectTypeSalesOrder,
				RootResourceID:   updated.SalesOrderID,
				Changes:          changes,
			}); apiErr != nil {
				return apiErr
			}

			return txSvc.mediators().Idempotency.CacheSuccessResponse(txCtx, idempotencyKey.TypeID, result)
		})

		if apiErr != nil {
			return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, apiErr)
		}

		return result, nil

	default:
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Unexpected recovery point: "+idempotencyKey.RecoveryPoint))
	}
}

// Resolves the acting user to the account_user row shipment.shipped_by_id references, 404ing like legacy
// when a user actor has no membership. Non-user actors (an API key is not an account user) ship unattributed.
func (s *shipmentSvcImpl) resolveShippedByID(ctx context.Context, identity *types.Identity, accountID string) (string, *apierror.APIError) {
	if identity.Actor == nil || identity.Actor.ID == "" {
		return "", nil
	}

	accountUserID, apiErr := s.repos.NewAccountUserRepo().ResolveAccountUserID(ctx, accountID, identity.Actor.ID)
	if apiErr != nil {
		if apierror.IsNotFound(apiErr) {
			if identity.Type == types.IdentityActorTypeUser {
				return "", apierror.NewResourceNotFoundError("Account user not found.")
			}
			return "", nil
		}
		return "", apiErr
	}

	return accountUserID, nil
}

// Rejects a carrier or service level the account cannot reach, before the override rewrites routing.
func (s *shipmentSvcImpl) checkAdminTrackingRouting(txCtx context.Context, params domain.AdminUpdateShipmentTrackingParams) *apierror.APIError {
	if params.CarrierID != nil {
		if _, apiErr := s.repos.NewCarrierRepo().Get(txCtx, domain.GetCarrierParams{AccountID: params.AccountID, CarrierID: *params.CarrierID}); apiErr != nil {
			return apiErr
		}
	}
	if serviceLevelID, ok := params.ServiceLevelID.Value(); ok {
		if _, apiErr := s.repos.NewServiceLevelRepo().Get(txCtx, params.AccountID, serviceLevelID); apiErr != nil {
			return apiErr
		}
	}
	return nil
}

func (s *shipmentSvcImpl) DeleteShipment(ctx context.Context, params domain.DeleteShipmentParams) *apierror.APIError {
	ctx, span := shipmentSvcTracer.Start(ctx, "service.shipment.delete")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainShipments, types.ActionUpdate); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	params.AccountID = identity.Target.AccountID

	shipmentRepo := s.repos.NewShipmentRepo()

	shipment, apiErr := shipmentRepo.Get(ctx, domain.GetShipmentParams{
		AccountID:  params.AccountID,
		ShipmentID: params.ShipmentID,
	})
	if apiErr != nil {
		if apierror.IsNotFound(apiErr) {
			wasDeleted, deletedCheckErr := s.repos.NewDeletedRecordRepo().Exists(ctx, constants.DeletedRecordResourceTypeShipment, params.ShipmentID)
			if deletedCheckErr != nil {
				return tracing.Trace(span, deletedCheckErr)
			}
			if wasDeleted {
				return tracing.Trace(
					span,
					apierror.NewAlreadyDeletedError("This shipment has already been deleted and can no longer be modified."),
				)
			}
		}
		return tracing.Trace(span, apiErr)
	}

	return s.withTx(ctx, func(txCtx context.Context, txSvc *shipmentSvcImpl) *apierror.APIError {
		// Unpack pick lines associated with this shipment's lines (clear packed_at)
		if apiErr := txSvc.repos.NewPickLineRepo().UnpackByShipment(txCtx, params.ShipmentID); apiErr != nil {
			return apiErr
		}
		// Find the pick for this shipment's order and mark it as unpacked (clear finished_at)
		pickID, apiErr := txSvc.repos.NewPickRepo().FindIDByShipmentOrder(txCtx, params.AccountID, params.ShipmentID)
		if apiErr != nil {
			return apiErr
		}
		if pickID != "" {
			if apiErr := txSvc.repos.NewPickRepo().ClearFinishedAt(txCtx, params.AccountID, pickID); apiErr != nil {
				return apiErr
			}
		}

		if apiErr := txSvc.repos.NewDeletedRecordRepo().Create(txCtx, constants.DeletedRecordResourceTypeShipment, shipment.ID, shipment); apiErr != nil {
			return apiErr
		}

		// Delete shipping cases first
		if apiErr := txSvc.repos.NewShippingCaseRepo().DeleteByShipment(txCtx, params.ShipmentID); apiErr != nil {
			return apiErr
		}
		// Delete shipment lines
		if apiErr := txSvc.repos.NewShipmentLineRepo().DeleteByShipment(txCtx, params.ShipmentID); apiErr != nil {
			return apiErr
		}
		// Delete shipment
		if apiErr := txSvc.repos.NewShipmentRepo().Delete(txCtx, params.AccountID, params.ShipmentID); apiErr != nil {
			return apiErr
		}

		changes := audit.ComputeChanges(shipment, (*domain.Shipment)(nil))

		if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
			ServiceName:      domain.ServiceName,
			Action:           constants.AuditActionDelete,
			ResourceType:     constants.ObjectTypeShipment,
			ResourceID:       shipment.ID,
			RootResourceType: constants.ObjectTypeSalesOrder,
			RootResourceID:   shipment.SalesOrderID,
			Changes:          changes,
		}); apiErr != nil {
			return apiErr
		}

		return nil
	})
}

func (s *shipmentSvcImpl) ShipShipment(ctx context.Context, params domain.ShipShipmentParams) (*domain.Shipment, *apierror.APIError) {
	ctx, span := shipmentSvcTracer.Start(ctx, "service.shipment.ship")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainShipments, types.ActionUpdate); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	params.AccountID = identity.Target.AccountID

	meds := s.mediators()

	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, identity)
	if apiErr != nil {
		return nil, apiErr
	}

	switch domain.RecoveryPoint(idempotencyKey.RecoveryPoint) {
	case domain.RecoveryPointFinished:
		cached, err := idempotency.UnmarshalCachedResponse[domain.Shipment](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		// Phase 1: Validate shipment and gather data
		shipmentRepo := s.repos.NewShipmentRepo()
		shipment, apiErr := shipmentRepo.Get(ctx, domain.GetShipmentParams{
			AccountID:  params.AccountID,
			ShipmentID: params.ShipmentID,
		})
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}

		if shipment.StatusCode == "shipped" {
			return nil, tracing.Trace(span, apierror.NewConflictErrorWithParam("Shipment has already been shipped.", "id"))
		}

		// Ship creates the invoice, so enforce the per-billing-period invoice limit here — matching
		// legacy's canCreateInvoice on ship — before any mutation.
		if apiErr := enforceInvoicesPerPeriodLimit(ctx, s.repos, params.AccountID); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}

		shippedByID, apiErr := s.resolveShippedByID(ctx, identity, params.AccountID)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}

		// Resolve the master tracking number for the shipment. Sandbox accounts get a placeholder;
		// real accounts buy carrier labels from Shippo here — a foreign mutation, so it runs before
		// the transaction and stages RecoveryPointShipLabelsCreated once its results are persisted.
		masterTracking, apiErr := s.resolveShipmentTracking(ctx, shipment, idempotencyKey.TypeID)
		if apiErr != nil {
			return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, apiErr)
		}

		// The invoice PDF embeds the letterhead logo, so fetch its bytes here: inside the transaction
		// a stalled logo host would hold the ship's row locks for the length of the request.
		letterheadLogo := fetchAccountLogo(ctx, s.repos, s.branding, params.AccountID)

		// Phase 3: Atomic transaction - mark shipped, create invoice, add SSCC
		var result *domain.Shipment
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *shipmentSvcImpl) *apierror.APIError {
			txShipmentRepo := txSvc.repos.NewShipmentRepo()
			txCaseRepo := txSvc.repos.NewShippingCaseRepo()

			// Mark shipping cases as shipped
			if apiErr := txCaseRepo.MarkShippedByShipment(txCtx, params.ShipmentID); apiErr != nil {
				return apiErr
			}

			// Add SSCC to shipping cases
			cases, apiErr := txCaseRepo.ListByShipment(txCtx, params.ShipmentID)
			if apiErr != nil {
				return apiErr
			}
			for _, sc := range cases {
				if sc.SSCC == nil {
					counter, apiErr := txCaseRepo.FindAndIncrementSsccCounter(txCtx, params.AccountID)
					if apiErr != nil {
						return apiErr
					}
					sscc := domain.GenerateSSCC(counter)
					if apiErr := txCaseRepo.AddSscc(txCtx, sc.ID, sscc); apiErr != nil {
						return apiErr
					}
				}
			}

			// Mark shipment as shipped
			if apiErr := txShipmentRepo.MarkShipped(txCtx, params.AccountID, params.ShipmentID, shippedByID); apiErr != nil {
				return apiErr
			}

			if masterTracking != nil {
				if apiErr := txShipmentRepo.SetMasterTracking(txCtx, params.AccountID, params.ShipmentID, *masterTracking); apiErr != nil {
					return apiErr
				}
			}

			if apiErr := txSvc.createInvoiceAndStampOrderOnShip(txCtx, shipment, params.EmailCustomer, letterheadLogo); apiErr != nil {
				return apiErr
			}

			// Re-fetch for response
			updated, apiErr := txShipmentRepo.Get(txCtx, domain.GetShipmentParams{
				AccountID:  params.AccountID,
				ShipmentID: params.ShipmentID,
			})
			if apiErr != nil {
				return apiErr
			}
			result = updated

			if slices.Contains(params.Includes, "lines") {
				lines, apiErr := txSvc.repos.NewShipmentLineRepo().ListByShipment(txCtx, params.ShipmentID)
				if apiErr != nil {
					return apiErr
				}
				result.Lines = lines
			}
			if slices.Contains(params.Includes, "shipping_cases") {
				cases, apiErr := txSvc.repos.NewShippingCaseRepo().ListByShipment(txCtx, params.ShipmentID)
				if apiErr != nil {
					return apiErr
				}
				result.ShippingCases = cases
			}

			changes := audit.ComputeChanges(shipment, updated)

			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:      domain.ServiceName,
				Action:           constants.AuditActionUpdate,
				ResourceType:     constants.ObjectTypeShipment,
				ResourceID:       updated.ID,
				RootResourceType: constants.ObjectTypeSalesOrder,
				RootResourceID:   updated.SalesOrderID,
				Changes:          changes,
			}); apiErr != nil {
				return apiErr
			}

			return txSvc.mediators().Idempotency.CacheSuccessResponse(txCtx, idempotencyKey.TypeID, result)
		})

		if apiErr != nil {
			return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, apiErr)
		}

		return result, nil

	case domain.RecoveryPointShipLabelsCreated:
		// Labels were already created in a prior attempt. Proceed with atomic phase.
		// Fetch old state for audit diff
		old, apiErr := s.repos.NewShipmentRepo().Get(ctx, domain.GetShipmentParams{
			AccountID:  params.AccountID,
			ShipmentID: params.ShipmentID,
		})
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}

		shippedByID, apiErr := s.resolveShippedByID(ctx, identity, params.AccountID)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}

		// Fetched before the transaction for the same reason as the ship path above.
		letterheadLogo := fetchAccountLogo(ctx, s.repos, s.branding, params.AccountID)

		var result *domain.Shipment
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *shipmentSvcImpl) *apierror.APIError {
			txShipmentRepo := txSvc.repos.NewShipmentRepo()
			txCaseRepo := txSvc.repos.NewShippingCaseRepo()

			if apiErr := txCaseRepo.MarkShippedByShipment(txCtx, params.ShipmentID); apiErr != nil {
				return apiErr
			}

			cases, apiErr := txCaseRepo.ListByShipment(txCtx, params.ShipmentID)
			if apiErr != nil {
				return apiErr
			}
			for _, sc := range cases {
				if sc.SSCC == nil {
					counter, apiErr := txCaseRepo.FindAndIncrementSsccCounter(txCtx, params.AccountID)
					if apiErr != nil {
						return apiErr
					}
					sscc := domain.GenerateSSCC(counter)
					if apiErr := txCaseRepo.AddSscc(txCtx, sc.ID, sscc); apiErr != nil {
						return apiErr
					}
				}
			}

			if apiErr := txShipmentRepo.MarkShipped(txCtx, params.AccountID, params.ShipmentID, shippedByID); apiErr != nil {
				return apiErr
			}

			if apiErr := txSvc.createInvoiceAndStampOrderOnShip(txCtx, old, params.EmailCustomer, letterheadLogo); apiErr != nil {
				return apiErr
			}

			updated, apiErr := txShipmentRepo.Get(txCtx, domain.GetShipmentParams{
				AccountID:  params.AccountID,
				ShipmentID: params.ShipmentID,
			})
			if apiErr != nil {
				return apiErr
			}
			result = updated

			if slices.Contains(params.Includes, "lines") {
				lines, apiErr := txSvc.repos.NewShipmentLineRepo().ListByShipment(txCtx, params.ShipmentID)
				if apiErr != nil {
					return apiErr
				}
				result.Lines = lines
			}
			if slices.Contains(params.Includes, "shipping_cases") {
				cases, apiErr := txSvc.repos.NewShippingCaseRepo().ListByShipment(txCtx, params.ShipmentID)
				if apiErr != nil {
					return apiErr
				}
				result.ShippingCases = cases
			}

			changes := audit.ComputeChanges(old, updated)

			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:      domain.ServiceName,
				Action:           constants.AuditActionUpdate,
				ResourceType:     constants.ObjectTypeShipment,
				ResourceID:       updated.ID,
				RootResourceType: constants.ObjectTypeSalesOrder,
				RootResourceID:   updated.SalesOrderID,
				Changes:          changes,
			}); apiErr != nil {
				return apiErr
			}

			return txSvc.mediators().Idempotency.CacheSuccessResponse(txCtx, idempotencyKey.TypeID, result)
		})

		if apiErr != nil {
			return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, apiErr)
		}

		return result, nil

	default:
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Unexpected recovery point: "+idempotencyKey.RecoveryPoint))
	}
}

func (s *shipmentSvcImpl) VoidShipment(ctx context.Context, params domain.VoidShipmentParams) (*domain.Shipment, *apierror.APIError) {
	ctx, span := shipmentSvcTracer.Start(ctx, "service.shipment.void")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainShipments, types.ActionUpdate); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	params.AccountID = identity.Target.AccountID

	meds := s.mediators()

	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, identity)
	if apiErr != nil {
		return nil, apiErr
	}

	switch domain.RecoveryPoint(idempotencyKey.RecoveryPoint) {
	case domain.RecoveryPointFinished:
		cached, err := idempotency.UnmarshalCachedResponse[domain.Shipment](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		// Phase 1: Validate shipment
		shipmentRepo := s.repos.NewShipmentRepo()
		shipment, apiErr := shipmentRepo.Get(ctx, domain.GetShipmentParams{
			AccountID:  params.AccountID,
			ShipmentID: params.ShipmentID,
		})
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}

		if shipment.StatusCode != "shipped" {
			return nil, tracing.Trace(span, apierror.NewConflictErrorWithParam("Shipment is not in shipped status.", "id"))
		}

		// Refund any purchased carrier labels before the atomic phase. Sandbox accounts never bought
		// real labels, so this is a no-op there; for real accounts it refunds each Shippo transaction
		// and drops the stored label, best-effort, then stages RecoveryPointVoidLabelsRefunded.
		if apiErr := s.refundShippingLabels(ctx, shipment, idempotencyKey.TypeID); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}

		// Phase 3: Atomic transaction - void cases, delete invoice, mark order unfulfilled, mark shipment packed
		fallthrough

	case domain.RecoveryPointVoidLabelsRefunded:
		var result *domain.Shipment
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *shipmentSvcImpl) *apierror.APIError {
			txShipmentRepo := txSvc.repos.NewShipmentRepo()
			txCaseRepo := txSvc.repos.NewShippingCaseRepo()
			txInvoiceRepo := txSvc.repos.NewInvoiceRepo()
			txSalesOrderRepo := txSvc.repos.NewSalesOrderRepo()

			// Look up the shipment to get the sales order ID for unfulfillment
			shipment, apiErr := txShipmentRepo.Get(txCtx, domain.GetShipmentParams{
				AccountID:  params.AccountID,
				ShipmentID: params.ShipmentID,
			})
			if apiErr != nil {
				return apiErr
			}

			// Delete invoice if one exists for this shipment
			invoiceID, apiErr := txShipmentRepo.FindInvoiceIDByShipment(txCtx, params.AccountID, params.ShipmentID)
			if apiErr != nil {
				return apiErr
			}
			if invoiceID != nil {
				if apiErr := txSvc.reverseInventoryOnVoid(txCtx, shipment); apiErr != nil {
					return apiErr
				}

				// Delete invoice lines then invoice
				if apiErr := txInvoiceRepo.DeleteLinesByInvoice(txCtx, *invoiceID); apiErr != nil {
					return apiErr
				}
				if apiErr := txInvoiceRepo.Delete(txCtx, params.AccountID, *invoiceID); apiErr != nil {
					return apiErr
				}

				// Voiding destroys the invoice outright, so the order's history has to record it going.
				if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
					ServiceName:      domain.ServiceName,
					Action:           constants.AuditActionDelete,
					ResourceType:     constants.ObjectTypeInvoice,
					ResourceID:       *invoiceID,
					RootResourceType: constants.ObjectTypeSalesOrder,
					RootResourceID:   shipment.SalesOrderID,
				}); apiErr != nil {
					return apiErr
				}
			}

			// Mark the sales order as unfulfilled (reset to "issued" status, clear completed_at and first_ship_at)
			if apiErr := txSalesOrderRepo.MarkUnfulfilled(txCtx, params.AccountID, shipment.SalesOrderID); apiErr != nil {
				return apiErr
			}

			// Void shipping cases (clear tracking, labels, freight amount)
			if apiErr := txCaseRepo.VoidByShipment(txCtx, params.ShipmentID); apiErr != nil {
				return apiErr
			}

			// Mark shipment as voided (back to packed, clear tracking/invoice/shipped info)
			if apiErr := txShipmentRepo.MarkVoided(txCtx, params.AccountID, params.ShipmentID); apiErr != nil {
				return apiErr
			}

			// Re-fetch for response
			updated, apiErr := txShipmentRepo.Get(txCtx, domain.GetShipmentParams{
				AccountID:  params.AccountID,
				ShipmentID: params.ShipmentID,
			})
			if apiErr != nil {
				return apiErr
			}
			result = updated

			changes := audit.ComputeChanges(shipment, updated)

			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:      domain.ServiceName,
				Action:           constants.AuditActionUpdate,
				ResourceType:     constants.ObjectTypeShipment,
				ResourceID:       updated.ID,
				RootResourceType: constants.ObjectTypeSalesOrder,
				RootResourceID:   updated.SalesOrderID,
				Changes:          changes,
			}); apiErr != nil {
				return apiErr
			}

			return txSvc.mediators().Idempotency.CacheSuccessResponse(txCtx, idempotencyKey.TypeID, result)
		})

		if apiErr != nil {
			return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, apiErr)
		}

		// After the commit, never inside it: the allocation requests the reversal wrote have to be
		// visible to the enqueuer's poll query for this kick to find anything.
		s.kickOutbox()

		return result, nil

	default:
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Unexpected recovery point: "+idempotencyKey.RecoveryPoint))
	}
}

func (s *shipmentSvcImpl) EstimateRate(ctx context.Context, params domain.EstimateRateParams) (float64, *apierror.APIError) {
	ctx, span := shipmentSvcTracer.Start(ctx, "service.shipment.estimate_rate")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return 0, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsAssignedActor(); apiErr != nil {
		return 0, tracing.Trace(span, apiErr)
	}
	if identity.IsInternalActor() {
		if apiErr := checkShipmentReadPermission(identity); apiErr != nil {
			return 0, tracing.Trace(span, apiErr)
		}
	} else if identity.IsCustomerUser() {
		if params.CustomerID == nil || identity.ActorAccountID() == nil || *identity.ActorAccountID() != *params.CustomerID {
			return 0, tracing.Trace(span, apierror.NewAuthorizationError("You are not authorized to access this resource."))
		}
	} else {
		return 0, tracing.Trace(span, apierror.NewValidationError("Invalid actor type."))
	}

	if !identity.IsTargetAccountSet() {
		return 0, tracing.Trace(span, apierror.NewAuthenticationError("The OpenMRP-Account-ID header is required."))
	}

	if identity.IsExternalTarget() {
		meds := s.mediators()
		if apiErr := meds.ReadAccess.CheckCounterpartyReadAccess(ctx, *identity.ActorAccountID(), identity.Target.AccountID); apiErr != nil {
			return 0, tracing.Trace(span, apiErr)
		}
	}

	params.AccountID = identity.Target.AccountID

	return estimateShippingRate(ctx, s.repos, s.shippoFactory, s.encryptionKey, params)
}

// estimateShippingRate computes the posted shipping rate for an order or shipment, mirroring Dashboard's estimatePostedShippingRate cascade: product-line freight exemption → customer/group freight exemption → shipping-term free/flat/min-order → carrier-without-Shippo → no Shippo integration → live Shippo rate (already marked up by the Shippo client). A nil shippoFactory short-circuits the live rate to 0. It is shared by the shipment estimate-rate endpoint and sales-order shipping-line synthesis.
func estimateShippingRate(ctx context.Context, repos domain.RepoFactory, shippoFactory domain.ShippoClientFactory, encryptionKey []byte, params domain.EstimateRateParams) (float64, *apierror.APIError) {
	// Check product line freight exemption: if any product line is freight exempt, rate is 0.
	if len(params.ProductLineIDs) > 0 {
		productLineRepo := repos.NewProductLineRepo()
		for _, plID := range params.ProductLineIDs {
			pl, apiErr := productLineRepo.Get(ctx, domain.GetProductLineParams{
				AccountID:     params.AccountID,
				ProductLineID: plID,
			})
			if apiErr != nil {
				continue
			}
			if pl.FreightPolicy == constants.FreightPolicyFree {
				return 0, nil
			}
		}
	}

	// Check customer-level freight exemptions and shipping term logic.
	if params.CustomerID != nil {
		customerRepo := repos.NewCustomerRepo()
		// Price groups carry their own freight policy, so they must be hydrated to evaluate group-level freight exemption below.
		customer, apiErr := customerRepo.Get(ctx, params.AccountID, *params.CustomerID, []string{"price_groups"})
		if apiErr != nil {
			return 0, apiErr
		}

		// Customer, its type group, or any price group is freight exempt.
		if isCustomerOrGroupFreightExempt(customer) {
			return 0, nil
		}

		// Check the customer's default shipping term. The free-shipping service-level
		// allowlist must be loaded so the minimum-order branch below can honor it.
		if customer.DefaultShippingTermID != nil {
			shippingTermRepo := repos.NewShippingTermRepo()
			shippingTerm, apiErr := shippingTermRepo.Get(ctx, domain.GetShippingTermParams{
				AccountID:      params.AccountID,
				ShippingTermID: *customer.DefaultShippingTermID,
				Includes:       []string{"free_shipping_service_levels"},
			})
			if apiErr != nil {
				return 0, apiErr
			}

			// Shipping term is free freight.
			if shippingTerm.Type == constants.ShippingTermTypeFreeFreight {
				return 0, nil
			}

			// Shipping term has a flat rate.
			if shippingTerm.Type == constants.ShippingTermTypeFlatRateFreight && shippingTerm.FlatRate != nil {
				flatRate, err := strconv.ParseFloat(shippingTerm.FlatRate.Value, 64)
				if err != nil {
					return 0, apierror.NewInternalError(err, "Failed to parse flat rate value.")
				}
				return flatRate, nil
			}

			// Shipping term has a minimum order value: free shipping over the threshold,
			// but only for the term's allowlisted service levels (or when it has no
			// allowlist). A non-allowlisted service selection over the threshold falls
			// through to the live carrier rate rather than shipping free. Matches legacy
			// order.repo.ts (freeShippingCarrierOptionIDs gating).
			if shippingTerm.MinimumOrderValue != nil && params.OrderTotal != nil {
				minValue, err := strconv.ParseFloat(shippingTerm.MinimumOrderValue.Value, 64)
				if err != nil {
					return 0, apierror.NewInternalError(err, "Failed to parse minimum order value.")
				}
				if *params.OrderTotal > minValue {
					if len(shippingTerm.FreeShippingServiceLevelIDs) == 0 || slices.Contains(shippingTerm.FreeShippingServiceLevelIDs, params.ServiceLevelID) {
						return 0, nil
					}
				}
			}
		}
	}

	// A carrier is required to fetch a live rate; without one there is no rate.
	if params.CarrierID == "" {
		return 0, nil
	}

	// Get carrier to find Shippo carrier account object ID.
	carrierRepo := repos.NewCarrierRepo()
	carrier, apiErr := carrierRepo.Get(ctx, domain.GetCarrierParams{AccountID: params.AccountID, CarrierID: params.CarrierID})
	if apiErr != nil {
		return 0, apiErr
	}

	// If carrier doesn't have Shippo configured, return 0 (no rate available).
	if carrier.ShippoCarrierAccountID == nil || *carrier.ShippoCarrierAccountID == "" {
		return 0, nil
	}

	// Get service level token (optional).
	var serviceLevelToken string
	if params.ServiceLevelID != "" {
		serviceLevelRepo := repos.NewServiceLevelRepo()
		serviceLevel, apiErr := serviceLevelRepo.Get(ctx, params.AccountID, params.ServiceLevelID)
		if apiErr != nil {
			return 0, apiErr
		}
		if serviceLevel.ServiceLevelToken != nil {
			serviceLevelToken = *serviceLevel.ServiceLevelToken
		}
	}

	// Check if account has Shippo integration enabled.
	integrationRepo := repos.NewAccountIntegrationRepo()
	hasIntegration, apiErr := integrationRepo.HasIntegration(ctx, params.AccountID, constants.IntegrationCodeShippo)
	if apiErr != nil {
		return 0, apiErr
	}
	if !hasIntegration || shippoFactory == nil {
		return 0, nil
	}

	// Get account Shippo API key.
	encryptedCreds, _, apiErr := integrationRepo.GetEncryptedCredentials(ctx, params.AccountID, constants.IntegrationCodeShippo)
	if apiErr != nil {
		return 0, apiErr
	}

	apiKey, apiErr := decryptShippoAPIKey(encryptedCreds, encryptionKey, params.AccountID)
	if apiErr != nil {
		return 0, apiErr
	}

	// A live rate requires a real ship-from address. Refuse to quote from an empty
	// origin (matches legacy, which fails order create with "Bill to address not
	// found" when the seller account has no default bill-to) rather than sending an
	// empty from-address to Shippo and returning a meaningless rate.
	if params.FromAddress.Zip == "" || params.FromAddress.Country == "" {
		return 0, apierror.NewValidationError("Cannot estimate shipping: the account has no default billing (ship-from) address.")
	}

	shippoClient := shippoFactory.Build(apiKey)

	rate, apiErr := shippoClient.FetchShippingRate(ctx, domain.FetchShippingRateParams{
		CarrierAccountObjectID: *carrier.ShippoCarrierAccountID,
		ServiceLevelToken:      serviceLevelToken,
		FromAddress:            params.FromAddress,
		ToAddress:              params.ToAddress,
		Parcels:                params.Parcels,
		Billing:                params.Billing,
	})
	if apiErr != nil {
		return 0, apiErr
	}

	return rate, nil
}

// isCustomerOrGroupFreightExempt reports whether the customer, its type group, or any of its price groups is freight exempt, mirroring Dashboard's CustomerUtils.isCustomerOrGroupFreightExempt. PriceGroups must be hydrated on the customer for the group check to be meaningful.
func isCustomerOrGroupFreightExempt(customer *domain.Customer) bool {
	if customer.FreightPolicy == constants.FreightPolicyFree {
		return true
	}
	if customer.TypeGroupFreightPolicy != nil && *customer.TypeGroupFreightPolicy == constants.FreightPolicyFree {
		return true
	}
	for _, pg := range customer.PriceGroups {
		if pg.FreightPolicy == constants.FreightPolicyFree {
			return true
		}
	}
	return false
}

func (s *shipmentSvcImpl) RateShop(ctx context.Context, params domain.RateShopParams) (*domain.RateShopResult, *apierror.APIError) {
	ctx, span := shipmentSvcTracer.Start(ctx, "service.shipment.rate_shop")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsAssignedActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if identity.IsInternalActor() {
		if apiErr := checkShipmentReadPermission(identity); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	} else if identity.IsCustomerUser() {
		if params.CustomerID == nil || identity.ActorAccountID() == nil || *identity.ActorAccountID() != *params.CustomerID {
			return nil, tracing.Trace(span, apierror.NewAuthorizationError("You are not authorized to access this resource."))
		}
	} else {
		return nil, tracing.Trace(span, apierror.NewValidationError("Invalid actor type."))
	}

	if !identity.IsTargetAccountSet() {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("The OpenMRP-Account-ID header is required."))
	}

	if identity.IsExternalTarget() {
		meds := s.mediators()
		if apiErr := meds.ReadAccess.CheckCounterpartyReadAccess(ctx, *identity.ActorAccountID(), identity.Target.AccountID); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}

	params.AccountID = identity.Target.AccountID

	// Origin (ship-from) defaults to the seller account's configured origin (its default billing address) when the caller omits it — customer portals never send the seller's address.
	if params.FromAddress.IsEmpty() {
		origin, apiErr := s.repos.NewSalesOrderRepo().GetAccountOriginAddress(ctx, params.AccountID)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		if origin != nil {
			params.FromAddress = *origin
		}
	}

	freightExemptResult := &domain.RateShopResult{
		Options:       []*domain.RateShopOption{},
		ExemptionType: new("freight_exempt"),
	}

	// 1. Check product line freight exemption: if any product line is freight exempt, return empty.
	if len(params.ProductLineIDs) > 0 {
		productLineRepo := s.repos.NewProductLineRepo()
		for _, plID := range params.ProductLineIDs {
			pl, apiErr := productLineRepo.Get(ctx, domain.GetProductLineParams{
				AccountID:     params.AccountID,
				ProductLineID: plID,
			})
			if apiErr != nil {
				continue
			}
			if pl.FreightPolicy == constants.FreightPolicyFree {
				return freightExemptResult, nil
			}
		}
	}

	// 2. Fetch customer and check customer/group freight exemption.
	var customer *domain.Customer
	var shippingTerm *domain.ShippingTerm
	if params.CustomerID != nil {
		customerRepo := s.repos.NewCustomerRepo()
		var apiErr *apierror.APIError
		// Price groups carry their own freight policy, so they must be hydrated to evaluate group-level freight exemption below.
		customer, apiErr = customerRepo.Get(ctx, params.AccountID, *params.CustomerID, []string{"price_groups"})
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}

		// Customer, its type group, or any price group is freight exempt.
		if isCustomerOrGroupFreightExempt(customer) {
			return freightExemptResult, nil
		}

		// 3. Check shipping term freight exemption.
		if customer.DefaultShippingTermID != nil {
			shippingTermRepo := s.repos.NewShippingTermRepo()
			// The free-shipping service levels drive the per-option free-shipping rules applied during post-processing and are only hydrated when this include is requested.
			shippingTerm, apiErr = shippingTermRepo.Get(ctx, domain.GetShippingTermParams{
				AccountID:      params.AccountID,
				ShippingTermID: *customer.DefaultShippingTermID,
				Includes:       []string{"free_shipping_service_levels"},
			})
			if apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}

			if shippingTerm.Type == constants.ShippingTermTypeFreeFreight {
				return freightExemptResult, nil
			}
		}
	}

	// Extract shipping term configuration.
	var flatRateValue *float64
	var minimumOrderValue *float64
	freeShippingOptionIDs := make(map[string]bool)

	if shippingTerm != nil {
		if shippingTerm.FlatRate != nil {
			v, err := strconv.ParseFloat(shippingTerm.FlatRate.Value, 64)
			if err == nil {
				flatRateValue = &v
			}
		}
		if shippingTerm.MinimumOrderValue != nil {
			v, err := strconv.ParseFloat(shippingTerm.MinimumOrderValue.Value, 64)
			if err == nil {
				minimumOrderValue = &v
			}
		}
		for _, optID := range shippingTerm.FreeShippingServiceLevelIDs {
			freeShippingOptionIDs[optID] = true
		}
	}

	// A flat rate only applies when the term is not a carrier-rate term (mirrors Dashboard's `!isCarrierRate && !!flatRate`); a carrier-rate term keeps live carrier rates even if a stray flat-rate value is stored.
	hasFlatRate := flatRateValue != nil && shippingTerm != nil && shippingTerm.Type != constants.ShippingTermTypeCarrierRateFreight
	hasMinimumOrder := minimumOrderValue != nil
	isMinimumOrderMet := hasMinimumOrder && params.OrderTotal != nil && *params.OrderTotal > *minimumOrderValue
	hasFreeShippingRules := len(freeShippingOptionIDs) > 0

	// 4. List all carriers for the account.
	carrierRepo := s.repos.NewCarrierRepo()
	carriersResult, apiErr := carrierRepo.List(ctx, domain.ListCarriersParams{
		AccountID: params.AccountID,
		Limit:     1000,
	})
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	// Load options for each carrier.
	for _, carrier := range carriersResult.Carriers {
		options, apiErr := carrierRepo.ListOptionsByCarrierID(ctx, params.AccountID, carrier.ID)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		carrier.ServiceLevels = options
	}

	// Filter for portal-enabled carriers/options when called by customer actor.
	carriers := carriersResult.Carriers
	if identity.IsCustomerUser() {
		var filtered []*domain.Carrier
		for _, c := range carriers {
			if !c.IsPortalEnabled {
				continue
			}
			var portalOptions []*domain.ServiceLevel
			for _, o := range c.ServiceLevels {
				if o.IsPortalEnabled {
					portalOptions = append(portalOptions, o)
				}
			}
			carrierCopy := *c
			carrierCopy.ServiceLevels = portalOptions
			filtered = append(filtered, &carrierCopy)
		}
		carriers = filtered
	}

	// 5. Build a Shippo client only if a carrier actually needs live Shippo rates.
	needsShippo := false
	for _, carrier := range carriers {
		if carrier.ShippoCarrierAccountID != nil && *carrier.ShippoCarrierAccountID != "" {
			needsShippo = true
			break
		}
	}

	var shippoClient domain.ShippoClient
	if needsShippo {
		integrationRepo := s.repos.NewAccountIntegrationRepo()
		hasShippoIntegration, apiErr := integrationRepo.HasIntegration(ctx, params.AccountID, constants.IntegrationCodeShippo)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}

		if hasShippoIntegration {
			encryptedCreds, _, apiErr := integrationRepo.GetEncryptedCredentials(ctx, params.AccountID, constants.IntegrationCodeShippo)
			if apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			apiKey, apiErr := decryptShippoAPIKey(encryptedCreds, s.encryptionKey, params.AccountID)
			if apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			shippoClient = s.shippoFactory.Build(apiKey)
		}
	}

	// 6. For each carrier, fetch rates. Each Shippo carrier costs a live rating round-trip, so they run concurrently and results are collected per carrier to keep ordering stable.
	//
	// The fan-out gets a budget rather than the whole request deadline. Shippo stalls on individual carriers often enough to matter — rate shopping is the single slowest endpoint in the API — and a carrier that never answers used to hold the request until the caller's deadline killed it, throwing away the rates every other carrier had already returned. A carrier that runs out of budget is dropped exactly like one that errors.
	rateCtx, cancelRates := timeutil.BudgetedContext(ctx, timeutil.FanOutReserve)
	defer cancelRates()

	var allOptions []*domain.RateShopOption
	shippoRatesByCarrier := make([][]domain.ShippoRateOption, len(carriers))
	var wg sync.WaitGroup

	for i, carrier := range carriers {
		if shippoClient == nil || carrier.ShippoCarrierAccountID == nil || *carrier.ShippoCarrierAccountID == "" {
			continue
		}

		wg.Add(1)
		go func(i int, carrierAccountID string) {
			defer wg.Done()
			rates, apiErr := shippoClient.FetchAllShippingRates(rateCtx, domain.FetchAllShippingRatesParams{
				CarrierAccountObjectID: carrierAccountID,
				FromAddress:            params.FromAddress,
				ToAddress:              params.ToAddress,
				Parcels:                params.Parcels,
			})
			if apiErr != nil {
				// Skip carriers that fail to fetch rates, including those the budget above cut short.
				slog.WarnContext(ctx, "rate shop: carrier returned no rates", "carrier_account_id", carrierAccountID, "error", apiErr.Error())
				return
			}
			shippoRatesByCarrier[i] = rates
		}(i, *carrier.ShippoCarrierAccountID)
	}
	wg.Wait()

	for i, carrier := range carriers {
		if carrier.ShippoCarrierAccountID == nil || *carrier.ShippoCarrierAccountID == "" {
			// Non-Shippo carrier: include each option with rate 0.
			for _, opt := range carrier.ServiceLevels {
				allOptions = append(allOptions, &domain.RateShopOption{
					CarrierID:        carrier.ID,
					CarrierName:      carrier.Name,
					ServiceLevelID:   opt.ID,
					ServiceLevelName: opt.Name,
					Rate:             0,
				})
			}
			continue
		}

		// Carrier is Shippo-configured but the account has no live Shippo integration: contribute no options, mirroring Dashboard's fetchAllShippoRates returning an empty list (a Shippo carrier is never surfaced at a fabricated rate of 0).
		if shippoClient == nil {
			continue
		}

		// Map Shippo rates to carrier options by matching service level token.
		for _, shippoRate := range shippoRatesByCarrier[i] {
			for _, opt := range carrier.ServiceLevels {
				if opt.ServiceLevelToken != nil && *opt.ServiceLevelToken == shippoRate.ServiceLevelToken {
					allOptions = append(allOptions, &domain.RateShopOption{
						CarrierID:        carrier.ID,
						CarrierName:      carrier.Name,
						ServiceLevelID:   opt.ID,
						ServiceLevelName: opt.Name,
						Rate:             shippoRate.Amount,
						EstimatedDays:    shippoRate.EstimatedDays,
					})
					break
				}
			}
		}
	}

	// 7. Post-process rates: apply flat rate, minimum order, and free shipping rules.
	for _, opt := range allOptions {
		isEligibleForFreeShipping := true
		if hasFreeShippingRules {
			isEligibleForFreeShipping = freeShippingOptionIDs[opt.ServiceLevelID]
		}

		if isMinimumOrderMet && isEligibleForFreeShipping {
			opt.Rate = 0
		} else if hasFlatRate {
			opt.Rate = *flatRateValue
		}
	}

	// 8. Sort by rate ascending.
	sort.Slice(allOptions, func(i, j int) bool {
		return allOptions[i].Rate < allOptions[j].Rate
	})

	// 9. Determine exemption type.
	var exemptionType *string
	if isMinimumOrderMet {
		exemptionType = new("minimum_order_met")
	} else if hasFlatRate {
		exemptionType = new("flat_rate")
	} else {
		exemptionType = new("none")
	}

	result := &domain.RateShopResult{
		Options:       allOptions,
		ExemptionType: exemptionType,
	}
	if hasFlatRate {
		result.FlatRate = flatRateValue
	}

	return result, nil
}

// checkShipmentReadPermission checks the appropriate read permission based on the identity context. Internal actors need shipments:read for their own account, or customers:read / suppliers:read for external accounts.
func checkShipmentReadPermission(identity *types.Identity) *apierror.APIError {
	if !identity.IsInternalActor() {
		return nil
	}
	if identity.IsTargetCustomerAccount() {
		return identity.CheckHasPermission(types.PermissionDomainCustomers, types.ActionRead)
	}
	if identity.IsTargetSupplierAccount() {
		return identity.CheckHasPermission(types.PermissionDomainSuppliers, types.ActionRead)
	}
	return identity.CheckHasPermission(types.PermissionDomainShipments, types.ActionRead)
}

// Rejects re-routing a shipment that has already left. The carrier and service level are what the
// purchased label was bought against, so changing them after the fact describes a shipment that
// does not exist; correcting the tracking number, note or number stays open.
func checkShipmentRoutingStillMutable(old *domain.Shipment, params domain.UpdateShipmentParams) *apierror.APIError {
	if old.ShippedAt == nil {
		return nil
	}
	if params.CarrierID != nil && *params.CarrierID != old.CarrierID {
		return apierror.NewConflictErrorWithParam("Cannot change the carrier of a shipped shipment.", "carrier_id")
	}
	if params.ServiceLevelID.WasProvided() && !equalStringPtr(params.ServiceLevelID.ValuePtr(), old.ServiceLevelID) {
		return apierror.NewConflictErrorWithParam("Cannot change the service level of a shipped shipment.", "service_level_id")
	}
	return nil
}

func equalStringPtr(a, b *string) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

// Creates the invoice a shipment bills for and advances the order's fulfillment state, inside the
// caller's ship transaction. Mirrors legacy's post-ship chain: invoice number is the shipment
// number, first-ship is stamped, and the order is marked fulfilled once every sale line is invoiced.
// Draws the shipped goods against the order's reservations, flipping each reserved issue to open and
// allocating it FIFO across receipts. Consuming the shipped qty leaves a partial shipment's balance reserved.
func (s *shipmentSvcImpl) allocateInventoryOnShip(txCtx context.Context, shipment *domain.Shipment, shipmentLines []*domain.ShipmentLine) *apierror.APIError {
	reservationRepo := s.repos.NewInventoryReservationRepo()

	for _, line := range shipmentLines {
		if line.OrderLineItemID == nil || *line.OrderLineItemID == "" {
			continue
		}
		measure := parseDecimalOrZero(line.QuantityValue)
		if !measure.IsPositive() {
			continue
		}

		// A shortfall means stock was never reserved for this line; the shipment still stands, so
		// the uncovered quantity is left for the inventory reconciliation rather than failing here.
		if _, apiErr := reservationRepo.AllocateReservationsForConsumption(txCtx, domain.ConsumptionAllocationParams{
			OrderID:   shipment.SalesOrderID,
			AccountID: shipment.AccountID,
			ItemID:    *line.OrderLineItemID,
			Measure:   measure,
			UnitID:    line.QuantityUnitID,
		}); apiErr != nil {
			return apiErr
		}
	}

	return nil
}

// Puts the shipped goods back on the order's reservation, unwinding each line's consumption newest
// first. Re-derives from the still-open issues, so a replayed void finds nothing left to reverse.
func (s *shipmentSvcImpl) reverseInventoryOnVoid(txCtx context.Context, shipment *domain.Shipment) *apierror.APIError {
	shipmentLines, apiErr := s.repos.NewShipmentLineRepo().ListByShipment(txCtx, shipment.ID)
	if apiErr != nil {
		return apiErr
	}

	mutationRepo := s.repos.NewInventoryMutationRepo()
	reversedUnits := make(map[string]string, len(shipmentLines))
	reversedMeasures := make(map[string]decimal.Decimal, len(shipmentLines))

	for _, line := range shipmentLines {
		if line.OrderLineItemID == nil || *line.OrderLineItemID == "" {
			continue
		}
		measure := parseDecimalOrZero(line.QuantityValue)
		if !measure.IsPositive() {
			continue
		}

		if apiErr := mutationRepo.ReverseInventoryForOrderItem(txCtx, shipment.AccountID, shipment.SalesOrderID, *line.OrderLineItemID, measure); apiErr != nil {
			return apiErr
		}

		itemID := *line.OrderLineItemID
		reversedUnits[itemID] = line.QuantityUnitID
		reversedMeasures[itemID] = reversedMeasures[itemID].Add(measure)
	}

	// Receipts the reversal released can now cover issues that were short, so allocation is asked for
	// again for whatever it touched — asked for, not done here: covering the demand inline meant
	// walking every open issue of every reversed item while holding this transaction's receipt locks,
	// in the opposite order from the consumer doing the same work.
	//
	// The item ids are sorted rather than ranged off the map, whose iteration order is randomized per
	// run. That only orders the outbox rows now, but it is the same set of ids the ledger work will
	// take locks on, and a set taken in two different orders is a deadlock nobody can reproduce.
	itemIDs := make([]string, 0, len(reversedUnits))
	for itemID := range reversedUnits {
		itemIDs = append(itemIDs, itemID)
	}
	itemIDs = mediator.SortedUniqueIDs(itemIDs)

	if apiErr := mediator.EnqueueAllocateOpenIssues(txCtx, s.repos, shipment.AccountID, itemIDs...); apiErr != nil {
		return apiErr
	}

	for _, itemID := range itemIDs {
		mediator.RecordInventoryAuditTrailOrLog(
			txCtx,
			s.repos,
			shipment.AccountID,
			itemID,
			reversedMeasures[itemID],
			reversedUnits[itemID],
			string(constants.InventoryActionTypeUserCorrection),
			nil,
			nil,
		)
	}

	return nil
}

func (s *shipmentSvcImpl) createInvoiceAndStampOrderOnShip(txCtx context.Context, shipment *domain.Shipment, emailCustomer bool, logo ackLogo) *apierror.APIError {
	lineRepo := s.repos.NewShipmentLineRepo()
	shipmentLines, apiErr := lineRepo.ListByShipment(txCtx, shipment.ID)
	if apiErr != nil {
		return apiErr
	}

	if apiErr := s.allocateInventoryOnShip(txCtx, shipment, shipmentLines); apiErr != nil {
		return apiErr
	}

	drafts := make([]domain.InvoiceLineDraft, 0, len(shipmentLines))
	for _, l := range shipmentLines {
		drafts = append(drafts, domain.InvoiceLineDraft{
			SalesOrderLineID: l.SalesOrderLineID,
			QuantityValue:    l.QuantityValue,
			QuantityUnitID:   l.QuantityUnitID,
		})
	}

	invoiceRepo := s.repos.NewInvoiceRepo()
	isDuplicate, apiErr := invoiceRepo.IsDuplicateNumber(txCtx, shipment.AccountID, shipment.Number)
	if apiErr != nil {
		return apiErr
	}
	if isDuplicate {
		return apierror.NewResourceConflictError("An invoice already exists for this shipment number.")
	}

	invoiceID, apiErr := id.GenID(id.InvoiceIDPrefix, nil)
	if apiErr != nil {
		return apiErr
	}
	if _, apiErr := invoiceRepo.CreateFromShipment(txCtx, domain.CreateInvoiceFromShipmentParams{
		AccountID:    shipment.AccountID,
		InvoiceID:    invoiceID,
		Number:       shipment.Number,
		SalesOrderID: shipment.SalesOrderID,
		ShippedLines: drafts,
	}); apiErr != nil {
		return apiErr
	}

	// Shipping is the only path that raises an invoice, so this is where its create event belongs.
	if apiErr := audit.NewPublisher().Publish(txCtx, s.repos.NewOutboxRepo(), audit.EventData{
		ServiceName:      domain.ServiceName,
		Action:           constants.AuditActionCreate,
		ResourceType:     constants.ObjectTypeInvoice,
		ResourceID:       invoiceID,
		RootResourceType: constants.ObjectTypeSalesOrder,
		RootResourceID:   shipment.SalesOrderID,
	}); apiErr != nil {
		return apiErr
	}

	s.meterInvoiceCreated(txCtx, shipment.AccountID, invoiceID)

	// Link the shipment to its invoice so void (which finds it via shipment.invoice_id) can delete it.
	if apiErr := s.repos.NewShipmentRepo().LinkInvoice(txCtx, shipment.AccountID, shipment.ID, invoiceID); apiErr != nil {
		return apiErr
	}

	salesOrderRepo := s.repos.NewSalesOrderRepo()
	if apiErr := salesOrderRepo.NoteFirstShipAt(txCtx, shipment.AccountID, shipment.SalesOrderID); apiErr != nil {
		return apiErr
	}

	// The order is fulfilled once every sale line is fully invoiced — the invoice just written is
	// counted, so this reads the post-invoice state.
	progress, apiErr := salesOrderRepo.GetFulfillmentProgress(txCtx, []string{shipment.SalesOrderID})
	if apiErr != nil {
		return apiErr
	}
	if p, ok := progress[shipment.SalesOrderID]; ok && p.InvoicedCompletion >= 1.0 {
		if apiErr := salesOrderRepo.MarkFulfilled(txCtx, shipment.AccountID, shipment.SalesOrderID); apiErr != nil {
			return apiErr
		}
	}

	// The invoice document backs both the PDF and the customer email, so assemble it once. A render
	// failure degrades to an attachment-free email rather than failing the ship.
	doc, attachment := s.buildInvoiceDocument(txCtx, shipment.AccountID, invoiceID, logo)

	// The sales rep is notified on every ship, independent of email_customer (legacy postShipActions
	// always emails the rep); the customer receives it only when asked.
	if apiErr := s.emailSalesRepOnShip(txCtx, shipment, doc, attachment); apiErr != nil {
		return apiErr
	}
	if emailCustomer {
		if apiErr := s.emailCustomerInvoiceOnShip(txCtx, shipment, invoiceID, doc, attachment); apiErr != nil {
			return apiErr
		}
	}

	return nil
}

// Meters a created invoice for usage billing, best effort: metering must never fail a ship (legacy
// swallows the reporting error). The command rides the outbox, so a rolled-back ship never meters.
func (s *shipmentSvcImpl) meterInvoiceCreated(txCtx context.Context, accountID, invoiceID string) {
	if s.billingPub == nil {
		return
	}
	// The outbox publisher reads the RepoFactory from the context; inject the transaction's factory
	// so the command commits with the invoice.
	if apiErr := s.billingPub.PublishReportInvoiceCreated(event.WithRepos(txCtx, s.repos), accountID, invoiceID); apiErr != nil {
		slog.WarnContext(txCtx, "invoice usage metering failed; shipping anyway",
			"account_id", accountID, "invoice_id", invoiceID, "error", apiErr.Error())
	}
}

// Renders the invoice PDF and base64-encodes it for email attachment, or returns nil on any failure
// — the email still goes out, just without the document, matching the acknowledgement's best-effort.
func (s *shipmentSvcImpl) buildInvoiceDocument(txCtx context.Context, accountID, invoiceID string, logo ackLogo) (invoiceDoc, *string) {
	invoice, apiErr := s.repos.NewInvoiceRepo().Get(txCtx, domain.GetInvoiceParams{AccountID: accountID, InvoiceID: invoiceID})
	if apiErr != nil {
		return invoiceDoc{}, nil
	}
	lines, apiErr := s.repos.NewInvoiceRepo().GetLines(txCtx, invoiceID)
	if apiErr != nil {
		return invoiceDoc{}, nil
	}

	doc := gatherInvoiceDoc(txCtx, s.repos, accountID, invoice, lines)
	doc.Header.OrderOnlineLink = portalRegisterLink(txCtx, s.repos, s.frontendURL, accountID)
	// Fetched before the transaction opened, because embedding needs the bytes and a stalled logo
	// host must not hold the ship's row locks.
	doc.Header.LogoImageType, doc.Header.LogoImage = logo.ImageType, logo.Image

	pdfBytes, err := buildInvoicePDF(doc)
	if err != nil {
		return doc, nil
	}
	encoded := base64.StdEncoding.EncodeToString(pdfBytes)
	return doc, &encoded
}

// Emails the customer the invoice and flags it sent. Gated on email_customer by the caller.
func (s *shipmentSvcImpl) emailCustomerInvoiceOnShip(txCtx context.Context, shipment *domain.Shipment, invoiceID string, doc invoiceDoc, attachment *string) *apierror.APIError {
	accountID := shipment.AccountID
	recipients, apiErr := s.repos.NewInvoiceRepo().GetEmailRecipients(txCtx, invoiceID)
	if apiErr != nil {
		return apiErr
	}
	if apiErr := s.publishInvoiceEmail(txCtx, accountID, shipment, doc, recipients, attachment); apiErr != nil {
		return apiErr
	}
	if len(recipients) == 0 {
		return nil
	}
	return s.repos.NewInvoiceRepo().MarkEmailSent(txCtx, accountID, invoiceID)
}

// Notifies the order's sales rep of the shipment's invoice. Never flags the invoice sent — that
// tracks whether the customer received it, and the rep copy is an internal notification.
func (s *shipmentSvcImpl) emailSalesRepOnShip(txCtx context.Context, shipment *domain.Shipment, doc invoiceDoc, attachment *string) *apierror.APIError {
	email, apiErr := s.repos.NewSalesOrderRepo().GetSalesRepEmail(txCtx, shipment.AccountID, shipment.SalesOrderID)
	if apiErr != nil {
		return apiErr
	}
	if email == nil {
		return nil
	}
	return s.publishInvoiceEmail(txCtx, shipment.AccountID, shipment, doc, []string{*email}, attachment)
}

// Stages an invoice email in the outbox, atomically with the invoice it bills. No-op on an empty
// recipient list. The send itself is async, so a downstream email failure never fails the ship.
func (s *shipmentSvcImpl) publishInvoiceEmail(txCtx context.Context, accountID string, shipment *domain.Shipment, doc invoiceDoc, recipients []string, attachment *string) *apierror.APIError {
	if s.notificationPub == nil || len(recipients) == 0 {
		return nil
	}

	invoiceNumber := shipment.Number
	params := doc.emailParams(shipmentMasterTrackingURL(shipment))
	// The document falls back to a blank header when its lookups fail, so keep the account name and
	// invoice number truthful even then.
	if params["account_name"] == "" {
		accountName, apiErr := s.repos.NewAccountRepo().GetName(txCtx, accountID)
		if apiErr != nil {
			return apiErr
		}
		params["account_name"] = accountName
	}
	if params["invoice_number"] == "" {
		params["invoice_number"] = textutil.FormatRecordNumber(invoiceNumber)
	}

	emailData := messaging.EmailSendData{
		To:         recipients,
		Subject:    fmt.Sprintf("Invoice %s", textutil.FormatRecordNumber(invoiceNumber)),
		TemplateID: constants.EmailTemplateInvoice,
		Params:     params,
		AccountID:  &accountID,
	}
	if attachment != nil {
		filename := fmt.Sprintf("invoice-%s.pdf", invoiceNumber)
		contentType := "application/pdf"
		emailData.AttachmentData = attachment
		emailData.AttachmentFilename = &filename
		emailData.AttachmentContentType = &contentType
	}

	pubCtx := event.WithRepos(txCtx, s.repos)
	return s.notificationPub.PublishSendEmail(pubCtx, emailData)
}

// Resolves the shipment's master tracking for the ship action: sandbox accounts get a deterministic
// placeholder to persist, real accounts buy carrier labels (which persist tracking themselves).
func (s *shipmentSvcImpl) resolveShipmentTracking(ctx context.Context, shipment *domain.Shipment, idempotencyTypeID string) (*string, *apierror.APIError) {
	accountCtx, apiErr := s.repos.NewAccountRepo().GetAccountContext(ctx, shipment.AccountID)
	if apiErr != nil {
		return nil, apiErr
	}
	if accountCtx.IsSandbox {
		tracking := sandboxTrackingNumber(shipment.ID)
		return &tracking, nil
	}
	return s.purchaseShippingLabels(ctx, shipment, idempotencyTypeID)
}

// Buys the shipment's carrier labels, persisting per-case tracking/label, master tracking and freight cost.
// Returns tracking when nothing was bought; nil once RecoveryPointShipLabelsCreated is staged against a re-buy.
func (s *shipmentSvcImpl) purchaseShippingLabels(ctx context.Context, shipment *domain.Shipment, idempotencyTypeID string) (*string, *apierror.APIError) {
	if s.shippoFactory == nil {
		return shipment.MasterTrackingNumber, nil
	}

	// A label is bought against a Shippo carrier account at a specific service level; without either
	// there is nothing to buy, matching legacy's "non-Shippo carriers don't generate labels".
	carrier, apiErr := s.repos.NewCarrierRepo().Get(ctx, domain.GetCarrierParams{AccountID: shipment.AccountID, CarrierID: shipment.CarrierID})
	if apiErr != nil {
		return nil, apiErr
	}
	if carrier.ShippoCarrierAccountID == nil || *carrier.ShippoCarrierAccountID == "" {
		return shipment.MasterTrackingNumber, nil
	}
	if shipment.ServiceLevelToken == nil || *shipment.ServiceLevelToken == "" {
		return shipment.MasterTrackingNumber, nil
	}

	shippoClient, apiErr := s.buildShippoClient(ctx, shipment.AccountID)
	if apiErr != nil {
		return nil, apiErr
	}
	if shippoClient == nil {
		return shipment.MasterTrackingNumber, nil
	}

	cases, apiErr := s.repos.NewShippingCaseRepo().ListByShipment(ctx, shipment.ID)
	if apiErr != nil {
		return nil, apiErr
	}
	if len(cases) == 0 {
		return shipment.MasterTrackingNumber, nil
	}

	// Ship-from is the account's configured origin (its default billing address). Refuse to buy a
	// label from an empty origin rather than printing one the carrier will reject.
	var from domain.ShippingAddress
	if origin, apiErr := s.repos.NewSalesOrderRepo().GetAccountOriginAddress(ctx, shipment.AccountID); apiErr != nil {
		return nil, apiErr
	} else if origin != nil {
		from = *origin
	}
	if from.Zip == "" || from.Country == "" {
		return nil, apierror.NewValidationError("Cannot buy a shipping label: the account has no default billing (ship-from) address.")
	}

	result, apiErr := shippoClient.CreateTransactionInstantLabel(ctx, domain.CreateLabelParams{
		CarrierAccountObjectID: *carrier.ShippoCarrierAccountID,
		ServiceLevelToken:      *shipment.ServiceLevelToken,
		FromAddress:            from,
		ToAddress:              shipmentToAddress(shipment),
		Parcels:                shippingCaseParcels(cases),
		Billing:                shipmentThirdPartyBilling(shipment, from),
	})
	if apiErr != nil {
		return nil, apiErr
	}
	// A result with no labels means no purchase happened (the stub client in test mode), so there is
	// nothing to persist and nothing to guard against re-buying.
	if len(result.Packages) == 0 {
		return shipment.MasterTrackingNumber, nil
	}

	caseRepo := s.repos.NewShippingCaseRepo()
	for i, c := range cases {
		if i >= len(result.Packages) {
			break
		}
		pkg := result.Packages[i]
		s.storeShippingLabel(ctx, shipment.AccountID, c.Number, pkg.LabelURL)
		if apiErr := caseRepo.UpdateWithShipmentInfo(ctx, c.ID, pkg.TrackingNumber, pkg.ShippoTransactionID, pkg.LabelURL); apiErr != nil {
			return nil, apiErr
		}
	}

	if result.MasterTrackingNumber != "" {
		if apiErr := s.repos.NewShipmentRepo().SetMasterTracking(ctx, shipment.AccountID, shipment.ID, result.MasterTrackingNumber); apiErr != nil {
			return nil, apiErr
		}
	}

	if apiErr := s.writeBackNegotiatedRate(ctx, shipment, result.NegotiatedRate); apiErr != nil {
		return nil, apiErr
	}

	if apiErr := s.repos.NewIdempotencyKeyRepo().AdvanceRecoveryPoint(ctx, idempotencyTypeID, domain.RecoveryPointShipLabelsCreated); apiErr != nil {
		return nil, apiErr
	}

	return nil, nil
}

// Bounds the pull of a carrier-hosted label: it is a remote host on the ship path, so it may neither
// hang the request nor stream an unbounded body into memory.
const (
	shippingLabelFetchTimeout = 15 * time.Second
	shippingLabelMaxBytes     = 10 << 20
	shippingLabelContentType  = "image/gif"
)

var shippingLabelHTTPClient = &http.Client{Timeout: shippingLabelFetchTimeout}

// Copies a purchased label into the shipping-labels bucket, where it outlives the carrier's own URL.
// Best-effort: the label is bought and the shipment real, so a failure only leaves that URL as the fallback.
func (s *shipmentSvcImpl) storeShippingLabel(ctx context.Context, accountID, caseNumber, labelURL string) {
	if s.s3Client == nil || s.shippingLabelsBucket == "" || labelURL == "" {
		return
	}

	key := shippingLabelS3Key(accountID, caseNumber)

	label, err := fetchShippingLabel(ctx, labelURL)
	if err != nil {
		slog.WarnContext(ctx, "shipping label fetch failed; shipping anyway",
			"account_id", accountID, "s3_key", key, "error", err.Error())
		return
	}

	if apiErr := s.s3Client.Upload(ctx, s.shippingLabelsBucket, key, bytes.NewReader(label), shippingLabelContentType); apiErr != nil {
		slog.WarnContext(ctx, "shipping label upload failed; shipping anyway",
			"account_id", accountID, "s3_key", key, "error", apiErr.Error())
	}
}

// Reads a carrier-hosted label into memory under a size cap, so an oversized or wrong URL cannot
// exhaust the process.
func fetchShippingLabel(ctx context.Context, labelURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, labelURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := shippingLabelHTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("label fetch returned status %d", resp.StatusCode)
	}

	label, err := io.ReadAll(io.LimitReader(resp.Body, shippingLabelMaxBytes+1))
	if err != nil {
		return nil, err
	}
	if len(label) > shippingLabelMaxBytes {
		return nil, fmt.Errorf("label exceeds the %d byte cap", shippingLabelMaxBytes)
	}

	return label, nil
}

// Records what the carrier actually charged on the order's freight line cost, mirroring legacy's
// updateShippingCost: the line is created when missing, and a zero rate is written so a stale cost clears.
func (s *shipmentSvcImpl) writeBackNegotiatedRate(ctx context.Context, shipment *domain.Shipment, rate float64) *apierror.APIError {
	shippingLine, apiErr := findOrAddFreightLine(ctx, s.repos, shipment.AccountID, shipment.SalesOrderID)
	if apiErr != nil {
		return apiErr
	}
	// An account with no shipping system product has nowhere to record the cost.
	if shippingLine == nil {
		return nil
	}

	// The cost rate carries the same currency-per-shipping-unit units as the line's price.
	value := decimal.NewFromFloat(rate).Round(2).String()
	_, apiErr = s.repos.NewSalesOrderLineRepo().Update(ctx, domain.UpdateSalesOrderLineParams{
		SalesOrderLineID:          shippingLine.ID,
		SalesOrderID:              shipment.SalesOrderID,
		AccountID:                 shipment.AccountID,
		UnitCostValue:             &value,
		UnitCostNumeratorUnitID:   &shippingLine.UnitPriceNumeratorUnitID,
		UnitCostDenominatorUnitID: &shippingLine.UnitPriceDenominatorUnitID,
	})
	return apiErr
}

// Builds a Shippo client from the account's stored integration credentials. Returns (nil, nil) when
// the account has no Shippo integration, so callers can skip the carrier round-trip.
func (s *shipmentSvcImpl) buildShippoClient(ctx context.Context, accountID string) (domain.ShippoClient, *apierror.APIError) {
	if s.shippoFactory == nil {
		return nil, nil
	}

	integrationRepo := s.repos.NewAccountIntegrationRepo()
	hasIntegration, apiErr := integrationRepo.HasIntegration(ctx, accountID, constants.IntegrationCodeShippo)
	if apiErr != nil {
		return nil, apiErr
	}
	if !hasIntegration {
		return nil, nil
	}

	encryptedCreds, _, apiErr := integrationRepo.GetEncryptedCredentials(ctx, accountID, constants.IntegrationCodeShippo)
	if apiErr != nil {
		return nil, apiErr
	}
	apiKey, apiErr := decryptShippoAPIKey(encryptedCreds, s.encryptionKey, accountID)
	if apiErr != nil {
		return nil, apiErr
	}

	return s.shippoFactory.Build(apiKey), nil
}

// Names the account's system freight product, whose order line carries the shipping charge.
const systemProductCodeShipping = "shipping"

// Nominal case dimensions used for rating and labels; only the weight varies per case.
const (
	shippingCaseLength = "23.5"
	shippingCaseWidth  = "13"
	shippingCaseHeight = "9.5"
)

// Turns the shipment's cases into carrier parcels, one per case and in case order so the purchased
// labels come back aligned with them.
func shippingCaseParcels(cases []*domain.ShippingCase) []domain.Parcel {
	parcels := make([]domain.Parcel, len(cases))
	for i, c := range cases {
		parcels[i] = domain.Parcel{
			Weight: c.FreightWeightValue,
			Length: shippingCaseLength,
			Width:  shippingCaseWidth,
			Height: shippingCaseHeight,
		}
	}
	return parcels
}

// Reads the shipment's ship-to into the address shape the carrier prints.
func shipmentToAddress(shipment *domain.Shipment) domain.ShippingAddress {
	return domain.ShippingAddress{
		Name:    ptrutil.Deref(shipment.ShippingAddressName),
		Street1: ptrutil.Deref(shipment.ShippingAddressStreetLine1),
		Street2: shipment.ShippingAddressStreetLine2,
		City:    ptrutil.Deref(shipment.ShippingAddressLocality),
		State:   ptrutil.Deref(shipment.ShippingAddressState),
		Zip:     ptrutil.Deref(shipment.ShippingAddressPostalCode),
		Country: ptrutil.Deref(shipment.ShippingAddressCountry),
		Phone:   shipment.ShippingAddressPhone,
		Email:   shipment.ShippingAddressEmail,
	}
}

// Bills freight to the third party named on the shipment, passing the seller's origin country and zip
// through as the billing address (matching Dashboard's createShippingLine). Nil when not third-party billed.
func shipmentThirdPartyBilling(shipment *domain.Shipment, origin domain.ShippingAddress) *domain.ShippingBilling {
	if shipment.CarrierBillingType == nil || *shipment.CarrierBillingType != string(constants.CarrierBillingTypeThirdParty) {
		return nil
	}
	return &domain.ShippingBilling{
		Type:    "THIRD_PARTY",
		Account: ptrutil.Deref(shipment.CarrierBillingAccount),
		Country: origin.Country,
		Zip:     origin.Zip,
	}
}

// Derives a stable sandbox tracking number from the shipment id, so a retried ship yields the same
// value rather than a new one each attempt.
func sandboxTrackingNumber(shipmentID string) string {
	suffix := shipmentID
	if len(suffix) > 8 {
		suffix = suffix[len(suffix)-8:]
	}
	return "SANDBOX-" + strings.ToUpper(suffix)
}

// Refunds the cases' purchased labels and drops their stored files before void clears them; sandbox no-ops.
// Best-effort, as in legacy: a carrier refusing a refund must not strand a shipment in shipped state.
func (s *shipmentSvcImpl) refundShippingLabels(ctx context.Context, shipment *domain.Shipment, idempotencyTypeID string) *apierror.APIError {
	accountCtx, apiErr := s.repos.NewAccountRepo().GetAccountContext(ctx, shipment.AccountID)
	if apiErr != nil {
		return apiErr
	}
	if accountCtx.IsSandbox {
		return nil
	}

	cases, apiErr := s.repos.NewShippingCaseRepo().ListByShipment(ctx, shipment.ID)
	if apiErr != nil {
		return apiErr
	}

	s.refundShippoTransactions(ctx, shipment.AccountID, cases)
	s.deleteStoredShippingLabels(ctx, shipment.AccountID, cases)

	return s.repos.NewIdempotencyKeyRepo().AdvanceRecoveryPoint(ctx, idempotencyTypeID, domain.RecoveryPointVoidLabelsRefunded)
}

// Refunds each case's purchased Shippo transaction, logging and continuing past any that fails.
func (s *shipmentSvcImpl) refundShippoTransactions(ctx context.Context, accountID string, cases []*domain.ShippingCase) {
	var transactionIDs []string
	for _, c := range cases {
		if c.ShippoTransactionID != nil && *c.ShippoTransactionID != "" {
			transactionIDs = append(transactionIDs, *c.ShippoTransactionID)
		}
	}
	if len(transactionIDs) == 0 {
		return
	}

	shippoClient, apiErr := s.buildShippoClient(ctx, accountID)
	if apiErr != nil {
		slog.WarnContext(ctx, "could not build shippo client to refund shipping labels; voiding anyway",
			"account_id", accountID, "error", apiErr.Error())
		return
	}
	if shippoClient == nil {
		return
	}

	for _, transactionID := range transactionIDs {
		if apiErr := shippoClient.RefundTransaction(ctx, transactionID); apiErr != nil {
			slog.WarnContext(ctx, "shippo label refund failed; voiding anyway",
				"account_id", accountID, "shippo_transaction_id", transactionID, "error", apiErr.Error())
		}
	}
}

// Removes each case's stored label object, logging and continuing past any that fails.
func (s *shipmentSvcImpl) deleteStoredShippingLabels(ctx context.Context, accountID string, cases []*domain.ShippingCase) {
	if s.s3Client == nil || s.shippingLabelsBucket == "" {
		return
	}
	for _, c := range cases {
		key := shippingLabelS3Key(accountID, c.Number)
		if apiErr := s.s3Client.Delete(ctx, s.shippingLabelsBucket, key); apiErr != nil {
			slog.WarnContext(ctx, "shipping label delete failed; voiding anyway",
				"account_id", accountID, "s3_key", key, "error", apiErr.Error())
		}
	}
}
