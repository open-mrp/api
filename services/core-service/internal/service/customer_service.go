package service

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/audit"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/contracts"
	"github.com/open-mrp/api/shared/id"
	"github.com/open-mrp/api/shared/idemresponse"
	"github.com/open-mrp/api/shared/messaging"
	apierror "github.com/open-mrp/apikit/apierror"
	"github.com/open-mrp/apikit/appctx"
	"github.com/open-mrp/apikit/field"
	"github.com/open-mrp/apikit/tracing"
)

var customerSvcTracer = tracing.GetTracer("core-service.service.customer")

type customerSvcImpl struct {
	repos           domain.RepoFactory
	mediatorFactory domain.MediatorFactory
	jobSvcFactory   domain.JobSvcFactory
	txManager       TransactionManager
}

type CustomerSvcConfig struct {
	// Repos (required) is the repository factory.
	Repos domain.RepoFactory

	// MediatorFactory (required) builds the mediators used by this service.
	MediatorFactory domain.MediatorFactory

	// JobSvcFactory (required) builds the job service an export records on.
	JobSvcFactory domain.JobSvcFactory

	// TxManager (required) wraps multi-step operations in database transactions.
	TxManager TransactionManager
}

func (c *CustomerSvcConfig) validate() error {
	if c.Repos == nil {
		return fmt.Errorf("customer service: Repos is required")
	}
	if c.MediatorFactory == nil {
		return fmt.Errorf("customer service: MediatorFactory is required")
	}
	if c.JobSvcFactory == nil {
		return fmt.Errorf("customer service: JobSvcFactory is required")
	}
	if c.TxManager == nil {
		return fmt.Errorf("customer service: TxManager is required")
	}
	return nil
}

func NewCustomerSvc(config *CustomerSvcConfig) domain.CustomerSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &customerSvcImpl{
		repos:           config.Repos,
		mediatorFactory: config.MediatorFactory,
		jobSvcFactory:   config.JobSvcFactory,
		txManager:       config.TxManager,
	}
}

func (s *customerSvcImpl) mediators() domain.Mediators {
	return s.mediatorFactory.Build(s.repos)
}

func (s *customerSvcImpl) withTx(ctx context.Context, fn func(context.Context, *customerSvcImpl) *apierror.APIError) *apierror.APIError {
	if s.txManager == nil {
		panic("txManager is nil")
	}

	return s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		txSvc := &customerSvcImpl{
			repos:           f,
			mediatorFactory: s.mediatorFactory,
			jobSvcFactory:   s.jobSvcFactory,
			txManager:       s.txManager,
		}
		return fn(txCtx, txSvc)
	})
}

