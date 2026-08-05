package service

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strconv"

	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/shared/appctx"
	"github.com/augno/api/shared/audit"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/contracts"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/id"
	"github.com/augno/api/shared/idempotency"
	"github.com/augno/api/shared/messaging"
	"github.com/augno/api/shared/tracing"
)

var customerSvcTracer = tracing.GetTracer("core-service.service.customer")

type customerSvcImpl struct {
	repos           domain.RepoFactory
	mediatorFactory domain.MediatorFactory
	txManager       TransactionManager
}

type CustomerSvcConfig struct {
	// Repos (required) is the repository factory.
	Repos domain.RepoFactory

	// MediatorFactory (required) builds the mediators used by this service.
	MediatorFactory domain.MediatorFactory

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

	identity, ok := appctx.GetIdentityFromContext(ctx)
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

	return s.repos.NewCustomerRepo().List(ctx, params)
}

// GetCustomer retrieves a single customer by account ID. Supports both internal and customer actors.
func (s *customerSvcImpl) GetCustomer(ctx context.Context, customerAccountID string, includes []string) (*domain.Customer, *apierror.APIError) {
	ctx, span := customerSvcTracer.Start(ctx, "service.customer.get")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
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
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("The Augno-Account-ID header is required."))
	}

	if identity.IsCustomerUser() {
		if customerAccountID != *identity.ActorAccountID() {
			return nil, tracing.Trace(span, apierror.NewResourceNotFoundError("Customer not found."))
		}
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

	return customer, nil
}

