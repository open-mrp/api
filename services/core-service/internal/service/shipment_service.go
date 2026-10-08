package service

import (
	"bytes"
	"context"
	"crypto/rand"
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
	"github.com/open-mrp/api/services/core-service/internal/ledgerlock"
	"github.com/open-mrp/api/services/core-service/internal/mediator"
	"github.com/open-mrp/api/shared/appctx"
	"github.com/open-mrp/api/shared/audit"
	s3client "github.com/open-mrp/api/shared/cloud/s3"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/crypto"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/field"
	"github.com/open-mrp/api/shared/id"
	"github.com/open-mrp/api/shared/idempotency"
	"github.com/open-mrp/api/shared/lease"
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
	portalURL            string
	branding             BrandingAssets
	dispatchLeases       lease.Repo
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

	// PortalURL (optional; default: "") is the customer portal base URL behind the invoice email's
	// order-online link when the merchant has no verified custom domain. Not validated; an empty URL drops the link.
	PortalURL string

	// Branding (optional) resolves the merchant logo for the invoice PDF letterhead. Omitted, it falls back to a text-only letterhead.
	Branding BrandingAssets

	// DispatchLeases (required) holds the per-shipment claim that keeps a ship, void or delete from
	// running alongside another on the same shipment.
	DispatchLeases lease.Repo
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
	if c.DispatchLeases == nil {
		return fmt.Errorf("shipment service: dispatch leases are required")
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
		portalURL:            config.PortalURL,
		branding:             config.Branding,
		dispatchLeases:       config.DispatchLeases,
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
			portalURL:            s.portalURL,
			dispatchLeases:       s.dispatchLeases,
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

	// A portal retrieves only the shipments of orders it bought, unless a request it was allowed to make includes another.
	if own := identity.PortalAccountID(); own != nil && !identity.IsIncludeRead() && shipment.CustomerID != *own {
		return nil, tracing.Trace(span, apierror.NewResourceNotFoundError("Shipment not found."))
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
			if apiErr := txSvc.checkShipmentRoutingInAccount(txCtx, params.AccountID, old, params.CarrierID, params.ServiceLevelID); apiErr != nil {
				return apiErr
			}
			if apiErr := checkUpdatedServiceLevelOnCarrier(txCtx, txSvc.repos, &old.CarrierID, old.ServiceLevelID, params.CarrierID, params.ServiceLevelID, "service_level_id"); apiErr != nil {
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

			if apiErr := txSvc.checkShipmentRoutingInAccount(txCtx, params.AccountID, old, params.CarrierID, params.ServiceLevelID); apiErr != nil {
				return apiErr
			}
			if apiErr := checkUpdatedServiceLevelOnCarrier(txCtx, txSvc.repos, &old.CarrierID, old.ServiceLevelID, params.CarrierID, params.ServiceLevelID, "service_level_id"); apiErr != nil {
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

// Rejects a carrier or service level the account cannot reach, before routing is rewritten; another
// tenant's ID 404s like an unknown one. The routing the shipment already has is not looked up again,
// so a carrier deleted since does not block editing the rest of the shipment.
func (s *shipmentSvcImpl) checkShipmentRoutingInAccount(txCtx context.Context, accountID string, old *domain.Shipment, carrierID *string, serviceLevelID field.Clearable[string]) *apierror.APIError {
	if carrierID != nil && *carrierID != old.CarrierID {
		if _, apiErr := s.repos.NewCarrierRepo().Get(txCtx, domain.GetCarrierParams{AccountID: accountID, CarrierID: *carrierID}); apiErr != nil {
			if apierror.IsNotFound(apiErr) {
				return apierror.NewResourceNotFoundError("No carrier found with the provided ID.").WithParam("carrier_id")
			}
			return apiErr
		}
	}
	if id, ok := serviceLevelID.Value(); ok && !equalStringPtr(&id, old.ServiceLevelID) {
		if _, apiErr := s.repos.NewServiceLevelRepo().Get(txCtx, accountID, id); apiErr != nil {
			if apierror.IsNotFound(apiErr) {
				return apierror.NewResourceNotFoundError("No service level found with the provided ID.").WithParam("service_level_id")
			}
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

	// Claimed first, so a ship cannot buy labels for cases this delete is about to remove.
	release, apiErr := s.claimDispatch(ctx, params.ShipmentID)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	defer release()

	shipmentRepo := s.repos.NewShipmentRepo()

	shipment, apiErr := shipmentRepo.Get(ctx, domain.GetShipmentParams{
		AccountID:  params.AccountID,
		ShipmentID: params.ShipmentID,
	})
	if apiErr != nil {
		if apierror.IsNotFound(apiErr) {
			wasDeleted, deletedCheckErr := s.repos.NewDeletedRecordRepo().ExistsInAccount(ctx, constants.DeletedRecordResourceTypeShipment, params.ShipmentID, params.AccountID)
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

	// A shipped shipment carries an invoice and possibly bought labels; void is what unwinds those.
	if shipment.StatusCode == string(constants.ShipmentStatusShipped) {
		return tracing.Trace(span, apierror.NewConflictErrorWithParam("A shipped shipment cannot be deleted; void it first.", "id"))
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

		if apiErr := txSvc.repos.NewDeletedRecordRepo().CreateInAccount(txCtx, constants.DeletedRecordResourceTypeShipment, shipment.ID, params.AccountID, shipment); apiErr != nil {
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

	case domain.RecoveryPointStarted, domain.RecoveryPointShipLabelsCreated:
		// Held across the label purchase and the atomic phase, so a second ship of this shipment —
		// another request, or this one replayed while it is still running — conflicts instead of buying again.
		release, apiErr := s.claimDispatch(ctx, params.ShipmentID)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		defer release()

		result, apiErr := s.shipClaimed(ctx, identity, params, idempotencyKey)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		return result, nil

	default:
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Unexpected recovery point: "+idempotencyKey.RecoveryPoint))
	}
}

// Ships a claimed shipment: checks, then the label purchase, then one transaction that ships and
// invoices it. Failures before anything is bought stay uncached, so a fixed request can reuse its key.
func (s *shipmentSvcImpl) shipClaimed(ctx context.Context, identity *types.Identity, params domain.ShipShipmentParams, idempotencyKey *domain.IdempotencyKey) (*domain.Shipment, *apierror.APIError) {
	meds := s.mediators()
	labelsBought := domain.RecoveryPoint(idempotencyKey.RecoveryPoint) == domain.RecoveryPointShipLabelsCreated

	shipment, apiErr := s.repos.NewShipmentRepo().Get(ctx, domain.GetShipmentParams{
		AccountID:  params.AccountID,
		ShipmentID: params.ShipmentID,
	})
	if apiErr != nil {
		return nil, apiErr
	}
	if shipment.StatusCode == string(constants.ShipmentStatusShipped) {
		return nil, apierror.NewConflictErrorWithParam("Shipment has already been shipped.", "id")
	}

	// Ship creates the invoice, so enforce the per-billing-period invoice limit here — matching
	// legacy's canCreateInvoice on ship. A ship resuming past its purchase already passed it.
	if !labelsBought {
		if apiErr := enforceInvoicesPerPeriodLimit(ctx, s.repos, params.AccountID); apiErr != nil {
			return nil, apiErr
		}
	}

	shippedByID, apiErr := s.resolveShippedByID(ctx, identity, params.AccountID)
	if apiErr != nil {
		return nil, apiErr
	}

	if apiErr := checkInvoiceNumberFree(ctx, s.repos, shipment); apiErr != nil {
		return nil, apiErr
	}

	labels := shipLabels{CostRecorded: labelsBought}
	if !labelsBought {
		purchase, planned, apiErr := s.planShipmentLabels(ctx, shipment)
		if apiErr != nil {
			return nil, apiErr
		}
		labels = planned
		if purchase != nil {
			labels, apiErr = s.buyShipmentLabels(ctx, shipment, purchase, idempotencyKey.TypeID)
			if apiErr != nil {
				return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, apiErr)
			}
		}
	}

	// The invoice PDF embeds the letterhead logo, so fetch its bytes here: inside the transaction
	// a stalled logo host would hold the ship's row locks for the length of the request.
	letterheadLogo := fetchAccountLogo(ctx, s.repos, s.branding, params.AccountID)

	var result *domain.Shipment
	apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *shipmentSvcImpl) *apierror.APIError {
		var apiErr *apierror.APIError
		result, apiErr = txSvc.markShippedAndInvoice(txCtx, shipment, shippedByID, labels, params, letterheadLogo)
		if apiErr != nil {
			return apiErr
		}
		return txSvc.mediators().Idempotency.CacheSuccessResponse(txCtx, idempotencyKey.TypeID, result)
	})
	if apiErr != nil {
		return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, apiErr)
	}

	return result, nil
}

// The ship's atomic phase: the shipped stamp, case SSCCs, tracking, invoice and freight cost commit
// together or not at all.
func (s *shipmentSvcImpl) markShippedAndInvoice(txCtx context.Context, shipment *domain.Shipment, shippedByID string, labels shipLabels, params domain.ShipShipmentParams, logo ackLogo) (*domain.Shipment, *apierror.APIError) {
	shipmentRepo := s.repos.NewShipmentRepo()
	caseRepo := s.repos.NewShippingCaseRepo()

	// First, so a ship that raced past the status check blocks on this row and then finds it shipped.
	if apiErr := shipmentRepo.MarkShipped(txCtx, params.AccountID, params.ShipmentID, shippedByID); apiErr != nil {
		return nil, apiErr
	}

	if apiErr := caseRepo.MarkShippedByShipment(txCtx, params.ShipmentID); apiErr != nil {
		return nil, apiErr
	}

	cases, apiErr := caseRepo.ListByShipment(txCtx, params.ShipmentID)
	if apiErr != nil {
		return nil, apiErr
	}
	for _, sc := range cases {
		if sc.SSCC != nil {
			continue
		}
		counter, apiErr := caseRepo.FindAndIncrementSsccCounter(txCtx, params.AccountID)
		if apiErr != nil {
			return nil, apiErr
		}
		if apiErr := caseRepo.AddSscc(txCtx, sc.ID, domain.GenerateSSCC(counter)); apiErr != nil {
			return nil, apiErr
		}
	}

	if labels.MasterTracking != nil {
		if apiErr := shipmentRepo.SetMasterTracking(txCtx, params.AccountID, params.ShipmentID, *labels.MasterTracking); apiErr != nil {
			return nil, apiErr
		}
	}

	if apiErr := s.createInvoiceAndStampOrderOnShip(txCtx, shipment, params.EmailCustomer, logo); apiErr != nil {
		return nil, apiErr
	}

	// Legacy records the carrier's charge on every ship; with no label bought it charged nothing.
	if !labels.CostRecorded {
		if apiErr := s.writeBackNegotiatedRate(txCtx, shipment, 0); apiErr != nil {
			return nil, apiErr
		}
	}

	updated, apiErr := shipmentRepo.Get(txCtx, domain.GetShipmentParams{
		AccountID:  params.AccountID,
		ShipmentID: params.ShipmentID,
	})
	if apiErr != nil {
		return nil, apiErr
	}

	if slices.Contains(params.Includes, "lines") {
		lines, apiErr := s.repos.NewShipmentLineRepo().ListByShipment(txCtx, params.ShipmentID)
		if apiErr != nil {
			return nil, apiErr
		}
		updated.Lines = lines
	}
	if slices.Contains(params.Includes, "shipping_cases") {
		cases, apiErr := caseRepo.ListByShipment(txCtx, params.ShipmentID)
		if apiErr != nil {
			return nil, apiErr
		}
		updated.ShippingCases = cases
	}

	if apiErr := audit.NewPublisher().Publish(txCtx, s.repos.NewOutboxRepo(), audit.EventData{
		ServiceName:      domain.ServiceName,
		Action:           constants.AuditActionUpdate,
		ResourceType:     constants.ObjectTypeShipment,
		ResourceID:       updated.ID,
		RootResourceType: constants.ObjectTypeSalesOrder,
		RootResourceID:   updated.SalesOrderID,
		Changes:          audit.ComputeChanges(shipment, updated),
	}); apiErr != nil {
		return nil, apiErr
	}

	return updated, nil
}

// Refuses a ship whose invoice number is already taken, before any label is bought for it.
func checkInvoiceNumberFree(ctx context.Context, repos domain.RepoFactory, shipment *domain.Shipment) *apierror.APIError {
	isDuplicate, apiErr := repos.NewInvoiceRepo().IsDuplicateNumber(ctx, shipment.AccountID, shipment.Number)
	if apiErr != nil {
		return apiErr
	}
	if isDuplicate {
		return apierror.NewResourceConflictError("An invoice already exists for this shipment number.")
	}
	return nil
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

	case domain.RecoveryPointStarted, domain.RecoveryPointVoidLabelsRefunded:
		release, apiErr := s.claimDispatch(ctx, params.ShipmentID)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		defer release()

		result, apiErr := s.voidClaimed(ctx, params, idempotencyKey)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		return result, nil

	default:
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Unexpected recovery point: "+idempotencyKey.RecoveryPoint))
	}
}

// Voids a claimed shipment: refunds its bought labels, then unwinds the ship in one transaction. A
// refund the carrier refuses stops the void before anything is cleared, uncached, so it can be retried.
func (s *shipmentSvcImpl) voidClaimed(ctx context.Context, params domain.VoidShipmentParams, idempotencyKey *domain.IdempotencyKey) (*domain.Shipment, *apierror.APIError) {
	meds := s.mediators()

	if domain.RecoveryPoint(idempotencyKey.RecoveryPoint) == domain.RecoveryPointStarted {
		shipment, apiErr := s.repos.NewShipmentRepo().Get(ctx, domain.GetShipmentParams{
			AccountID:  params.AccountID,
			ShipmentID: params.ShipmentID,
		})
		if apiErr != nil {
			return nil, apiErr
		}
		if shipment.StatusCode != string(constants.ShipmentStatusShipped) {
			return nil, apierror.NewConflictErrorWithParam("Shipment is not in shipped status.", "id")
		}

		if apiErr := s.refundShippingLabels(ctx, shipment, idempotencyKey.TypeID); apiErr != nil {
			return nil, apiErr
		}
	}

	// The items this void will hand back, resolved on the pool so their ordering roots can be the
	// transaction's first statements. Reading the lines inside the transaction and locking after the
	// reversal has already written receipts is the inversion, not the fix (Corollary A).
	voidLines, apiErr := s.repos.NewShipmentLineRepo().ListByShipment(ctx, params.ShipmentID)
	if apiErr != nil {
		return nil, apiErr
	}
	voidItemIDs := make([]string, 0, len(voidLines))
	for _, line := range voidLines {
		if line.OrderLineItemID != nil {
			voidItemIDs = append(voidItemIDs, *line.OrderLineItemID)
		}
	}

	var result *domain.Shipment
	apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *shipmentSvcImpl) *apierror.APIError {
		scope, apiErr := ledgerlock.Acquire(txCtx, txSvc.repos.NewInventoryMutationRepo(), voidItemIDs)
		if apiErr != nil {
			return apiErr
		}

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
			if apiErr := txSvc.reverseInventoryOnVoid(txCtx, scope, shipment); apiErr != nil {
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

	return result, nil
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

// estimateShippingRate computes the posted shipping rate for an order or shipment, mirroring Dashboard's estimatePostedShippingRate cascade: product-line freight exemption → customer/group freight exemption → shipping-term free/flat/min-order → carrier-without-Shippo → no Shippo integration → live Shippo rate (already marked up by the Shippo client). A nil shippoFactory short-circuits the live rate to 0. An inactive integration is refused and a carrier that quotes no rate is unavailable, never 0. It is shared by the shipment estimate-rate endpoint and sales-order shipping-line synthesis.
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

	shippoClient, apiErr := accountShippoClient(ctx, repos, shippoFactory, encryptionKey, params.AccountID)
	if apiErr != nil {
		return 0, apiErr
	}
	if shippoClient == nil {
		return 0, nil
	}

	// A live rate requires a real ship-from address. Refuse to quote from an empty
	// origin (matches legacy, which fails order create with "Bill to address not
	// found" when the seller account has no default bill-to) rather than sending an
	// empty from-address to Shippo and returning a meaningless rate.
	if params.FromAddress.Zip == "" || params.FromAddress.Country == "" {
		return 0, apierror.NewValidationError("Cannot estimate shipping: the account has no default billing (ship-from) address.")
	}

	rate, apiErr := shippoClient.FetchShippingRate(ctx, domain.FetchShippingRateParams{
		CarrierAccountObjectID: *carrier.ShippoCarrierAccountID,
		ServiceLevelToken:      serviceLevelToken,
		FromAddress:            params.FromAddress,
		ToAddress:              params.ToAddress,
		Parcels:                params.Parcels,
		Billing:                params.Billing,
		CachedOnly:             params.CachedOnly,
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

	carrierIDs := make([]string, len(carriersResult.Carriers))
	for i, carrier := range carriersResult.Carriers {
		carrierIDs[i] = carrier.ID
	}
	optionsByCarrier, apiErr := carrierRepo.ListOptionsByCarrierIDs(ctx, params.AccountID, carrierIDs)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	for _, carrier := range carriersResult.Carriers {
		carrier.ServiceLevels = optionsByCarrier[carrier.ID]
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
			encryptedCreds, isActive, apiErr := integrationRepo.GetEncryptedCredentials(ctx, params.AccountID, constants.IntegrationCodeShippo)
			if apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			// An inactive integration rates nothing, as legacy's rate shop dropped every carrier it refused.
			if isActive {
				apiKey, apiErr := decryptShippoAPIKey(encryptedCreds, s.encryptionKey, params.AccountID)
				if apiErr != nil {
					return nil, tracing.Trace(span, apiErr)
				}
				shippoClient, apiErr = s.shippoFactory.Build(apiKey)
				if apiErr != nil {
					return nil, tracing.Trace(span, apiErr)
				}
			}
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
	result.Carriers, result.ServiceLevels = rateShopReferences(carriers, allOptions)

	return result, nil
}

// rateShopReferences returns the carriers and service levels the options name. They are part of the quote, so whoever may rate shop sees them without carriers:read.
func rateShopReferences(carriers []*domain.Carrier, options []*domain.RateShopOption) ([]*domain.Carrier, []*domain.ServiceLevel) {
	carrierByID := make(map[string]*domain.Carrier, len(carriers))
	levelByID := map[string]*domain.ServiceLevel{}
	for _, c := range carriers {
		carrierByID[c.ID] = c
		for _, sl := range c.ServiceLevels {
			levelByID[sl.ID] = sl
		}
	}

	var outCarriers []*domain.Carrier
	var outLevels []*domain.ServiceLevel
	seenCarrier := map[string]bool{}
	seenLevel := map[string]bool{}
	for _, o := range options {
		if c, ok := carrierByID[o.CarrierID]; ok && !seenCarrier[c.ID] {
			seenCarrier[c.ID] = true
			// The options already name the service levels; the carrier's own list would repeat them.
			record := *c
			record.ServiceLevels = nil
			outCarriers = append(outCarriers, &record)
		}
		if sl, ok := levelByID[o.ServiceLevelID]; ok && !seenLevel[sl.ID] {
			seenLevel[sl.ID] = true
			outLevels = append(outLevels, sl)
		}
	}
	return outCarriers, outLevels
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

	// The lines are already in hand, so the whole item set is known before any of it is written. The
	// loop below then walks them in whatever order the slice holds, which no longer matters.
	itemIDs := make([]string, 0, len(shipmentLines))
	for _, line := range shipmentLines {
		if line.OrderLineItemID != nil {
			itemIDs = append(itemIDs, *line.OrderLineItemID)
		}
	}
	scope, apiErr := ledgerlock.Acquire(txCtx, reservationRepo, itemIDs)
	if apiErr != nil {
		return apiErr
	}

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
		if _, apiErr := reservationRepo.AllocateReservationsForConsumption(txCtx, scope, domain.ConsumptionAllocationParams{
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
func (s *shipmentSvcImpl) reverseInventoryOnVoid(txCtx context.Context, scope *ledgerlock.Scope, shipment *domain.Shipment) *apierror.APIError {
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

		if apiErr := mutationRepo.ReverseInventoryForOrderItem(txCtx, scope, shipment.AccountID, shipment.SalesOrderID, *line.OrderLineItemID, measure); apiErr != nil {
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
	itemIDs = ledgerlock.SortedUnique(itemIDs)

	if apiErr := mediator.EnqueueAllocateOpenIssues(txCtx, s.repos, shipment.AccountID, itemIDs...); apiErr != nil {
		return apiErr
	}

	for _, itemID := range itemIDs {
		if apiErr := mediator.RecordInventoryAuditTrail(
			txCtx,
			s.repos,
			shipment.AccountID,
			itemID,
			reversedMeasures[itemID],
			reversedUnits[itemID],
			string(constants.InventoryActionTypeUserCorrection),
			nil,
			nil,
		); apiErr != nil {
			return apiErr
		}
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
	doc, attachment, apiErr := s.buildInvoiceDocument(txCtx, shipment.AccountID, invoiceID, logo)
	if apiErr != nil {
		// The lines could not be priced, so any email would state the wrong amount. The invoice itself
		// is sound and can be emailed once the units are fixed; the ship should not fail over it.
		slog.WarnContext(txCtx, "invoice lines could not be priced; skipping the ship's invoice emails",
			"account_id", shipment.AccountID, "invoice_id", invoiceID, "error", apiErr.Error())
		return nil
	}

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
//
// An error means the lines could not be priced, and the document must not be sent.
func (s *shipmentSvcImpl) buildInvoiceDocument(txCtx context.Context, accountID, invoiceID string, logo ackLogo) (invoiceDoc, *string, *apierror.APIError) {
	invoice, apiErr := s.repos.NewInvoiceRepo().Get(txCtx, domain.GetInvoiceParams{AccountID: accountID, InvoiceID: invoiceID})
	if apiErr != nil {
		return invoiceDoc{}, nil, nil
	}
	lines, apiErr := s.repos.NewInvoiceRepo().GetLines(txCtx, invoiceID)
	if apiErr != nil {
		return invoiceDoc{}, nil, nil
	}

	doc, apiErr := gatherInvoiceDoc(txCtx, s.repos, accountID, invoice, lines)
	if apiErr != nil {
		return invoiceDoc{}, nil, apiErr
	}
	doc.Header.OrderOnlineLink = portalRegisterLink(txCtx, s.repos, s.portalURL, accountID)
	// Fetched before the transaction opened, because embedding needs the bytes and a stalled logo
	// host must not hold the ship's row locks.
	doc.Header.LogoImageType, doc.Header.LogoImage, doc.Header.LogoURL = logo.ImageType, logo.Image, logo.URL

	pdfBytes, err := buildInvoicePDF(doc)
	if err != nil {
		return doc, nil, nil
	}
	encoded := base64.StdEncoding.EncodeToString(pdfBytes)
	return doc, &encoded, nil
}

// Emails the customer the invoice and flags it sent. Gated on email_customer by the caller.
func (s *shipmentSvcImpl) emailCustomerInvoiceOnShip(txCtx context.Context, shipment *domain.Shipment, invoiceID string, doc invoiceDoc, attachment *string) *apierror.APIError {
	accountID := shipment.AccountID
	recipients, apiErr := s.repos.NewInvoiceRepo().GetEmailRecipients(txCtx, accountID, invoiceID)
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

// What the ship's label step leaves for the atomic phase to stamp.
type shipLabels struct {
	// MasterTracking, when set, replaces the shipment's master tracking number.
	MasterTracking *string
	// CostRecorded means the carrier's charge is already on the freight line and must not be zeroed.
	CostRecorded bool
}

// A label purchase resolved and validated in full before any money is spent.
type labelPurchase struct {
	client domain.ShippoClient
	// Holds the shipment's cases in parcel order; the purchased packages come back in the same order.
	cases  []*domain.ShippingCase
	params domain.CreateLabelParams
}

// labelPhonePlaceholder stands in for a phone nobody recorded; carriers refuse a label without one.
const labelPhonePlaceholder = "555-555-5555"

// Decides what the ship does about carrier labels without buying any: a placeholder for a sandbox,
// nothing for a carrier or account that cannot buy, the stored labels when they were already bought,
// otherwise a purchase that has passed every check legacy ran before calling the carrier.
func (s *shipmentSvcImpl) planShipmentLabels(ctx context.Context, shipment *domain.Shipment) (*labelPurchase, shipLabels, *apierror.APIError) {
	accountCtx, apiErr := s.repos.NewAccountRepo().GetAccountContext(ctx, shipment.AccountID)
	if apiErr != nil {
		return nil, shipLabels{}, apiErr
	}
	if accountCtx.IsSandbox {
		tracking := sandboxTrackingNumber(shipment.ID)
		return nil, shipLabels{MasterTracking: &tracking}, nil
	}

	// A label is bought against a Shippo carrier account at a specific service level; without either
	// there is nothing to buy, matching legacy's "non-Shippo carriers don't generate labels".
	carrier, apiErr := s.repos.NewCarrierRepo().Get(ctx, domain.GetCarrierParams{AccountID: shipment.AccountID, CarrierID: shipment.CarrierID})
	if apiErr != nil {
		return nil, shipLabels{}, apiErr
	}
	if carrier.ShippoCarrierAccountID == nil || *carrier.ShippoCarrierAccountID == "" {
		return nil, shipLabels{}, nil
	}
	if shipment.ServiceLevelToken == nil || *shipment.ServiceLevelToken == "" {
		return nil, shipLabels{}, nil
	}

	cases, apiErr := s.repos.NewShippingCaseRepo().ListByShipment(ctx, shipment.ID)
	if apiErr != nil {
		return nil, shipLabels{}, apiErr
	}
	if len(cases) == 0 {
		return nil, shipLabels{}, nil
	}

	if stored, bought, apiErr := boughtShipmentLabels(shipment, cases); apiErr != nil || bought {
		return nil, stored, apiErr
	}

	shippoClient, apiErr := s.accountShippoClient(ctx, shipment.AccountID)
	if apiErr != nil {
		return nil, shipLabels{}, apiErr
	}
	if shippoClient == nil {
		return nil, shipLabels{}, nil
	}

	for _, c := range cases {
		if !parseDecimalOrZero(c.FreightWeightValue).IsPositive() {
			return nil, shipLabels{}, apierror.NewValidationError(fmt.Sprintf("Shipping case %s has no freight weight; weigh every case before buying labels.", c.Number))
		}
	}

	// Ship-from is the account's configured origin (its default billing address). Refuse to buy a
	// label from an empty origin rather than printing one the carrier will reject.
	var from domain.ShippingAddress
	if origin, apiErr := s.repos.NewSalesOrderRepo().GetAccountOriginAddress(ctx, shipment.AccountID); apiErr != nil {
		return nil, shipLabels{}, apiErr
	} else if origin != nil {
		from = *origin
	}
	if from.Zip == "" || from.Country == "" {
		return nil, shipLabels{}, apierror.NewValidationError("Cannot buy a shipping label: the account has no default billing (ship-from) address.")
	}

	to := shipmentToAddress(shipment)
	toPhone, fromPhone, apiErr := s.labelPhones(ctx, shipment, from)
	if apiErr != nil {
		return nil, shipLabels{}, apiErr
	}
	to.Phone, from.Phone = &toPhone, &fromPhone
	to.Company, from.Company = labelCompany(to.Name), labelCompany(from.Name)

	return &labelPurchase{
		client: shippoClient,
		cases:  cases,
		params: domain.CreateLabelParams{
			CarrierAccountObjectID: *carrier.ShippoCarrierAccountID,
			ServiceLevelToken:      *shipment.ServiceLevelToken,
			FromAddress:            from,
			ToAddress:              to,
			Parcels:                labelParcels(shipment, cases),
			Billing:                shipmentThirdPartyBilling(shipment),
			Metadata:               shipment.ID,
		},
	}, shipLabels{}, nil
}

// Reads labels an earlier attempt bought for these cases: buying again would charge twice and orphan
// the first purchase, whose transactions void refunds. Labels on only some of the cases are refused.
func boughtShipmentLabels(shipment *domain.Shipment, cases []*domain.ShippingCase) (shipLabels, bool, *apierror.APIError) {
	bought := 0
	for _, c := range cases {
		if c.ShippoTransactionID != nil && *c.ShippoTransactionID != "" {
			bought++
		}
	}
	switch bought {
	case 0:
		return shipLabels{}, false, nil
	case len(cases):
		tracking := shipment.MasterTrackingNumber
		if tracking == nil || *tracking == "" {
			tracking = cases[0].TrackingNumber
		}
		return shipLabels{MasterTracking: tracking, CostRecorded: true}, true, nil
	default:
		return shipLabels{}, false, apierror.NewConflictErrorWithParam("Labels were already bought for some of this shipment's cases but not the rest; the rest cannot be bought on their own.", "id")
	}
}

// Resolves label phones as legacy did: the recipient at the buyer's own number, then the ship-to's,
// then the seller's; the sender at its origin's number, then the seller's.
func (s *shipmentSvcImpl) labelPhones(ctx context.Context, shipment *domain.Shipment, from domain.ShippingAddress) (toPhone, fromPhone string, apiErr *apierror.APIError) {
	accounts, apiErr := s.repos.NewAccountRepo().GetByIDs(ctx, []string{shipment.AccountID, shipment.CustomerID})
	if apiErr != nil {
		return "", "", apiErr
	}
	brandingPhone := make(map[string]string, len(accounts))
	for _, a := range accounts {
		if a != nil && a.Branding != nil {
			brandingPhone[a.ID] = ptrutil.Deref(a.Branding.PhoneNumber)
		}
	}
	seller := brandingPhone[shipment.AccountID]

	toPhone = firstNonBlank(brandingPhone[shipment.CustomerID], ptrutil.Deref(shipment.ShippingAddressPhone), seller, labelPhonePlaceholder)
	fromPhone = firstNonBlank(ptrutil.Deref(from.Phone), seller, labelPhonePlaceholder)
	return toPhone, fromPhone, nil
}

func firstNonBlank(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

// Prints the address name as the company too, as legacy's label addresses did.
func labelCompany(name string) *string {
	if name == "" {
		return nil
	}
	return &name
}

// Turns the cases into label parcels referenced the way legacy printed them: the customer's PO and
// the order number when there is a PO, else the order and case numbers. Each parcel carries its case id.
func labelParcels(shipment *domain.Shipment, cases []*domain.ShippingCase) []domain.Parcel {
	parcels := shippingCaseParcels(cases)
	po := strings.TrimSpace(ptrutil.Deref(shipment.CustomerPONumber))
	for i, c := range cases {
		if po != "" {
			parcels[i].Reference1 = "PO#" + po
			parcels[i].Reference2 = "SO#" + shipment.SalesOrderNumber
		} else {
			parcels[i].Reference1 = "SO#" + shipment.SalesOrderNumber
			parcels[i].Reference2 = "C#" + c.Number
		}
		parcels[i].Metadata = c.ID
	}
	return parcels
}

// Bound the label purchase, detached from the request so a caller that gives up cannot cut it off
// between paying and recording; copying the labels to the bucket is best-effort and budgeted apart.
const (
	labelPurchaseBudget = 60 * time.Second
	labelStoreBudget    = 30 * time.Second
)

// Buys the planned labels and records them before anything else can fail. The recorded transactions
// are what stop a retry from buying again and what void refunds.
func (s *shipmentSvcImpl) buyShipmentLabels(ctx context.Context, shipment *domain.Shipment, purchase *labelPurchase, idempotencyTypeID string) (shipLabels, *apierror.APIError) {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), labelPurchaseBudget)
	defer cancel()

	result, apiErr := purchase.client.CreateTransactionInstantLabel(ctx, purchase.params)
	if apiErr != nil {
		return shipLabels{}, apiErr
	}
	// No labels means no purchase happened (the stub client in test mode): nothing to record.
	if len(result.Packages) == 0 {
		return shipLabels{}, nil
	}

	var recordErr *apierror.APIError
	if len(result.Packages) != len(purchase.cases) {
		recordErr = apierror.NewInternalError(nil, fmt.Sprintf("bought %d labels for %d cases", len(result.Packages), len(purchase.cases)))
	} else {
		recordErr = s.recordLabelPurchase(ctx, shipment, purchase.cases, result, idempotencyTypeID)
	}
	if recordErr != nil {
		return shipLabels{}, boughtButUnrecorded(ctx, shipment, result, recordErr)
	}

	storeCtx, cancelStore := context.WithTimeout(context.WithoutCancel(ctx), labelStoreBudget)
	defer cancelStore()
	for i, c := range purchase.cases {
		s.storeShippingLabel(storeCtx, shipment.AccountID, c.Number, result.Packages[i].LabelURL)
	}

	return shipLabels{CostRecorded: true}, nil
}

// Reports labels that were paid for but could not be recorded. Not transient: a retry would find no
// recorded transaction and buy again, so the purchase is logged for reconciliation instead.
func boughtButUnrecorded(ctx context.Context, shipment *domain.Shipment, result *domain.LabelResult, cause *apierror.APIError) *apierror.APIError {
	transactionIDs := make([]string, len(result.Packages))
	for i, pkg := range result.Packages {
		transactionIDs[i] = pkg.ShippoTransactionID
	}
	slog.ErrorContext(ctx, "shipping labels were bought but could not be recorded",
		"account_id", shipment.AccountID, "shipment_id", shipment.ID,
		"shippo_transaction_ids", strings.Join(transactionIDs, ","), "error", cause.Error())
	return apierror.NewResourceConflictError("The shipping labels were purchased but could not be recorded. Contact support before shipping this shipment again.")
}

// Bound the attempts at recording a purchase: the labels are paid for, so a database blip is retried
// rather than leaving them unrecorded.
const (
	recordLabelAttempts = 3
	recordLabelBackoff  = 250 * time.Millisecond
)

// Records bought labels in one transaction — each case's tracking and refundable transaction, the
// master tracking, the freight cost and the recovery point — so a retry finds all of it or none.
func (s *shipmentSvcImpl) recordLabelPurchase(ctx context.Context, shipment *domain.Shipment, cases []*domain.ShippingCase, result *domain.LabelResult, idempotencyTypeID string) *apierror.APIError {
	record := func(txCtx context.Context, txSvc *shipmentSvcImpl) *apierror.APIError {
		caseRepo := txSvc.repos.NewShippingCaseRepo()
		for i, c := range cases {
			pkg := result.Packages[i]
			if apiErr := caseRepo.UpdateWithShipmentInfo(txCtx, c.ID, pkg.TrackingNumber, pkg.ShippoTransactionID, pkg.LabelURL); apiErr != nil {
				return apiErr
			}
		}
		if result.MasterTrackingNumber != "" {
			if apiErr := txSvc.repos.NewShipmentRepo().SetMasterTracking(txCtx, shipment.AccountID, shipment.ID, result.MasterTrackingNumber); apiErr != nil {
				return apiErr
			}
		}
		if apiErr := txSvc.writeBackNegotiatedRate(txCtx, shipment, result.NegotiatedRate); apiErr != nil {
			return apiErr
		}
		return txSvc.repos.NewIdempotencyKeyRepo().AdvanceRecoveryPoint(txCtx, idempotencyTypeID, domain.RecoveryPointShipLabelsCreated)
	}

	var apiErr *apierror.APIError
	for attempt := 1; attempt <= recordLabelAttempts; attempt++ {
		apiErr = s.withTx(ctx, record)
		if apiErr == nil || !apiErr.IsTransient || attempt == recordLabelAttempts {
			return apiErr
		}
		select {
		case <-ctx.Done():
			return apiErr
		case <-time.After(time.Duration(attempt) * recordLabelBackoff):
		}
	}
	return apiErr
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

// Builds the account's Shippo client for the ship and void paths.
func (s *shipmentSvcImpl) accountShippoClient(ctx context.Context, accountID string) (domain.ShippoClient, *apierror.APIError) {
	return accountShippoClient(ctx, s.repos, s.shippoFactory, s.encryptionKey, accountID)
}

// Builds the account's Shippo client from its stored credentials. No integration (or no factory) is
// (nil, nil), so a caller can skip the carrier; one switched off is refused, as legacy's
// "Shippo integration is inactive." was.
func accountShippoClient(ctx context.Context, repos domain.RepoFactory, factory domain.ShippoClientFactory, encryptionKey []byte, accountID string) (domain.ShippoClient, *apierror.APIError) {
	if factory == nil {
		return nil, nil
	}

	integrationRepo := repos.NewAccountIntegrationRepo()
	hasIntegration, apiErr := integrationRepo.HasIntegration(ctx, accountID, constants.IntegrationCodeShippo)
	if apiErr != nil {
		return nil, apiErr
	}
	if !hasIntegration {
		return nil, nil
	}

	encryptedCreds, isActive, apiErr := integrationRepo.GetEncryptedCredentials(ctx, accountID, constants.IntegrationCodeShippo)
	if apiErr != nil {
		return nil, apiErr
	}
	if !isActive {
		return nil, apierror.NewValidationError("Shippo integration is inactive.")
	}
	apiKey, apiErr := decryptShippoAPIKey(encryptedCreds, encryptionKey, accountID)
	if apiErr != nil {
		return nil, apiErr
	}

	return factory.Build(apiKey)
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

// Bills freight to the third party the order names, with the order's billing-address country and zip
// as the billing address — legacy's shipment adapter, which read the order alone. Nil when not third-party billed.
func shipmentThirdPartyBilling(shipment *domain.Shipment) *domain.ShippingBilling {
	if shipment.OrderCarrierBillingType == nil || *shipment.OrderCarrierBillingType != string(constants.CarrierBillingTypeThirdParty) {
		return nil
	}
	return &domain.ShippingBilling{
		Type:    "THIRD_PARTY",
		Account: ptrutil.Deref(shipment.OrderCarrierBillingAccount),
		Country: ptrutil.Deref(shipment.BillingAddressCountry),
		Zip:     ptrutil.Deref(shipment.BillingAddressZip),
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

// labelRefundBudget bounds refunding a void's labels, detached from the request so a caller that gives
// up cannot leave the refunds half-recorded.
const labelRefundBudget = 60 * time.Second

// Refunds the cases' bought labels and drops their stored files before void clears them; sandbox no-ops.
// A refused refund aborts the void, as in legacy: clearing the case would erase a charged label's record.
func (s *shipmentSvcImpl) refundShippingLabels(ctx context.Context, shipment *domain.Shipment, idempotencyTypeID string) *apierror.APIError {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), labelRefundBudget)
	defer cancel()

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

	if apiErr := s.refundShippoTransactions(ctx, shipment.AccountID, cases); apiErr != nil {
		return apiErr
	}
	s.deleteStoredShippingLabels(ctx, shipment.AccountID, cases)

	return s.repos.NewIdempotencyKeyRepo().AdvanceRecoveryPoint(ctx, idempotencyTypeID, domain.RecoveryPointVoidLabelsRefunded)
}

// Refunds each case's bought Shippo transaction, stopping at the first the carrier refuses. A label
// already refunded counts as refunded, so a retried void passes the ones an earlier attempt got through.
func (s *shipmentSvcImpl) refundShippoTransactions(ctx context.Context, accountID string, cases []*domain.ShippingCase) *apierror.APIError {
	var transactionIDs []string
	for _, c := range cases {
		if c.ShippoTransactionID != nil && *c.ShippoTransactionID != "" {
			transactionIDs = append(transactionIDs, *c.ShippoTransactionID)
		}
	}
	if len(transactionIDs) == 0 {
		return nil
	}

	shippoClient, apiErr := s.accountShippoClient(ctx, accountID)
	if apiErr != nil {
		return apiErr
	}
	if shippoClient == nil {
		return apierror.NewValidationError("The shipment's shipping labels cannot be refunded: the account has no Shippo integration.")
	}

	for _, transactionID := range transactionIDs {
		if apiErr := shippoClient.RefundTransaction(ctx, transactionID); apiErr != nil {
			return apiErr
		}
	}
	return nil
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

// dispatchClaimTTL outlives the longest ship or void attempt — a detached label purchase plus the
// atomic phase — and frees the shipment by itself if an attempt dies holding it.
const dispatchClaimTTL = 3 * time.Minute

// Claims the shipment for one ship, void or delete attempt, so a second attempt — even this request
// replayed while the first still runs — conflicts instead of buying or invoicing again.
func (s *shipmentSvcImpl) claimDispatch(ctx context.Context, shipmentID string) (func(), *apierror.APIError) {
	name := "shipment-dispatch:" + shipmentID
	holder := rand.Text()

	acquired, err := s.dispatchLeases.Acquire(ctx, name, holder, dispatchClaimTTL)
	if err != nil {
		return nil, apierror.NewInternalError(err, "Failed to claim the shipment.")
	}
	if !acquired {
		return nil, apierror.NewConflictErrorWithParam("This shipment is already being shipped, voided or deleted by another request; retry once it finishes.", "id")
	}

	return func() {
		releaseCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		if err := s.dispatchLeases.Release(releaseCtx, name, holder); err != nil {
			slog.WarnContext(ctx, "shipment dispatch claim release failed; it lapses on its own",
				"shipment_id", shipmentID, "error", err.Error())
		}
	}, nil
}