// ListCustomers returns a paginated list of customers for the caller's account.
// Internal actors only.
func (s *customerSvcImpl) ListCustomers(ctx context.Context, params domain.ListCustomersParams) (*domain.ListCustomersResult, *apierror.APIError) {
	ctx, span := customerSvcTracer.Start(ctx, "service.customer.list")
	defer span.End()

	identity, ok := appctx.Identity[*types.Identity](ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainCustomers, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	params.AccountID = identity.Target.AccountID

	result, apiErr := s.repos.NewCustomerRepo().List(ctx, params)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if slices.Contains(params.Includes, customerIncludeNotificationRecipients) {
		for _, c := range result.Items {
			if apiErr := s.loadNotificationRecipients(ctx, params.AccountID, c); apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
		}
	}
	return result, nil
}

// customerIncludeNotificationRecipients is the include that loads a customer's notification recipients.
const customerIncludeNotificationRecipients = "notification_recipients"

// loadNotificationRecipients fills in c's notification recipients, as an empty list when it has none.
func (s *customerSvcImpl) loadNotificationRecipients(ctx context.Context, ownerAccountID string, c *domain.Customer) *apierror.APIError {
	relationID, apiErr := s.repos.NewAccountRelationRepo().FindRelationByOwnerAndCounterparty(ctx, ownerAccountID, c.ID)
	if apiErr != nil {
		return apiErr
	}
	recipients, apiErr := s.hydrateNotificationRecipients(ctx, c.ID, relationID)
	if apiErr != nil {
		return apiErr
	}
	if recipients == nil {
		recipients = []domain.NotificationRecipient{}
	}
	c.NotificationRecipients = recipients
	return nil
}

// GetCustomer retrieves a single customer by account ID. A customer or supplier portal actor reads only its own account's record.
func (s *customerSvcImpl) GetCustomer(ctx context.Context, customerAccountID string, includes []string) (*domain.Customer, *apierror.APIError) {
	ctx, span := customerSvcTracer.Start(ctx, "service.customer.get")
	defer span.End()

	identity, ok := appctx.Identity[*types.Identity](ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsAssignedActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := checkCustomerReadPermission(identity); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if !identity.IsTargetAccountSet() {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("The OpenMRP-Account-ID header is required."))
	}

	if own := identity.PortalAccountID(); own != nil && customerAccountID != *own {
		return nil, tracing.Trace(span, apierror.NewNotFoundError("Customer not found."))
	}

	if identity.IsExternalTarget() {
		meds := s.mediators()
		if apiErr := meds.ReadAccess.CheckCounterpartyReadAccess(ctx, *identity.ActorAccountID(), identity.Target.AccountID); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}

	ownerAccountID := identity.Target.AccountID

	customer, apiErr := s.repos.NewCustomerRepo().Get(ctx, ownerAccountID, customerAccountID, includes)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if slices.Contains(includes, customerIncludeNotificationRecipients) {
		if apiErr := s.loadNotificationRecipients(ctx, ownerAccountID, customer); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}

	return customer, nil
}

func (s *customerSvcImpl) BatchGetCustomers(ctx context.Context, customerAccountIDs []string) ([]*domain.Customer, *apierror.APIError) {
	ctx, span := customerSvcTracer.Start(ctx, "service.customer.batch_get")
	defer span.End()

	identity, ok := appctx.Identity[*types.Identity](ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsAssignedActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := checkCustomerReadPermission(identity); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if !identity.IsTargetAccountSet() {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("The OpenMRP-Account-ID header is required."))
	}

	ids := customerAccountIDs
	if own := identity.PortalAccountID(); own != nil && !identity.IsIncludeRead() {
		ids = nil
		if slices.Contains(customerAccountIDs, *own) {
			ids = []string{*own}
		}
	}

	if identity.IsExternalTarget() {
		meds := s.mediators()
		if apiErr := meds.ReadAccess.CheckCounterpartyReadAccess(ctx, *identity.ActorAccountID(), identity.Target.AccountID); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}

	customers, apiErr := s.repos.NewCustomerRepo().GetByIDs(ctx, identity.Target.AccountID, ids)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return customers, nil
}

// CreateCustomer creates a new customer account with idempotency.
func (s *customerSvcImpl) CreateCustomer(ctx context.Context, params domain.CreateCustomerParams) (*domain.Customer, *apierror.APIError) {
	ctx, span := customerSvcTracer.Start(ctx, "service.customer.create")
	defer span.End()

	identity, ok := appctx.Identity[*types.Identity](ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainCustomers, types.ActionCreate); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if params.BillToAddress == nil {
		return nil, tracing.Trace(span, apierror.NewValidationError("A bill-to address is required when creating a customer."))
	}
	if params.ShipToAddress == nil {
		return nil, tracing.Trace(span, apierror.NewValidationError("A ship-to address is required when creating a customer."))
	}

	accountID, apiErr := id.GenID(id.AccountIDPrefix, nil)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	relationID, apiErr := id.GenID(id.AccountRelationIDPrefix, nil)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	brandingID, apiErr := id.GenID(id.AccountBrandingIDPrefix, nil)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	params.OwnerAccountID = identity.Target.AccountID
	if params.ArchiveStatus == "" {
		params.ArchiveStatus = constants.ArchiveStatusActive
	}

	meds := s.mediators()

	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, identity)
	if apiErr != nil {
		return nil, apiErr
	}

	switch domain.RecoveryPoint(idempotencyKey.RecoveryPoint) {
	case domain.RecoveryPointFinished:
		cached, err := idemresponse.UnmarshalCachedResponse[domain.Customer](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.Customer
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *customerSvcImpl) *apierror.APIError {
			txCustomerRepo := txSvc.repos.NewCustomerRepo()

			if params.ParentCustomerID != nil {
				parentRelationID, apiErr := resolveParentCustomer(txCtx, txSvc.repos, params.OwnerAccountID, *params.ParentCustomerID, nil)
				if apiErr != nil {
					return apiErr
				}
				params.ParentRelationID = &parentRelationID
			}

			// Every create takes a number, typed in or allocated, so it queues behind the owner's other
			// number writers before anything else is read.
			if apiErr := txCustomerRepo.LockNumbers(txCtx, params.OwnerAccountID); apiErr != nil {
				return apiErr
			}

			if apiErr := checkCustomerRefs(txCtx, txSvc.repos, params.OwnerAccountID, newCustomerRefs(params)); apiErr != nil {
				return apiErr
			}
			if apiErr := checkServiceLevelOnCarrier(txCtx, txSvc.repos, params.DefaultCarrierID, params.DefaultServiceLevelID, "defaults.service_level_id"); apiErr != nil {
				return apiErr
			}

			// Generate or auto-assign customer number.
			var customerNumber string
			if params.Number != nil && *params.Number != "" {
				customerNumber = *params.Number

				// Check for duplicate customer number.
				exists, apiErr := txCustomerRepo.ExistsByNumber(txCtx, params.OwnerAccountID, customerNumber, nil)
				if apiErr != nil {
					return apiErr
				}
				if exists {
					return apierror.NewConflictErrorWithParam("number", "A customer with this number already exists.")
				}
			} else {
				// Auto-generate the next customer number, reserved in one locked statement so
				// two customers created at the same moment cannot be given the same one, and past
				// any number someone typed in.
				sysPropertyID, apiErr := id.GenID(id.SysPropertyIDPrefix, nil)
				if apiErr != nil {
					return apiErr
				}

				nextNum, apiErr := txCustomerRepo.AllocateNextCustomerNumber(txCtx, sysPropertyID, params.OwnerAccountID)
				if apiErr != nil {
					return apiErr
				}
				customerNumber = strconv.FormatInt(nextNum, 10)
			}

			// Create inline bill-to address.
			if params.BillToAddress != nil {
				addrID, geoID, acctAddrID, apiErr := generateAddressIDs()
				if apiErr != nil {
					return apiErr
				}
				params.BillToAddress.AccountID = accountID
				if _, apiErr := txSvc.repos.NewAddressRepo().Create(txCtx, addrID, geoID, acctAddrID, *params.BillToAddress); apiErr != nil {
					return apiErr
				}
				params.BillToAddressID = &addrID
			}

			// Create inline ship-to address, reusing the bill-to if identical.
			if params.ShipToAddress != nil {
				if params.BillToAddress != nil && addressParamsEqual(*params.BillToAddress, *params.ShipToAddress) {
					params.ShipToAddressID = params.BillToAddressID
				} else {
					addrID, geoID, acctAddrID, apiErr := generateAddressIDs()
					if apiErr != nil {
						return apiErr
					}
					params.ShipToAddress.AccountID = accountID
					if _, apiErr := txSvc.repos.NewAddressRepo().Create(txCtx, addrID, geoID, acctAddrID, *params.ShipToAddress); apiErr != nil {
						return apiErr
					}
					params.ShipToAddressID = &addrID
				}
			}

			// Create credit limit quantity if provided.
			if params.CreditLimitValue != nil && params.CreditLimitUnitID != nil {
				creditLimitQtyID, apiErr := id.GenID(id.QuantityIDPrefix, nil)
				if apiErr != nil {
					return apiErr
				}
				if apiErr := txCustomerRepo.InsertCreditLimitQuantity(txCtx, creditLimitQtyID, *params.CreditLimitValue, *params.CreditLimitUnitID); apiErr != nil {
					return apiErr
				}
				params.CreditLimitID = &creditLimitQtyID
			}

			if _, apiErr := txCustomerRepo.Create(txCtx, accountID, relationID, brandingID, params, customerNumber); apiErr != nil {
				return apiErr
			}

			// Ensure billing/shipping addresses are linked to the customer account.
			if params.BillToAddressID != nil {
				if apiErr := ensureAccountAddressLink(txCtx, txCustomerRepo, accountID, *params.BillToAddressID); apiErr != nil {
					return apiErr
				}
			}
			if params.ShipToAddressID != nil {
				if apiErr := ensureAccountAddressLink(txCtx, txCustomerRepo, accountID, *params.ShipToAddressID); apiErr != nil {
					return apiErr
				}
			}

			// Create price groups if provided.
			for _, pgID := range params.CustomerPriceGroupIDs {
				priceGroupRelID, apiErr := id.GenID(id.AccountRelationPriceGroupIDPrefix, nil)
				if apiErr != nil {
					return apiErr
				}
				if apiErr := txCustomerRepo.InsertPriceGroup(txCtx, priceGroupRelID, relationID, pgID); apiErr != nil {
					return apiErr
				}
			}

			// Re-fetch customer to include price groups and full data for audit and response.
			result, apiErr = txCustomerRepo.Get(txCtx, params.OwnerAccountID, accountID, customerAuditIncludes(params.Includes))
			if apiErr != nil {
				return apiErr
			}
			if slices.Contains(params.Includes, customerIncludeNotificationRecipients) {
				if apiErr := txSvc.loadNotificationRecipients(txCtx, params.OwnerAccountID, result); apiErr != nil {
					return apiErr
				}
			}

			changes := audit.ComputeChanges(nil, result)

			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionCreate,
				ResourceType: constants.ObjectTypeCustomer,
				ResourceID:   result.ID,
				Changes:      changes,
			}); apiErr != nil {
				return apiErr
			}

			// Only worth a message once there is an email to key the Stripe customer on;
			// the first update that adds one enqueues the sync instead.
			if result.Email != nil && *result.Email != "" {
				if apiErr := enqueueStripeCustomerSync(txCtx, txSvc.repos, params.OwnerAccountID, result.ID); apiErr != nil {
					return apiErr
				}
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

// UpdateCustomer partially updates an existing customer account with idempotency.
func (s *customerSvcImpl) UpdateCustomer(ctx context.Context, params domain.UpdateCustomerParams) (*domain.Customer, *apierror.APIError) {
	ctx, span := customerSvcTracer.Start(ctx, "service.customer.update")
	defer span.End()

	identity, ok := appctx.Identity[*types.Identity](ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainCustomers, types.ActionUpdate); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	params.OwnerAccountID = identity.Target.AccountID

	meds := s.mediators()

	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, identity)
	if apiErr != nil {
		return nil, apiErr
	}

	switch domain.RecoveryPoint(idempotencyKey.RecoveryPoint) {
	case domain.RecoveryPointFinished:
		cached, err := idemresponse.UnmarshalCachedResponse[domain.Customer](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.Customer
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *customerSvcImpl) *apierror.APIError {
			txCustomerRepo := txSvc.repos.NewCustomerRepo()

			// Before the first read, or the number check below reads a snapshot older than the lock.
			if params.Number != nil && *params.Number != "" {
				if apiErr := txCustomerRepo.LockNumbers(txCtx, params.OwnerAccountID); apiErr != nil {
					return apiErr
				}
			}

			old, apiErr := txCustomerRepo.Get(txCtx, params.OwnerAccountID, params.CustomerAccountID, []string{"price_groups", "bill_to_address", "ship_to_address"})
			if apiErr != nil {
				return apiErr
			}

			if apiErr := checkCustomerRefs(txCtx, txSvc.repos, params.OwnerAccountID, changedCustomerRefs(params, old)); apiErr != nil {
				return apiErr
			}
			if apiErr := checkUpdatedServiceLevelOnCarrier(txCtx, txSvc.repos, old.DefaultCarrierID, old.DefaultServiceLevelID, params.DefaultCarrierID, params.DefaultServiceLevelID, "defaults.service_level_id"); apiErr != nil {
				return apiErr
			}

			if params.DefaultCarrierID == nil {
				params.DefaultCarrierID = old.DefaultCarrierID
			}
			params.DefaultServiceLevelID = params.DefaultServiceLevelID.BackfillUnsetPtr(old.DefaultServiceLevelID)
			if params.DefaultPaymentTermID == nil {
				params.DefaultPaymentTermID = old.DefaultPaymentTermID
			}
			if params.DefaultShippingTermID == nil {
				params.DefaultShippingTermID = old.DefaultShippingTermID
			}
			params.DefaultSalesRepID = params.DefaultSalesRepID.BackfillUnsetPtr(old.DefaultSalesRepID)
			if params.CustomerTypeGroupID == nil {
				params.CustomerTypeGroupID = old.TypeGroupID
			}
			params.Instructions = params.Instructions.BackfillUnsetPtr(old.Instructions)
			if params.ArchiveStatus == nil {
				params.ArchiveStatus = &old.ArchiveStatus
			}
			params.CarrierBillingAccount = params.CarrierBillingAccount.BackfillUnsetPtr(old.CarrierBillingAccount)
			params.BillToAddressID = params.BillToAddressID.BackfillUnsetPtr(old.BillToAddressID)
			params.ShipToAddressID = params.ShipToAddressID.BackfillUnsetPtr(old.ShipToAddressID)
			params.DefaultLeadTimeDays = params.DefaultLeadTimeDays.BackfillUnsetPtr(old.DefaultLeadTimeDays)
			params.ReceiveCalendarID = params.ReceiveCalendarID.BackfillUnsetPtr(old.ReceiveCalendarID)
			params.Supply = params.Supply.BackfillUnsetPtr(old.Supply)

			switch {
			case params.CreditLimit.IsUnset():
				params.CreditLimitID = old.CreditLimitID
			case params.CreditLimit.IsClear():
				if old.CreditLimitID != nil {
					if apiErr := txCustomerRepo.DeleteCreditLimitQuantity(txCtx, *old.CreditLimitID); apiErr != nil {
						return apiErr
					}
				}
				params.CreditLimitID = nil
			case params.CreditLimit.IsSet():
				qv, _ := params.CreditLimit.Value()
				if old.CreditLimitID != nil {
					if apiErr := txCustomerRepo.UpdateCreditLimitQuantity(txCtx, *old.CreditLimitID, qv.Value, qv.UnitID); apiErr != nil {
						return apiErr
					}
					params.CreditLimitID = old.CreditLimitID
				} else {
					creditLimitQtyID, apiErr := id.GenID(id.QuantityIDPrefix, nil)
					if apiErr != nil {
						return apiErr
					}
					if apiErr := txCustomerRepo.InsertCreditLimitQuantity(txCtx, creditLimitQtyID, qv.Value, qv.UnitID); apiErr != nil {
						return apiErr
					}
					params.CreditLimitID = &creditLimitQtyID
				}
			}

			// Check for duplicate customer number if being updated.
			if params.Number != nil && *params.Number != "" {
				exists, apiErr := txCustomerRepo.ExistsByNumber(txCtx, params.OwnerAccountID, *params.Number, &params.CustomerAccountID)
				if apiErr != nil {
					return apiErr
				}
				if exists {
					return apierror.NewConflictErrorWithParam("number", "This customer number is already taken.")
				}
			}

			// Get the relation ID for this customer.
			relationID, apiErr := txCustomerRepo.GetRelationID(txCtx, params.OwnerAccountID, params.CustomerAccountID)
			if apiErr != nil {
				return apiErr
			}

			switch {
			case params.ParentCustomerID.IsUnset():
				params.ParentRelationID = old.ParentAccountRelationID
			case params.ParentCustomerID.IsClear():
				params.ParentRelationID = nil
			default:
				parentID, _ := params.ParentCustomerID.Value()
				if old.ParentAccountID != nil && *old.ParentAccountID == parentID {
					params.ParentRelationID = old.ParentAccountRelationID
					break
				}
				parentRelationID, apiErr := resolveParentCustomer(txCtx, txSvc.repos, params.OwnerAccountID, parentID, &relationID)
				if apiErr != nil {
					return apiErr
				}
				params.ParentRelationID = &parentRelationID
			}

			var recipientsBefore []domain.NotificationRecipientRef
			if params.HasNotificationRecipients {
				recipientsBefore, apiErr = txSvc.replaceNotificationRecipients(txCtx, params.CustomerAccountID, relationID, params.NotificationRecipients, "notification_recipients")
				if apiErr != nil {
					return apiErr
				}
			}

			// A new default address must be one of the customer's own: linking any other id would attach
			// another account's address to this customer, readable through its includes.
			if apiErr := checkCustomerDefaultAddresses(txCtx, txCustomerRepo, params, old); apiErr != nil {
				return apiErr
			}

			// Update the account_relation record.
			if apiErr := txCustomerRepo.Update(txCtx, relationID, params); apiErr != nil {
				return apiErr
			}

			if params.BillToAddressID.IsSet() {
				addrID, _ := params.BillToAddressID.Value()
				if apiErr := ensureAccountAddressLink(txCtx, txCustomerRepo, params.CustomerAccountID, addrID); apiErr != nil {
					return apiErr
				}
			}
			if params.ShipToAddressID.IsSet() {
				addrID, _ := params.ShipToAddressID.Value()
				if apiErr := ensureAccountAddressLink(txCtx, txCustomerRepo, params.CustomerAccountID, addrID); apiErr != nil {
					return apiErr
				}
			}

			if params.Email.WasProvided() || params.Phone.WasProvided() || params.URL.WasProvided() {
				email := params.Email.StringPtrAfterBackfill(old.Email)
				phone := params.Phone.StringPtrAfterBackfill(old.Phone)
				url := params.URL.StringPtrAfterBackfill(old.URL)
				if apiErr := txCustomerRepo.UpdateBranding(txCtx, params.CustomerAccountID, email, phone, url); apiErr != nil {
					return apiErr
				}
			}

			// Replace price groups if explicitly provided.
			if params.HasCustomerPriceGroupIDs {
				if apiErr := txCustomerRepo.DeletePriceGroups(txCtx, relationID); apiErr != nil {
					return apiErr
				}
				for _, pgID := range params.CustomerPriceGroupIDs {
					priceGroupRelID, apiErr := id.GenID(id.AccountRelationPriceGroupIDPrefix, nil)
					if apiErr != nil {
						return apiErr
					}
					if apiErr := txCustomerRepo.InsertPriceGroup(txCtx, priceGroupRelID, relationID, pgID); apiErr != nil {
						return apiErr
					}
				}
			}

			// Re-fetch customer to include all updated data for audit and response.
			result, apiErr = txCustomerRepo.Get(txCtx, params.OwnerAccountID, params.CustomerAccountID, customerAuditIncludes(params.Includes))
			if apiErr != nil {
				return apiErr
			}
			if slices.Contains(params.Includes, customerIncludeNotificationRecipients) {
				if apiErr := txSvc.loadNotificationRecipients(txCtx, params.OwnerAccountID, result); apiErr != nil {
					return apiErr
				}
			}

			changes := audit.ComputeChanges(old, result)
			if params.HasNotificationRecipients {
				after, apiErr := txSvc.repos.NewAccountRelationRepo().ListNotificationRecipients(txCtx, relationID)
				if apiErr != nil {
					return apiErr
				}
				if change, changed := recipientsAuditChange(recipientsBefore, after); changed {
					changes = append(changes, change)
				}
			}

			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionUpdate,
				ResourceType: constants.ObjectTypeCustomer,
				ResourceID:   result.ID,
				Changes:      changes,
			}); apiErr != nil {
				return apiErr
			}

			// Only the fields Stripe holds are worth a round trip; the rest of a customer
			// (terms, carriers, price groups) means nothing to it.
			if stripeCustomerFieldsChanged(old, result) {
				if apiErr := enqueueStripeCustomerSync(txCtx, txSvc.repos, params.OwnerAccountID, result.ID); apiErr != nil {
					return apiErr
				}
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

// DeleteCustomer deletes a customer and its associated relations.
func (s *customerSvcImpl) DeleteCustomer(ctx context.Context, params domain.DeleteCustomerParams) *apierror.APIError {
	ctx, span := customerSvcTracer.Start(ctx, "service.customer.delete")
	defer span.End()

	identity, ok := appctx.Identity[*types.Identity](ctx)
	if !ok || identity == nil {
		return tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainCustomers, types.ActionDelete); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	params.OwnerAccountID = identity.Target.AccountID

	repo := s.repos.NewCustomerRepo()

	// Fetch the customer before deleting for audit trail.
	customer, apiErr := repo.Get(ctx, params.OwnerAccountID, params.CustomerAccountID, []string{"price_groups"})
	if apiErr != nil {
		if apierror.HasCode(apiErr, apierror.CodeResourceNotFound) {
			wasDeleted, deletedCheckErr := s.repos.NewDeletedRecordRepo().ExistsInAccount(ctx, constants.DeletedRecordResourceTypeCustomer, params.CustomerAccountID, params.OwnerAccountID)
			if deletedCheckErr != nil {
				return tracing.Trace(span, deletedCheckErr)
			}
			if wasDeleted {
				return tracing.Trace(
					span,
					apierror.NewGoneError("This customer has already been deleted and can no longer be modified."),
				)
			}
		}
		return tracing.Trace(span, apiErr)
	}

	apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *customerSvcImpl) *apierror.APIError {
		relationID, apiErr := txSvc.repos.NewCustomerRepo().GetRelationID(txCtx, params.OwnerAccountID, params.CustomerAccountID)
		if apiErr != nil {
			return apiErr
		}
		labels, apiErr := txSvc.repos.NewCustomerRepo().InUseBy(txCtx, params.OwnerAccountID, params.CustomerAccountID, relationID)
		if apiErr != nil {
			return apiErr
		}
		if len(labels) > 0 {
			return domain.CustomerInUse.InUseError(labels)
		}

		if apiErr := txSvc.repos.NewDeletedRecordRepo().CreateInAccount(txCtx, constants.DeletedRecordResourceTypeCustomer, customer.ID, params.OwnerAccountID, customer); apiErr != nil {
			return apiErr
		}

		if apiErr := txSvc.repos.NewCustomerRepo().Delete(txCtx, params.OwnerAccountID, params.CustomerAccountID); apiErr != nil {
			return apiErr
		}

		changes := audit.ComputeChanges(customer, (*domain.Customer)(nil))

		if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
			ServiceName:  domain.ServiceName,
			Action:       constants.AuditActionDelete,
			ResourceType: constants.ObjectTypeCustomer,
			ResourceID:   customer.ID,
			Changes:      changes,
		}); apiErr != nil {
			return apiErr
		}

		return nil
	})

	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