// CreateCustomer creates a new customer account with idempotency.
func (s *customerSvcImpl) CreateCustomer(ctx context.Context, params domain.CreateCustomerParams) (*domain.Customer, *apierror.APIError) {
	ctx, span := customerSvcTracer.Start(ctx, "service.customer.create")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
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

	meds := s.mediators()

	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, identity)
	if apiErr != nil {
		return nil, apiErr
	}

	switch domain.RecoveryPoint(idempotencyKey.RecoveryPoint) {
	case domain.RecoveryPointFinished:
		cached, err := idempotency.UnmarshalCachedResponse[domain.Customer](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.Customer
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *customerSvcImpl) *apierror.APIError {
			txCustomerRepo := txSvc.repos.NewCustomerRepo()

			// Validate that the sales rep account user ID belongs to this account.
			if params.DefaultSalesRepID != nil {
				txAccountUserRepo := txSvc.repos.NewAccountUserRepo()
				_, apiErr := txAccountUserRepo.GetDetailByAccountAndID(txCtx, params.OwnerAccountID, *params.DefaultSalesRepID, nil)
				if apiErr != nil {
					if apiErr.Code != apierror.ErrorCodeResourceNotFound {
						return apiErr
					}
					return apierror.NewResourceNotFoundError("No sales rep found with the provided ID.").WithParam("default_sales_rep_id")
				}
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
					return apierror.NewConflictErrorWithParam("A customer with this number already exists.", "number")
				}
			} else {
				// Auto-generate the next customer number.
				nextNum, apiErr := txCustomerRepo.GetNextCustomerNumber(txCtx, params.OwnerAccountID)
				if apiErr != nil {
					return apiErr
				}
				customerNumber = strconv.FormatInt(nextNum, 10)

				sysPropertyID, apiErr := id.GenID(id.SysPropertyIDPrefix, nil)
				if apiErr != nil {
					return apiErr
				}
				if apiErr := txCustomerRepo.UpdateNextCustomerNumber(txCtx, sysPropertyID, params.OwnerAccountID, customerNumber); apiErr != nil {
					return apiErr
				}
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

	identity, ok := appctx.GetIdentityFromContext(ctx)
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
		cached, err := idempotency.UnmarshalCachedResponse[domain.Customer](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.Customer
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *customerSvcImpl) *apierror.APIError {
			txCustomerRepo := txSvc.repos.NewCustomerRepo()

			old, apiErr := txCustomerRepo.Get(txCtx, params.OwnerAccountID, params.CustomerAccountID, []string{"price_groups", "notification_preferences", "bill_to_address", "ship_to_address"})
			if apiErr != nil {
				return apiErr
			}

			if params.DefaultSalesRepID.IsSet() {
				repID, _ := params.DefaultSalesRepID.Value()
				txAccountUserRepo := txSvc.repos.NewAccountUserRepo()
				_, apiErr := txAccountUserRepo.GetDetailByAccountAndID(txCtx, params.OwnerAccountID, repID, nil)
				if apiErr != nil {
					if apiErr.Code != apierror.ErrorCodeResourceNotFound {
						return apiErr
					}
					return apierror.NewResourceNotFoundError("No sales rep found with the provided ID.").WithParam("default_sales_rep_id")
				}
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
			params.Note = params.Note.BackfillUnsetPtr(old.Note)
			params.CarrierBillingAccount = params.CarrierBillingAccount.BackfillUnsetPtr(old.CarrierBillingAccount)
			params.BillToAddressID = params.BillToAddressID.BackfillUnsetPtr(old.BillToAddressID)
			params.ShipToAddressID = params.ShipToAddressID.BackfillUnsetPtr(old.ShipToAddressID)

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
					return apierror.NewConflictErrorWithParam("This customer number is already taken.", "number")
				}
			}

			// Get the relation ID for this customer.
			relationID, apiErr := txCustomerRepo.GetRelationID(txCtx, params.OwnerAccountID, params.CustomerAccountID)
			if apiErr != nil {
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

			// Update account name if provided.
			if params.Name != nil {
				if apiErr := txCustomerRepo.UpdateName(txCtx, params.CustomerAccountID, *params.Name); apiErr != nil {
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

			changes := audit.ComputeChanges(old, result)

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

	identity, ok := appctx.GetIdentityFromContext(ctx)
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
	customer, apiErr := repo.Get(ctx, params.OwnerAccountID, params.CustomerAccountID, []string{"price_groups", "notification_preferences"})
	if apiErr != nil {
		if apierror.IsNotFound(apiErr) {
			wasDeleted, deletedCheckErr := s.repos.NewDeletedRecordRepo().Exists(ctx, constants.DeletedRecordResourceTypeCustomer, params.CustomerAccountID)
			if deletedCheckErr != nil {
				return tracing.Trace(span, deletedCheckErr)
			}
			if wasDeleted {
				return tracing.Trace(
					span,
					apierror.NewAlreadyDeletedError("This customer has already been deleted and can no longer be modified."),
				)
			}
		}
		return tracing.Trace(span, apiErr)
	}

	salesOrderRepo := s.repos.NewSalesOrderRepo()
	orderCount, apiErr := salesOrderRepo.CountSalesOrdersForBuyerAccounts(ctx, params.OwnerAccountID, []string{params.CustomerAccountID})
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if orderCount > 0 {
		return tracing.Trace(span, apierror.NewResourceConflictError(
			"Cannot delete this customer while sales orders still reference them. Delete or reassign those orders, or merge customers first.",
		))
	}

	apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *customerSvcImpl) *apierror.APIError {
		if apiErr := txSvc.repos.NewDeletedRecordRepo().Create(txCtx, constants.DeletedRecordResourceTypeCustomer, customer.ID, customer); apiErr != nil {
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

	identity, ok := appctx.GetIdentityFromContext(ctx)
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
		customer, apiErr := customerRepo.Get(ctx, params.OwnerAccountID, customerID, []string{"price_groups", "notification_preferences"})
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
		return tracing.Trace(span, apierror.NewResourceConflictError(
			"Cannot delete customers while sales orders still reference them. Delete or reassign those orders, or merge customers first.",
		))
	}

	apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *customerSvcImpl) *apierror.APIError {
		for _, customer := range customers {
			if apiErr := txSvc.repos.NewDeletedRecordRepo().Create(txCtx, constants.DeletedRecordResourceTypeCustomer, customer.ID, customer); apiErr != nil {
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
// Supports customer actor access.
func (s *customerSvcImpl) GetFrequentlyOrderedProducts(ctx context.Context, customerAccountID string) ([]*domain.FrequentlyOrderedProduct, *apierror.APIError) {
	ctx, span := customerSvcTracer.Start(ctx, "service.customer.frequently_ordered_products")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsAssignedActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if !identity.IsTargetAccountSet() {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("The Augno-Account-ID header is required."))
	}

	if identity.IsCustomerUser() {
		if customerAccountID != *identity.ActorAccountID() {
			return nil, tracing.Trace(span, apierror.NewResourceNotFoundError("Customer not found."))
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

	return s.repos.NewCustomerRepo().GetFrequentlyOrderedProducts(ctx, ownerAccountID, customerAccountID)
}

// managedOrderNotificationTypeCodes are the notification types the customer-facing recipient endpoints manage. Purchase-order submission prefs (the reverse direction) are intentionally excluded so they are preserved across updates.
var managedOrderNotificationTypeCodes = []string{
	string(constants.AccountRelationNotificationTypeInvoice),
	string(constants.AccountRelationNotificationTypeOrderAcknowledgement),
}

// authorizeCustomerNotificationRecipientAccess resolves the caller identity and gates access to a customer relationship's notification recipients. Read (write=false) mirrors GetFrequentlyOrderedProducts; write additionally requires the order-placing capability (customer users) or Customers/Suppliers update (internal actors).
func (s *customerSvcImpl) authorizeCustomerNotificationRecipientAccess(ctx context.Context, customerAccountID string, write bool) (*types.Identity, *apierror.APIError) {
	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, apierror.NewInvariantViolationError("Identity not found in context.")
	}
	if apiErr := identity.CheckIsAssignedActor(); apiErr != nil {
		return nil, apiErr
	}
	if !identity.IsTargetAccountSet() {
		return nil, apierror.NewAuthenticationError("The Augno-Account-ID header is required.")
	}

	if identity.IsCustomerUser() {
		// Customer users may only read or manage the recipients on their own relationship.
		if customerAccountID != *identity.ActorAccountID() {
			return nil, apierror.NewResourceNotFoundError("Customer not found.")
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

	// Validate the requested recipients before touching the database: every notification type must be a managed order type, and every account user must belong to the customer's account.
	for _, r := range params.Recipients {
		if r.AccountUserID == "" {
			return nil, tracing.Trace(span, apierror.NewValidationError("Each notification recipient must reference an account user."))
		}
		if len(r.NotificationTypeCodes) == 0 {
			return nil, tracing.Trace(span, apierror.NewValidationError("Each notification recipient must include at least one notification type."))
		}
		for _, code := range r.NotificationTypeCodes {
			if !slices.Contains(managedOrderNotificationTypeCodes, code) {
				return nil, tracing.Trace(span, apierror.NewValidationError(fmt.Sprintf("Notification type %q is not supported for customer order notifications.", code)))
			}
		}
		if _, apiErr := s.repos.NewAccountUserRepo().GetDetailByAccountAndID(ctx, params.CustomerAccountID, r.AccountUserID, nil); apiErr != nil {
			if apiErr.Code == apierror.ErrorCodeResourceNotFound {
				return nil, tracing.Trace(span, apierror.NewValidationError("Notification recipient account user not found on this account."))
			}
			return nil, tracing.Trace(span, apiErr)
		}
	}

	relationID, apiErr := s.repos.NewAccountRelationRepo().FindRelationByOwnerAndCounterparty(ctx, identity.Target.AccountID, params.CustomerAccountID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *customerSvcImpl) *apierror.APIError {
		relationRepo := txSvc.repos.NewAccountRelationRepo()
		// Replace the managed set: drop existing invoice/acknowledgement prefs for the relation, then insert the requested set. PO-submission prefs are left untouched.
		if apiErr := relationRepo.DeleteNotificationPreferencesByTypes(txCtx, relationID, managedOrderNotificationTypeCodes); apiErr != nil {
			return apiErr
		}
		seen := make(map[string]struct{})
		for _, r := range params.Recipients {
			for _, code := range r.NotificationTypeCodes {
				key := r.AccountUserID + "\x00" + code
				if _, ok := seen[key]; ok {
					continue
				}
				seen[key] = struct{}{}
				prefID, apiErr := id.GenID(id.AccountRelationNotificationPreferenceIDPrefix, nil)
				if apiErr != nil {
					return apiErr
				}
				if apiErr := relationRepo.CreateNotificationPreference(txCtx, prefID, relationID, r.AccountUserID, code); apiErr != nil {
					return apiErr
				}
			}
		}
		return nil
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

// MergeCustomers merges source customers into a target customer.
func (s *customerSvcImpl) MergeCustomers(ctx context.Context, params domain.MergeCustomersParams) (*domain.Customer, *apierror.APIError) {
	ctx, span := customerSvcTracer.Start(ctx, "service.customer.merge")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
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

	params.OwnerAccountID = identity.Target.AccountID

	// Validate target is not in source list.
	if slices.Contains(params.SourceCustomerIDs, params.TargetCustomerID) {
		return nil, tracing.Trace(span, apierror.NewValidationError("Target customer cannot be in the source customer list."))
	}

	// Reject duplicate source IDs.
	seen := make(map[string]bool, len(params.SourceCustomerIDs))
	for _, sourceID := range params.SourceCustomerIDs {
		if seen[sourceID] {
			return nil, tracing.Trace(span, apierror.NewValidationError("Duplicate source customer IDs are not allowed."))
		}
		seen[sourceID] = true
	}

	// Verify all customers exist by fetching the target and sources.
	customerRepo := s.repos.NewCustomerRepo()
	targetOld, apiErr := customerRepo.Get(ctx, params.OwnerAccountID, params.TargetCustomerID, []string{"price_groups", "notification_preferences"})
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	sourceCustomers := make([]*domain.Customer, 0, len(params.SourceCustomerIDs))
	for _, sourceID := range params.SourceCustomerIDs {
		source, apiErr := customerRepo.Get(ctx, params.OwnerAccountID, sourceID, []string{"price_groups", "notification_preferences"})
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		sourceCustomers = append(sourceCustomers, source)
	}

	// Get target relation ID.
	targetRelationID, apiErr := customerRepo.GetRelationID(ctx, params.OwnerAccountID, params.TargetCustomerID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	// Get source relation IDs for sub-record cleanup.
	sourceRelationIDs := make([]string, 0, len(params.SourceCustomerIDs))
	for _, sourceID := range params.SourceCustomerIDs {
		relID, apiErr := customerRepo.GetRelationID(ctx, params.OwnerAccountID, sourceID)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		sourceRelationIDs = append(sourceRelationIDs, relID)
	}

	meds := s.mediators()

	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, identity)
	if apiErr != nil {
		return nil, apiErr
	}

	switch domain.RecoveryPoint(idempotencyKey.RecoveryPoint) {
	case domain.RecoveryPointFinished:
		cached, err := idempotency.UnmarshalCachedResponse[domain.Customer](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.Customer
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *customerSvcImpl) *apierror.APIError {
			txCustomerRepo := txSvc.repos.NewCustomerRepo()

			// Phase 1: Reassign all foreign key references from sources to target.
			if apiErr := txCustomerRepo.MergeOrders(txCtx, params.OwnerAccountID, params.TargetCustomerID, params.SourceCustomerIDs); apiErr != nil {
				return apiErr
			}
			if apiErr := txCustomerRepo.MergeInvoices(txCtx, params.OwnerAccountID, params.TargetCustomerID, params.SourceCustomerIDs); apiErr != nil {
				return apiErr
			}
			if apiErr := txCustomerRepo.MergeShipments(txCtx, params.OwnerAccountID, params.TargetCustomerID, params.SourceCustomerIDs); apiErr != nil {
				return apiErr
			}
			if apiErr := txCustomerRepo.MergeDeliveries(txCtx, params.OwnerAccountID, params.TargetCustomerID, params.SourceCustomerIDs); apiErr != nil {
				return apiErr
			}
			if apiErr := txCustomerRepo.MergeTransactions(txCtx, params.OwnerAccountID, params.TargetCustomerID, params.SourceCustomerIDs); apiErr != nil {
				return apiErr
			}
			if apiErr := txCustomerRepo.MergeAccountPrices(txCtx, params.OwnerAccountID, params.TargetCustomerID, params.SourceCustomerIDs); apiErr != nil {
				return apiErr
			}
			if apiErr := txCustomerRepo.MergeInventoryReceipts(txCtx, params.OwnerAccountID, params.TargetCustomerID, params.SourceCustomerIDs); apiErr != nil {
				return apiErr
			}
			if apiErr := txCustomerRepo.MergeReceivingOrders(txCtx, params.OwnerAccountID, params.TargetCustomerID, params.SourceCustomerIDs); apiErr != nil {
				return apiErr
			}
			if apiErr := txCustomerRepo.MergeInventoryIssues(txCtx, params.TargetCustomerID, params.SourceCustomerIDs); apiErr != nil {
				return apiErr
			}

			// Phase 2: Consolidate price groups — move non-duplicates from source to target, delete rest.
			targetPriceGroupIDs, apiErr := txCustomerRepo.GetRelationPriceGroupIDs(txCtx, targetRelationID)
			if apiErr != nil {
				return apiErr
			}
			targetPGSet := make(map[string]bool, len(targetPriceGroupIDs))
			for _, gid := range targetPriceGroupIDs {
				targetPGSet[gid] = true
			}
			sourcePriceGroups, apiErr := txCustomerRepo.GetRelationsPriceGroups(txCtx, sourceRelationIDs)
			if apiErr != nil {
				return apiErr
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
				if apiErr := txCustomerRepo.MoveRelationPriceGroups(txCtx, targetRelationID, pgToMove); apiErr != nil {
					return apiErr
				}
			}
			if len(pgToDelete) > 0 {
				if apiErr := txCustomerRepo.DeletePriceGroupsByIDs(txCtx, pgToDelete); apiErr != nil {
					return apiErr
				}
			}

			// Phase 3: Consolidate product lines — move non-duplicates from source to target, delete rest.
			targetProductLineIDs, apiErr := txCustomerRepo.GetRelationProductLineIDs(txCtx, targetRelationID)
			if apiErr != nil {
				return apiErr
			}
			targetPLSet := make(map[string]bool, len(targetProductLineIDs))
			for _, plID := range targetProductLineIDs {
				targetPLSet[plID] = true
			}
			sourceProductLines, apiErr := txCustomerRepo.GetRelationsProductLines(txCtx, sourceRelationIDs)
			if apiErr != nil {
				return apiErr
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
				if apiErr := txCustomerRepo.MoveRelationProductLines(txCtx, targetRelationID, plToMove); apiErr != nil {
					return apiErr
				}
			}
			if len(plToDelete) > 0 {
				if apiErr := txCustomerRepo.DeleteProductLinesByIDs(txCtx, plToDelete); apiErr != nil {
					return apiErr
				}
			}

			// Phase 4: Delete source notification preferences (user-bound, can't transfer).
			if apiErr := txCustomerRepo.DeleteNotificationPreferences(txCtx, sourceRelationIDs); apiErr != nil {
				return apiErr
			}

			// Phase 5: Re-parent child account relations.
			if apiErr := txCustomerRepo.ReparentChildRelations(txCtx, targetRelationID, sourceRelationIDs); apiErr != nil {
				return apiErr
			}

			// Phase 6: Consolidate account addresses — create missing on target, delete source.
			targetAddressIDs, apiErr := txCustomerRepo.GetAccountAddressIDs(txCtx, params.TargetCustomerID)
			if apiErr != nil {
				return apiErr
			}
			targetAddrSet := make(map[string]bool, len(targetAddressIDs))
			for _, addrID := range targetAddressIDs {
				targetAddrSet[addrID] = true
			}
			for _, sourceID := range params.SourceCustomerIDs {
				sourceAddrIDs, apiErr := txCustomerRepo.GetAccountAddressIDs(txCtx, sourceID)
				if apiErr != nil {
					return apiErr
				}
				for _, addrID := range sourceAddrIDs {
					if !targetAddrSet[addrID] {
						newID, apiErr := id.GenID(id.AccountAddressIDPrefix, nil)
						if apiErr != nil {
							return apiErr
						}
						if apiErr := txCustomerRepo.InsertAccountAddress(txCtx, newID, params.TargetCustomerID, addrID); apiErr != nil {
							return apiErr
						}
						targetAddrSet[addrID] = true
					}
				}
				if apiErr := txCustomerRepo.DeleteAccountAddresses(txCtx, sourceID); apiErr != nil {
					return apiErr
				}
			}

			// Phase 7: Consolidate account users — reassign non-duplicates, delete dupes.
			targetUsers, apiErr := txCustomerRepo.GetAccountUsers(txCtx, params.TargetCustomerID)
			if apiErr != nil {
				return apiErr
			}
			targetUserSet := make(map[string]bool, len(targetUsers))
			for _, u := range targetUsers {
				targetUserSet[u.UserID] = true
			}
			for _, sourceID := range params.SourceCustomerIDs {
				sourceUsers, apiErr := txCustomerRepo.GetAccountUsers(txCtx, sourceID)
				if apiErr != nil {
					return apiErr
				}
				var usersToMove []string
				for _, u := range sourceUsers {
					if !targetUserSet[u.UserID] {
						usersToMove = append(usersToMove, u.ID)
						targetUserSet[u.UserID] = true
					}
				}
				if len(usersToMove) > 0 {
					if apiErr := txCustomerRepo.MoveAccountUsers(txCtx, params.TargetCustomerID, usersToMove); apiErr != nil {
						return apiErr
					}
				}
				if apiErr := txCustomerRepo.DeleteAccountUsers(txCtx, sourceID); apiErr != nil {
					return apiErr
				}
			}

			// Phase 8: Delete source customer relations.
			if apiErr := txCustomerRepo.BulkDelete(txCtx, params.OwnerAccountID, params.SourceCustomerIDs); apiErr != nil {
				return apiErr
			}

			// Phase 9: Fetch the merged target customer for audit and response.
			result, apiErr = txCustomerRepo.Get(txCtx, params.OwnerAccountID, params.TargetCustomerID, customerAuditIncludes(params.Includes))
			if apiErr != nil {
				return apiErr
			}

			// Publish audit events: delete for each source, update for target.
			publisher := audit.NewPublisher()
			outboxRepo := txSvc.repos.NewOutboxRepo()

			for _, source := range sourceCustomers {
				changes := audit.ComputeChanges(source, (*domain.Customer)(nil))

				if apiErr := publisher.Publish(txCtx, outboxRepo, audit.EventData{
					ServiceName:  domain.ServiceName,
					Action:       constants.AuditActionDelete,
					ResourceType: constants.ObjectTypeCustomer,
					ResourceID:   source.ID,
					Changes:      changes,
				}); apiErr != nil {
					return apiErr
				}
			}

			changes := audit.ComputeChanges(targetOld, result)

			if apiErr := publisher.Publish(txCtx, outboxRepo, audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionUpdate,
				ResourceType: constants.ObjectTypeCustomer,
				ResourceID:   result.ID,
				Changes:      changes,
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
	if identity, ok := appctx.GetIdentityFromContext(ctx); ok {
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