// BulkDeleteCustomers deletes multiple customers at once.
func (s *customerSvcImpl) BulkDeleteCustomers(ctx context.Context, params domain.BulkDeleteCustomersParams) *apierror.APIError {
	ctx, span := customerSvcTracer.Start(ctx, "service.customer.bulk_delete")
	defer span.End()

	identity, ok := appctx.Identity[*types.Identity](ctx)
	if !ok || identity == nil {
		return tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainCustomers, types.ActionDelete); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	params.OwnerAccountID = identity.Target.AccountID

	// Fetch all customers before deletion for audit trail.
	customerRepo := s.repos.NewCustomerRepo()
	customers := make([]*domain.Customer, 0, len(params.CustomerIDs))
	for _, customerID := range params.CustomerIDs {
		customer, apiErr := customerRepo.Get(ctx, params.OwnerAccountID, customerID, []string{"price_groups"})
		if apiErr != nil {
			return tracing.Trace(span, apiErr)
		}
		customers = append(customers, customer)
	}

	salesOrderRepo := s.repos.NewSalesOrderRepo()
	orderCount, apiErr := salesOrderRepo.CountSalesOrdersForBuyerAccounts(ctx, params.OwnerAccountID, params.CustomerIDs)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if orderCount > 0 {
		return tracing.Trace(span, apierror.NewConflictError(
			"Cannot delete customers while sales orders still reference them. Delete or reassign those orders, or merge customers first.",
		))
	}

	apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *customerSvcImpl) *apierror.APIError {
		for _, customer := range customers {
			if apiErr := txSvc.repos.NewDeletedRecordRepo().CreateInAccount(txCtx, constants.DeletedRecordResourceTypeCustomer, customer.ID, params.OwnerAccountID, customer); apiErr != nil {
				return apiErr
			}
		}

		if apiErr := txSvc.repos.NewCustomerRepo().BulkDelete(txCtx, params.OwnerAccountID, params.CustomerIDs); apiErr != nil {
			return apiErr
		}

		publisher := audit.NewPublisher()
		outboxRepo := txSvc.repos.NewOutboxRepo()

		for _, customer := range customers {
			changes := audit.ComputeChanges(customer, (*domain.Customer)(nil))

			if apiErr := publisher.Publish(txCtx, outboxRepo, audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionDelete,
				ResourceType: constants.ObjectTypeCustomer,
				ResourceID:   customer.ID,
				Changes:      changes,
			}); apiErr != nil {
				return apiErr
			}
		}

		return nil
	})

	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

// GetFrequentlyOrderedProducts returns the most frequently ordered products for a customer.
// A customer or supplier portal actor reads only its own.
func (s *customerSvcImpl) GetFrequentlyOrderedProducts(ctx context.Context, customerAccountID string) ([]*domain.FrequentlyOrderedProduct, *apierror.APIError) {
	ctx, span := customerSvcTracer.Start(ctx, "service.customer.frequently_ordered_products")
	defer span.End()

	identity, ok := appctx.Identity[*types.Identity](ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsAssignedActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if !identity.IsTargetAccountSet() {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("The OpenMRP-Account-ID header is required."))
	}

	if own := identity.PortalAccountID(); own != nil {
		if customerAccountID != *own {
			return nil, tracing.Trace(span, apierror.NewNotFoundError("Customer not found."))
		}
	} else if identity.IsInternalActor() {
		if identity.IsTargetCustomerAccount() {
			if apiErr := identity.CheckHasPermission(types.PermissionDomainCustomers, types.ActionRead); apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
		} else if identity.IsTargetSupplierAccount() {
			if apiErr := identity.CheckHasPermission(types.PermissionDomainSuppliers, types.ActionRead); apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
		} else {
			if apiErr := identity.CheckHasPermission(types.PermissionDomainItems, types.ActionRead); apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
		}
	}

	if identity.IsExternalTarget() {
		meds := s.mediators()
		if apiErr := meds.ReadAccess.CheckCounterpartyReadAccess(ctx, *identity.ActorAccountID(), identity.Target.AccountID); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}

	ownerAccountID := identity.Target.AccountID

	// A customer who has ordered nothing and an account that is not a customer at all both
	// produce an empty list, so without this read the endpoint answers for any account ID —
	// including one belonging to another seller.
	if _, apiErr := s.repos.NewCustomerRepo().Get(ctx, ownerAccountID, customerAccountID, nil); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return s.repos.NewCustomerRepo().GetFrequentlyOrderedProducts(ctx, ownerAccountID, customerAccountID)
}

// managedOrderNotificationTypeCodes are the notification types the customer-facing recipient endpoints manage. Purchase-order submission prefs (the reverse direction) are intentionally excluded so they are preserved across updates.
var managedOrderNotificationTypeCodes = []string{
	string(constants.AccountRelationNotificationTypeInvoice),
	string(constants.AccountRelationNotificationTypeOrderAcknowledgement),
	string(constants.AccountRelationNotificationTypeShipment),
	string(constants.AccountRelationNotificationTypeStatement),
}

// authorizeCustomerNotificationRecipientAccess resolves the caller identity and gates access to a customer relationship's notification recipients. Read (write=false) mirrors GetFrequentlyOrderedProducts; write additionally requires the order-placing capability (customer users) or Customers/Suppliers update (internal actors).
func (s *customerSvcImpl) authorizeCustomerNotificationRecipientAccess(ctx context.Context, customerAccountID string, write bool) (*types.Identity, *apierror.APIError) {
	identity, ok := appctx.Identity[*types.Identity](ctx)
	if !ok || identity == nil {
		return nil, apierror.NewInvariantViolationError("Identity not found in context.")
	}
	if apiErr := identity.CheckIsAssignedActor(); apiErr != nil {
		return nil, apiErr
	}
	if !identity.IsTargetAccountSet() {
		return nil, apierror.NewAuthenticationError("The OpenMRP-Account-ID header is required.")
	}

	if identity.IsCustomerUser() {
		// Customer users may only read or manage the recipients on their own relationship.
		if customerAccountID != *identity.ActorAccountID() {
			return nil, apierror.NewNotFoundError("Customer not found.")
		}
		if write {
			// Managing order-notification recipients is a customer-side capability tied to placing orders.
			if apiErr := identity.CheckHasRelationCapability(types.PermissionDomainPurchaseOrders, types.ActionCreate); apiErr != nil {
				return nil, apiErr
			}
		}
	} else if identity.IsInternalActor() {
		action := types.ActionRead
		if write {
			action = types.ActionUpdate
		}
		domainCode := types.PermissionDomainCustomers
		if identity.IsTargetSupplierAccount() {
			domainCode = types.PermissionDomainSuppliers
		}
		if apiErr := identity.CheckHasPermission(domainCode, action); apiErr != nil {
			return nil, apiErr
		}
	} else {
		return nil, apierror.NewAuthorizationError("You do not have access to this customer.")
	}

	if identity.IsExternalTarget() {
		meds := s.mediators()
		if apiErr := meds.ReadAccess.CheckCounterpartyReadAccess(ctx, *identity.ActorAccountID(), identity.Target.AccountID); apiErr != nil {
			return nil, apiErr
		}
	}

	return identity, nil
}

// ListCustomerNotificationRecipients returns the default order-notification recipients configured for a customer relationship. Supports both internal and customer actors.
func (s *customerSvcImpl) ListCustomerNotificationRecipients(ctx context.Context, customerAccountID string) ([]domain.NotificationRecipient, *apierror.APIError) {
	ctx, span := customerSvcTracer.Start(ctx, "service.customer.list_notification_recipients")
	defer span.End()

	identity, apiErr := s.authorizeCustomerNotificationRecipientAccess(ctx, customerAccountID, false)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	relationID, apiErr := s.repos.NewAccountRelationRepo().FindRelationByOwnerAndCounterparty(ctx, identity.Target.AccountID, customerAccountID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	recipients, apiErr := s.hydrateNotificationRecipients(ctx, customerAccountID, relationID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return recipients, nil
}

// hydrateNotificationRecipients lists a relation's stored recipient refs and resolves each to a
// full account user on the customer's account. Recipients whose account user no longer exists are
// dropped. Account-user resolution is scoped to customerAccountID (the buyer), which is why it
// happens here rather than through the gateway's target-scoped account-user loader.
func (s *customerSvcImpl) hydrateNotificationRecipients(ctx context.Context, customerAccountID, relationID string) ([]domain.NotificationRecipient, *apierror.APIError) {
	refs, apiErr := s.repos.NewAccountRelationRepo().ListNotificationRecipients(ctx, relationID)
	if apiErr != nil {
		return nil, apiErr
	}
	if len(refs) == 0 {
		return nil, nil
	}

	ids := make([]string, 0, len(refs))
	for _, ref := range refs {
		ids = append(ids, ref.AccountUserID)
	}

	details, apiErr := s.repos.NewAccountUserRepo().GetByIDs(ctx, customerAccountID, ids)
	if apiErr != nil {
		return nil, apiErr
	}
	detailByID := make(map[string]*domain.AccountUserDetail, len(details))
	for _, d := range details {
		detailByID[d.ID] = d
	}

	recipients := make([]domain.NotificationRecipient, 0, len(refs))
	for _, ref := range refs {
		detail, ok := detailByID[ref.AccountUserID]
		if !ok {
			// Account user was removed since it was configured; drop it rather than emit a stub.
			continue
		}
		recipients = append(recipients, domain.NotificationRecipient{
			AccountUser:           detail,
			NotificationTypeCodes: ref.NotificationTypeCodes,
		})
	}

	return recipients, nil
}

// UpdateCustomerNotificationRecipients replaces the managed (invoice / order acknowledgement) default recipients for a customer relationship. The provided list is the full desired set.
func (s *customerSvcImpl) UpdateCustomerNotificationRecipients(ctx context.Context, params domain.UpdateCustomerNotificationRecipientsParams) ([]domain.NotificationRecipient, *apierror.APIError) {
	ctx, span := customerSvcTracer.Start(ctx, "service.customer.update_notification_recipients")
	defer span.End()

	identity, apiErr := s.authorizeCustomerNotificationRecipientAccess(ctx, params.CustomerAccountID, true)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	relationID, apiErr := s.repos.NewAccountRelationRepo().FindRelationByOwnerAndCounterparty(ctx, identity.Target.AccountID, params.CustomerAccountID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *customerSvcImpl) *apierror.APIError {
		before, apiErr := txSvc.replaceNotificationRecipients(txCtx, params.CustomerAccountID, relationID, params.Recipients, "recipients")
		if apiErr != nil {
			return apiErr
		}
		after, apiErr := txSvc.repos.NewAccountRelationRepo().ListNotificationRecipients(txCtx, relationID)
		if apiErr != nil {
			return apiErr
		}
		var changes []audit.FieldChange
		if change, changed := recipientsAuditChange(before, after); changed {
			changes = append(changes, change)
		}
		return audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
			ServiceName:  domain.ServiceName,
			Action:       constants.AuditActionUpdate,
			ResourceType: constants.ObjectTypeCustomer,
			ResourceID:   params.CustomerAccountID,
			Changes:      changes,
		})
	})
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	recipients, apiErr := s.hydrateNotificationRecipients(ctx, params.CustomerAccountID, relationID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return recipients, nil
}

// replaceNotificationRecipients checks recipients and makes them the relation's managed recipients, leaving its
// purchase-order submission recipients alone. param names the request field the recipients came from. Returns the
// recipients the relation held before, for the audit diff.
func (s *customerSvcImpl) replaceNotificationRecipients(ctx context.Context, customerAccountID, relationID string, recipients []domain.NotificationRecipientInput, param string) ([]domain.NotificationRecipientRef, *apierror.APIError) {
	for i, r := range recipients {
		at := fmt.Sprintf("%s[%d]", param, i)
		if r.AccountUserID == "" {
			return nil, apierror.NewMissingFieldError(at+".account_user_id", "Each notification recipient must reference an account user.")
		}
		if len(r.NotificationTypeCodes) == 0 {
			return nil, apierror.NewMissingFieldError(at+".documents", "Each notification recipient must receive at least one document.")
		}
		for _, code := range r.NotificationTypeCodes {
			if !slices.Contains(managedOrderNotificationTypeCodes, code) {
				return nil, apierror.NewInvalidFormatError(at+".documents", fmt.Sprintf("Notification type %q is not supported for customer notifications.", code))
			}
		}
		if _, apiErr := s.repos.NewAccountUserRepo().GetDetailByAccountAndID(ctx, customerAccountID, r.AccountUserID, nil); apiErr != nil {
			if apiErr.Code == apierror.CodeResourceNotFound {
				return nil, apierror.NewNotFoundError("No account user of this customer has the provided ID.").WithParam(at + ".account_user_id")
			}
			return nil, apiErr
		}
	}

	relationRepo := s.repos.NewAccountRelationRepo()
	before, apiErr := relationRepo.ListNotificationRecipients(ctx, relationID)
	if apiErr != nil {
		return nil, apiErr
	}
	if apiErr := relationRepo.DeleteNotificationPreferencesByTypes(ctx, relationID, managedOrderNotificationTypeCodes); apiErr != nil {
		return nil, apiErr
	}
	seen := make(map[string]struct{})
	for _, r := range recipients {
		for _, code := range r.NotificationTypeCodes {
			key := r.AccountUserID + "\x00" + code
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			prefID, apiErr := id.GenID(id.AccountRelationNotificationPreferenceIDPrefix, nil)
			if apiErr != nil {
				return nil, apiErr
			}
			if apiErr := relationRepo.CreateNotificationPreference(ctx, prefID, relationID, r.AccountUserID, code); apiErr != nil {
				return nil, apiErr
			}
		}
	}
	return before, nil
}

// recipientsAuditChange is the notification_recipients change between two sets of a relation's recipients.
func recipientsAuditChange(before, after []domain.NotificationRecipientRef) (audit.FieldChange, bool) {
	old, updated := managedRecipientsAuditValue(before), managedRecipientsAuditValue(after)
	if slices.EqualFunc(old, updated, managedRecipientAuditValue.equal) {
		return audit.FieldChange{}, false
	}
	return audit.NewFieldChange("notification_recipients", old, updated), true
}

// resolveParentCustomer is the relation of the owner's customer parentID, which a customer is about to be placed
// under. childRelationID is the customer's own relation, nil for one being created; the parent may not be the
// customer or one of its descendants. An archived customer can't take new child customers.
func resolveParentCustomer(ctx context.Context, repos domain.RepoFactory, ownerAccountID, parentID string, childRelationID *string) (string, *apierror.APIError) {
	const param = "parent_customer_id"
	parent, apiErr := repos.NewCustomerRepo().Get(ctx, ownerAccountID, parentID, nil)
	if apierror.HasCode(apiErr, apierror.CodeResourceNotFound) {
		return "", apierror.NewNotFoundError("No customer found with the provided ID.").WithParam(param)
	}
	if apiErr != nil {
		return "", apiErr
	}
	if apiErr := domain.RequireAssignable(param, "customer", parent.ArchiveStatus); apiErr != nil {
		return "", apiErr
	}
	parentRelationID, apiErr := repos.NewCustomerRepo().GetRelationID(ctx, ownerAccountID, parentID)
	if apiErr != nil {
		return "", apiErr
	}
	if childRelationID != nil {
		if apiErr := checkNoHierarchyCycle(ctx, repos.NewAccountRelationRepo(), parentRelationID, *childRelationID); apiErr != nil {
			if apierror.HasCode(apiErr, apierror.CodeResourceConflict) {
				return "", apierror.NewInvalidFormatError(param, "A customer can't be placed under itself or one of its own child customers.")
			}
			return "", apiErr
		}
	}
	return parentRelationID, nil
}

// MergeCustomers accepts a merge of source customers into a target customer: it checks the customers exist and
// records the merge on a job, which ExecuteMergeCustomers runs.
func (s *customerSvcImpl) MergeCustomers(ctx context.Context, params domain.MergeCustomersParams) (*domain.Job, *apierror.APIError) {
	ctx, span := customerSvcTracer.Start(ctx, "service.customer.merge.enqueue")
	defer span.End()

	identity, ok := appctx.Identity[*types.Identity](ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainCustomers, types.ActionUpdate); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainCustomers, types.ActionDelete); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if !identity.IsTargetAccountSet() {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("The OpenMRP-Account-ID header is required."))
	}

	params.OwnerAccountID = identity.Target.AccountID

	if len(params.SourceCustomerIDs) == 0 {
		return nil, tracing.Trace(span, apierror.NewMissingFieldError("source_customer_ids", "Give at least one customer to merge."))
	}
	if slices.Contains(params.SourceCustomerIDs, params.TargetCustomerID) {
		return nil, tracing.Trace(span, apierror.NewInvalidFormatError("source_customer_ids", "Target customer cannot be in the source customer list."))
	}
	seen := make(map[string]bool, len(params.SourceCustomerIDs))
	for _, sourceID := range params.SourceCustomerIDs {
		if seen[sourceID] {
			return nil, tracing.Trace(span, apierror.NewInvalidFormatError("source_customer_ids", "Duplicate source customer IDs are not allowed."))
		}
		seen[sourceID] = true
	}

	meds := s.mediators()
	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, identity)
	if apiErr != nil {
		return nil, apiErr
	}

	switch domain.RecoveryPoint(idempotencyKey.RecoveryPoint) {
	case domain.RecoveryPointFinished:
		cached, err := idemresponse.UnmarshalCachedResponse[domain.Job](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		// Reads the database, so it runs after the key is claimed: a customer that doesn't exist is a bad request the
		// caller learns about now rather than from a failed job.
		customerRepo := s.repos.NewCustomerRepo()
		if _, apiErr := customerRepo.Get(ctx, params.OwnerAccountID, params.TargetCustomerID, nil); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		for _, sourceID := range params.SourceCustomerIDs {
			if _, apiErr := customerRepo.Get(ctx, params.OwnerAccountID, sourceID, nil); apiErr != nil {
				if apierror.HasCode(apiErr, apierror.CodeResourceNotFound) {
					return nil, tracing.Trace(span, apierror.NewNotFoundError("No customer found with the provided ID.").WithParam("source_customer_ids"))
				}
				return nil, tracing.Trace(span, apiErr)
			}
		}

		jobItems, err := json.Marshal(domain.MergeCustomersJob{TargetCustomerID: params.TargetCustomerID, SourceCustomerIDs: params.SourceCustomerIDs})
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Failed to marshal merge payload."))
		}
		createdByID := jobCreatedByID(ctx, s.repos, params.OwnerAccountID, identity)

		var raisedJob *domain.Job
		apiErr = s.txManager.WithTx(ctx, func(txCtx context.Context, txRepos domain.RepoFactory) *apierror.APIError {
			job, apiErr := s.jobSvcFactory.Build(txRepos).CreateJob(txCtx, domain.CreateJobServiceParams{
				JobItems:     jobItems,
				Type:         constants.JobTypeMergeCustomers,
				ResourceType: constants.ObjectTypeCustomer,
				CreatedByID:  createdByID,
			})
			if apiErr != nil {
				return apiErr
			}
			raisedJob = job

			// Enqueued through the outbox inside the transaction, so the command is published if and only if the job
			// commits.
			payloadJSON, err := json.Marshal(domain.BulkOperationJobEvent{JobID: job.ID})
			if err != nil {
				return apierror.NewInternalError(err, "Failed to marshal merge job event.")
			}
			msg := contracts.AmqpMessage{Data: payloadJSON, Identity: identity}
			if requestID, ok := appctx.GetRequestID(txCtx); ok {
				msg.RequestID = requestID
			}
			if _, err := txRepos.NewOutboxRepo().Create(txCtx, messaging.OutboxMessageInput{
				ServiceName: domain.ServiceName,
				MessageType: string(messaging.MergeCustomers.RoutingKey()),
				Destination: messaging.ApplicationExchange,
				RoutingKey:  string(messaging.MergeCustomers.RoutingKey()),
				Payload:     msg,
			}); err != nil {
				return apierror.NewInternalError(err, "Failed to create merge outbox message.")
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

// ExecuteMergeCustomers runs an accepted merge and settles its job. The inbox de-dupes redeliveries; the terminal
// guard covers a replay that outlives its record.
func (s *customerSvcImpl) ExecuteMergeCustomers(ctx context.Context, event domain.BulkOperationJobEvent) *apierror.APIError {
	ctx, span := customerSvcTracer.Start(ctx, "service.customer.merge.execute")
	defer span.End()

	if event.JobID == "" {
		return tracing.Trace(span, apierror.NewValidationError("Merge job event is missing a job."))
	}
	identity, ok := appctx.Identity[*types.Identity](ctx)
	if !ok || identity == nil {
		return tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}
	ownerAccountID := identity.Target.AccountID

	jobs := s.jobSvcFactory.Build(s.repos)
	job, apiErr := jobs.GetJobForExecution(ctx, event.JobID)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if job.IsTerminal() {
		return nil
	}

	var payload domain.MergeCustomersJob
	if err := json.Unmarshal(job.JobItems, &payload); err != nil {
		apiErr := apierror.NewInternalError(err, "Job items are not a merge payload.")
		jobs.FailJob(ctx, domain.FailJobParams{JobID: job.ID, ApiErr: apiErr})
		return tracing.Trace(span, apiErr)
	}

	if _, apiErr := jobs.StartJob(ctx, domain.StartJobParams{JobID: job.ID}); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *customerSvcImpl) *apierror.APIError {
		target, apiErr := mergeCustomersInTx(txCtx, txSvc.repos, ownerAccountID, payload)
		if apiErr != nil {
			return apiErr
		}
		// Settled in the same transaction as the writes, so a completed job always has its merge.
		return s.jobSvcFactory.Build(txSvc.repos).CompleteJob(txCtx, domain.CompleteJobParams{
			JobID: job.ID,
			Results: []domain.RowResult{{
				Index:        0,
				Status:       constants.JobResultStatusUpdated,
				ResourceType: constants.ObjectTypeCustomer,
				ID:           target.ID,
				Name:         &target.Name,
			}},
		})
	})
	if apiErr != nil {
		jobs.FailJob(ctx, domain.FailJobParams{JobID: job.ID, ApiErr: apiErr})
		return tracing.Trace(span, apiErr)
	}
	return nil
}

// mergeCustomersInTx folds job's source customers into its target inside the caller's transaction and returns the
// merged target.
func mergeCustomersInTx(ctx context.Context, repos domain.RepoFactory, ownerAccountID string, job domain.MergeCustomersJob) (*domain.Customer, *apierror.APIError) {
	customerRepo := repos.NewCustomerRepo()
	targetOld, apiErr := customerRepo.Get(ctx, ownerAccountID, job.TargetCustomerID, []string{"price_groups"})
	if apiErr != nil {
		return nil, apiErr
	}
	sourceCustomers := make([]*domain.Customer, 0, len(job.SourceCustomerIDs))
	for _, sourceID := range job.SourceCustomerIDs {
		source, apiErr := customerRepo.Get(ctx, ownerAccountID, sourceID, []string{"price_groups"})
		if apiErr != nil {
			return nil, apiErr
		}
		sourceCustomers = append(sourceCustomers, source)
	}

	// Open orders moving onto the target take its default rep, as if they had been placed under it. An exempt target earns no commission, so its orders keep theirs.
	retagRepID := targetOld.DefaultSalesRepID
	if retagRepID != nil {
		exempt, apiErr := customerRepo.IsCommissionExempt(ctx, ownerAccountID, job.TargetCustomerID)
		if apiErr != nil {
			return nil, apiErr
		}
		if exempt {
			retagRepID = nil
		}
	}

	targetRelationID, apiErr := customerRepo.GetRelationID(ctx, ownerAccountID, job.TargetCustomerID)
	if apiErr != nil {
		return nil, apiErr
	}
	sourceRelationIDs := make([]string, 0, len(job.SourceCustomerIDs))
	for _, sourceID := range job.SourceCustomerIDs {
		relID, apiErr := customerRepo.GetRelationID(ctx, ownerAccountID, sourceID)
		if apiErr != nil {
			return nil, apiErr
		}
		sourceRelationIDs = append(sourceRelationIDs, relID)
	}

	if retagRepID != nil {
		if apiErr := retagMergedOrders(ctx, repos, ownerAccountID, job.SourceCustomerIDs, *retagRepID); apiErr != nil {
			return nil, apiErr
		}
	}

	// Phase 1: Reassign all foreign key references from sources to target.
	if apiErr := customerRepo.MergeOrders(ctx, ownerAccountID, job.TargetCustomerID, job.SourceCustomerIDs); apiErr != nil {
		return nil, apiErr
	}
	if apiErr := customerRepo.MergeInvoices(ctx, ownerAccountID, job.TargetCustomerID, job.SourceCustomerIDs); apiErr != nil {
		return nil, apiErr
	}
	if apiErr := customerRepo.MergeShipments(ctx, ownerAccountID, job.TargetCustomerID, job.SourceCustomerIDs); apiErr != nil {
		return nil, apiErr
	}
	if apiErr := customerRepo.MergeDeliveries(ctx, ownerAccountID, job.TargetCustomerID, job.SourceCustomerIDs); apiErr != nil {
		return nil, apiErr
	}
	if apiErr := customerRepo.MergeTransactions(ctx, ownerAccountID, job.TargetCustomerID, job.SourceCustomerIDs); apiErr != nil {
		return nil, apiErr
	}
	if apiErr := customerRepo.MergeAccountPrices(ctx, ownerAccountID, job.TargetCustomerID, job.SourceCustomerIDs); apiErr != nil {
		return nil, apiErr
	}
	if apiErr := customerRepo.MergeInventoryReceipts(ctx, ownerAccountID, job.TargetCustomerID, job.SourceCustomerIDs); apiErr != nil {
		return nil, apiErr
	}
	if apiErr := customerRepo.MergeReceivingOrders(ctx, ownerAccountID, job.TargetCustomerID, job.SourceCustomerIDs); apiErr != nil {
		return nil, apiErr
	}
	if apiErr := customerRepo.MergeInventoryIssues(ctx, job.TargetCustomerID, job.SourceCustomerIDs); apiErr != nil {
		return nil, apiErr
	}

	// Phase 2: Consolidate price groups — move non-duplicates from source to target, delete rest.
	targetPriceGroupIDs, apiErr := customerRepo.GetRelationPriceGroupIDs(ctx, targetRelationID)
	if apiErr != nil {
		return nil, apiErr
	}
	targetPGSet := make(map[string]bool, len(targetPriceGroupIDs))
	for _, gid := range targetPriceGroupIDs {
		targetPGSet[gid] = true
	}
	sourcePriceGroups, apiErr := customerRepo.GetRelationsPriceGroups(ctx, sourceRelationIDs)
	if apiErr != nil {
		return nil, apiErr
	}
	var pgToMove, pgToDelete []string
	for _, pg := range sourcePriceGroups {
		if !targetPGSet[pg.AccountGroupID] {
			pgToMove = append(pgToMove, pg.ID)
			targetPGSet[pg.AccountGroupID] = true
		} else {
			pgToDelete = append(pgToDelete, pg.ID)
		}
	}
	if len(pgToMove) > 0 {
		if apiErr := customerRepo.MoveRelationPriceGroups(ctx, targetRelationID, pgToMove); apiErr != nil {
			return nil, apiErr
		}
	}
	if len(pgToDelete) > 0 {
		if apiErr := customerRepo.DeletePriceGroupsByIDs(ctx, pgToDelete); apiErr != nil {
			return nil, apiErr
		}
	}

	// Phase 3: Consolidate product lines — move non-duplicates from source to target, delete rest.
	targetProductLineIDs, apiErr := customerRepo.GetRelationProductLineIDs(ctx, targetRelationID)
	if apiErr != nil {
		return nil, apiErr
	}
	targetPLSet := make(map[string]bool, len(targetProductLineIDs))
	for _, plID := range targetProductLineIDs {
		targetPLSet[plID] = true
	}
	sourceProductLines, apiErr := customerRepo.GetRelationsProductLines(ctx, sourceRelationIDs)
	if apiErr != nil {
		return nil, apiErr
	}
	var plToMove, plToDelete []string
	for _, pl := range sourceProductLines {
		if !targetPLSet[pl.ProductLineID] {
			plToMove = append(plToMove, pl.ID)
			targetPLSet[pl.ProductLineID] = true
		} else {
			plToDelete = append(plToDelete, pl.ID)
		}
	}
	if len(plToMove) > 0 {
		if apiErr := customerRepo.MoveRelationProductLines(ctx, targetRelationID, plToMove); apiErr != nil {
			return nil, apiErr
		}
	}
	if len(plToDelete) > 0 {
		if apiErr := customerRepo.DeleteProductLinesByIDs(ctx, plToDelete); apiErr != nil {
			return nil, apiErr
		}
	}

	// Phase 4: Delete source notification preferences (user-bound, can't transfer).
	if apiErr := customerRepo.DeleteNotificationPreferences(ctx, sourceRelationIDs); apiErr != nil {
		return nil, apiErr
	}

	// Phase 5: Re-parent child account relations.
	if apiErr := customerRepo.ReparentChildRelations(ctx, targetRelationID, sourceRelationIDs); apiErr != nil {
		return nil, apiErr
	}

	// Phase 6: Consolidate account addresses — create missing on target, delete source.
	targetAddressIDs, apiErr := customerRepo.GetAccountAddressIDs(ctx, job.TargetCustomerID)
	if apiErr != nil {
		return nil, apiErr
	}
	targetAddrSet := make(map[string]bool, len(targetAddressIDs))
	for _, addrID := range targetAddressIDs {
		targetAddrSet[addrID] = true
	}
	for _, sourceID := range job.SourceCustomerIDs {
		sourceAddrIDs, apiErr := customerRepo.GetAccountAddressIDs(ctx, sourceID)
		if apiErr != nil {
			return nil, apiErr
		}
		for _, addrID := range sourceAddrIDs {
			if !targetAddrSet[addrID] {
				newID, apiErr := id.GenID(id.AccountAddressIDPrefix, nil)
				if apiErr != nil {
					return nil, apiErr
				}
				if apiErr := customerRepo.InsertAccountAddress(ctx, newID, job.TargetCustomerID, addrID); apiErr != nil {
					return nil, apiErr
				}
				targetAddrSet[addrID] = true
			}
		}
		if apiErr := customerRepo.DeleteAccountAddresses(ctx, sourceID); apiErr != nil {
			return nil, apiErr
		}
	}

	// Phase 7: Consolidate account users — reassign non-duplicates, delete dupes.
	targetUsers, apiErr := customerRepo.GetAccountUsers(ctx, job.TargetCustomerID)
	if apiErr != nil {
		return nil, apiErr
	}
	targetUserSet := make(map[string]bool, len(targetUsers))
	for _, u := range targetUsers {
		targetUserSet[u.UserID] = true
	}
	for _, sourceID := range job.SourceCustomerIDs {
		sourceUsers, apiErr := customerRepo.GetAccountUsers(ctx, sourceID)
		if apiErr != nil {
			return nil, apiErr
		}
		var usersToMove []string
		for _, u := range sourceUsers {
			if !targetUserSet[u.UserID] {
				usersToMove = append(usersToMove, u.ID)
				targetUserSet[u.UserID] = true
			}
		}
		if len(usersToMove) > 0 {
			if apiErr := customerRepo.MoveAccountUsers(ctx, job.TargetCustomerID, usersToMove); apiErr != nil {
				return nil, apiErr
			}
		}
		if apiErr := customerRepo.DeleteAccountUsers(ctx, sourceID); apiErr != nil {
			return nil, apiErr
		}
	}

	// Phase 8: Delete source customer relations.
	if apiErr := customerRepo.BulkDelete(ctx, ownerAccountID, job.SourceCustomerIDs); apiErr != nil {
		return nil, apiErr
	}

	result, apiErr := customerRepo.Get(ctx, ownerAccountID, job.TargetCustomerID, []string{"price_groups"})
	if apiErr != nil {
		return nil, apiErr
	}

	// Publish audit events: delete for each source, update for target.
	publisher := audit.NewPublisher()
	outboxRepo := repos.NewOutboxRepo()
	for _, source := range sourceCustomers {
		if apiErr := publisher.Publish(ctx, outboxRepo, audit.EventData{
			ServiceName:  domain.ServiceName,
			Action:       constants.AuditActionDelete,
			ResourceType: constants.ObjectTypeCustomer,
			ResourceID:   source.ID,
			Changes:      audit.ComputeChanges(source, (*domain.Customer)(nil)),
		}); apiErr != nil {
			return nil, apiErr
		}
	}
	if apiErr := publisher.Publish(ctx, outboxRepo, audit.EventData{
		ServiceName:  domain.ServiceName,
		Action:       constants.AuditActionUpdate,
		ResourceType: constants.ObjectTypeCustomer,
		ResourceID:   result.ID,
		Changes:      audit.ComputeChanges(targetOld, result),
	}); apiErr != nil {
		return nil, apiErr
	}
	return result, nil
}

// retagMergedOrders moves the source customers' open, uninvoiced orders onto salesRepID. Invoiced orders keep their rep so commission already reported does not move.
func retagMergedOrders(txCtx context.Context, repos domain.RepoFactory, ownerAccountID string, sourceCustomerIDs []string, salesRepID string) *apierror.APIError {
	customerRepo := repos.NewCustomerRepo()
	orders, apiErr := customerRepo.ListMergeRetaggableOrders(txCtx, ownerAccountID, sourceCustomerIDs, salesRepID)
	if apiErr != nil || len(orders) == 0 {
		return apiErr
	}
	orderIDs := make([]string, len(orders))
	for i, o := range orders {
		orderIDs[i] = o.OrderID
	}
	if apiErr := customerRepo.SetOrdersSalesRep(txCtx, ownerAccountID, salesRepID, orderIDs); apiErr != nil {
		return apiErr
	}
	publisher := audit.NewPublisher()
	outboxRepo := repos.NewOutboxRepo()
	for _, o := range orders {
		if apiErr := publisher.Publish(txCtx, outboxRepo, audit.EventData{
			ServiceName:  domain.ServiceName,
			Action:       constants.AuditActionUpdate,
			ResourceType: constants.ObjectTypeSalesOrder,
			ResourceID:   o.OrderID,
			Changes:      []audit.FieldChange{audit.NewFieldChange("sales_rep_id", o.SalesRepID, &salesRepID)},
			Metadata:     map[string]any{"source": "customer_merge"},
		}); apiErr != nil {
			return apiErr
		}
	}
	return nil
}

// checkCustomerReadPermission checks the appropriate read permission based on the identity context.
// Internal actors need customers:read for their own account, or customers:read / suppliers:read for external accounts.
func checkCustomerReadPermission(identity *types.Identity) *apierror.APIError {
	if !identity.IsInternalActor() {
		return nil
	}
	if identity.IsTargetCustomerAccount() {
		return identity.CheckHasPermission(types.PermissionDomainCustomers, types.ActionRead)
	}
	if identity.IsTargetSupplierAccount() {
		return identity.CheckHasPermission(types.PermissionDomainSuppliers, types.ActionRead)
	}
	return identity.CheckHasPermission(types.PermissionDomainCustomers, types.ActionRead)
}

// checkCustomerDefaultAddresses refuses a default billing or shipping address that changes to one the
// customer's account does not hold. An unchanged default is left alone.
func checkCustomerDefaultAddresses(ctx context.Context, repo domain.CustomerRepo, params domain.UpdateCustomerParams, old *domain.Customer) *apierror.APIError {
	var held []string
	for _, ref := range []struct {
		value field.Clearable[string]
		was   *string
		param string
	}{
		{params.BillToAddressID, old.BillToAddressID, "defaults.bill_to_address_id"},
		{params.ShipToAddressID, old.ShipToAddressID, "defaults.ship_to_address_id"},
	} {
		addressID, ok := ref.value.Value()
		if !ok || (ref.was != nil && *ref.was == addressID) {
			continue
		}
		if held == nil {
			ids, apiErr := repo.GetAccountAddressIDs(ctx, params.CustomerAccountID)
			if apiErr != nil {
				return apiErr
			}
			held = append(ids, "")
		}
		if !slices.Contains(held, addressID) {
			return apierror.NewNotFoundError("No address of this customer has the provided ID.").WithParam(ref.param)
		}
	}
	return nil
}

// ensureAccountAddressLink creates an account_address record linking the given address to the account, if one does not already exist.
func ensureAccountAddressLink(ctx context.Context, repo domain.CustomerRepo, accountID, addressID string) *apierror.APIError {
	existingIDs, apiErr := repo.GetAccountAddressIDs(ctx, accountID)
	if apiErr != nil {
		return apiErr
	}
	if slices.Contains(existingIDs, addressID) {
		return nil
	}
	newID, apiErr := id.GenID(id.AccountAddressIDPrefix, nil)
	if apiErr != nil {
		return apiErr
	}
	return repo.InsertAccountAddress(ctx, newID, accountID, addressID)
}

// addressParamsEqual returns true when two CreateAddressParams represent the same physical address (ignoring AccountID which is set later).
func addressParamsEqual(a, b domain.CreateAddressParams) bool {
	return a.Name == b.Name &&
		optStrEqual(a.Phone, b.Phone) &&
		optStrEqual(a.Email, b.Email) &&
		a.IsDropShip == b.IsDropShip &&
		optStrEqual(a.StreetLine1, b.StreetLine1) &&
		optStrEqual(a.StreetLine2, b.StreetLine2) &&
		optStrEqual(a.Locality, b.Locality) &&
		optStrEqual(a.State, b.State) &&
		optStrEqual(a.PostalCode, b.PostalCode) &&
		a.Country == b.Country
}

func optStrEqual(a, b *string) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

// customerAuditIncludes merges user-requested includes with the includes required for correct audit change tracking (price_groups and notification_preferences).
func customerAuditIncludes(userIncludes []string) []string {
	auditRequired := []string{"price_groups", "notification_preferences"}
	merged := make([]string, len(auditRequired))
	copy(merged, auditRequired)
	for _, inc := range userIncludes {
		if !slices.Contains(merged, inc) {
			merged = append(merged, inc)
		}
	}
	return merged
}

// stripeCustomerFieldsChanged reports whether an update touched anything Stripe mirrors: the customer's email, name, or number.
func stripeCustomerFieldsChanged(old, updated *domain.Customer) bool {
	if old == nil || updated == nil {
		return updated != nil
	}
	return !optStrEqual(old.Email, updated.Email) ||
		old.Name != updated.Name ||
		old.Number != updated.Number
}

// enqueueStripeCustomerSync writes the outbox command that mirrors a customer onto the account's connected Stripe integration.
//
// It is published inside the caller's transaction so the command commits with the customer row: a Stripe write that fails, or an account with no Stripe integration at all, must never fail or roll back the customer mutation itself. The consumer no-ops when the integration is absent, so this is published unconditionally rather than probing for one here.
func enqueueStripeCustomerSync(ctx context.Context, repos domain.RepoFactory, ownerAccountID, customerAccountID string) *apierror.APIError {
	payload, err := json.Marshal(domain.SyncStripeCustomerEvent{
		OwnerAccountID:    ownerAccountID,
		CustomerAccountID: customerAccountID,
	})
	if err != nil {
		return apierror.NewInternalError(err, "Failed to marshal Stripe customer sync event.")
	}

	msg := contracts.AmqpMessage{Data: payload}
	if identity, ok := appctx.Identity[*types.Identity](ctx); ok {
		msg.Identity = identity
	}
	if requestID, ok := appctx.GetRequestID(ctx); ok {
		msg.RequestID = requestID
	}

	if _, err := repos.NewOutboxRepo().Create(ctx, messaging.OutboxMessageInput{
		ServiceName: "core-service",
		MessageType: string(contracts.CoreCmdSyncStripeCustomer),
		Destination: messaging.ApplicationExchange,
		RoutingKey:  string(contracts.CoreCmdSyncStripeCustomer),
		Payload:     msg,
	}); err != nil {
		return apierror.NewInternalError(err, "Failed to create outbox message for Stripe customer sync.")
	}

	return nil
}

// managedRecipientAuditValue is one default order recipient as a customer's audit diff shows it.
type managedRecipientAuditValue struct {
	AccountUserID         string   `json:"account_user_id"`
	NotificationTypeCodes []string `json:"notification_type_codes"`
}

func (a managedRecipientAuditValue) equal(b managedRecipientAuditValue) bool {
	return a.AccountUserID == b.AccountUserID && slices.Equal(a.NotificationTypeCodes, b.NotificationTypeCodes)
}

// managedRecipientsAuditValue keeps the managed order types of each recipient, sorted so the diff ignores row order.
func managedRecipientsAuditValue(refs []domain.NotificationRecipientRef) []managedRecipientAuditValue {
	out := []managedRecipientAuditValue{}
	for _, ref := range refs {
		codes := slices.DeleteFunc(slices.Clone(ref.NotificationTypeCodes), func(code string) bool {
			return !slices.Contains(managedOrderNotificationTypeCodes, code)
		})
		if len(codes) == 0 {
			continue
		}
		slices.Sort(codes)
		out = append(out, managedRecipientAuditValue{AccountUserID: ref.AccountUserID, NotificationTypeCodes: codes})
	}
	slices.SortFunc(out, func(a, b managedRecipientAuditValue) int { return strings.Compare(a.AccountUserID, b.AccountUserID) })
	return out
}
