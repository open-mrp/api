package service

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/event"
	"github.com/open-mrp/api/shared/audit"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/id"
	"github.com/open-mrp/api/shared/idemresponse"
	"github.com/open-mrp/api/shared/messaging"
	apierror "github.com/open-mrp/apikit/apierror"
	"github.com/open-mrp/apikit/appctx"
	"github.com/open-mrp/apikit/tracing"
	"go.opentelemetry.io/otel/trace"
)

var registrationFlowSvcTracer = tracing.GetTracer("core-service.registration_flow_service")

type registrationFlowSvcImpl struct {
	repos           domain.RepoFactory
	mediatorFactory domain.MediatorFactory
	txManager       TransactionManager
}

type RegistrationFlowSvcConfig struct {
	// Repos (required) is the repository factory.
	Repos domain.RepoFactory

	// MediatorFactory (required) builds the mediators used by this service.
	MediatorFactory domain.MediatorFactory

	// TxManager (required) wraps multi-step operations in database transactions.
	TxManager TransactionManager
}

func (c *RegistrationFlowSvcConfig) validate() error {
	if c.Repos == nil {
		return fmt.Errorf("registration flow service: repos is required")
	}
	if c.MediatorFactory == nil {
		return fmt.Errorf("registration flow service: mediator factory is required")
	}
	if c.TxManager == nil {
		return fmt.Errorf("registration flow service: tx manager is required")
	}
	return nil
}

func NewRegistrationFlowSvc(config *RegistrationFlowSvcConfig) domain.RegistrationFlowSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &registrationFlowSvcImpl{
		repos:           config.Repos,
		mediatorFactory: config.MediatorFactory,
		txManager:       config.TxManager,
	}
}

func (s *registrationFlowSvcImpl) mediators() domain.Mediators {
	return s.mediatorFactory.Build(s.repos)
}

func (s *registrationFlowSvcImpl) withTx(ctx context.Context, fn func(context.Context, *registrationFlowSvcImpl) *apierror.APIError) *apierror.APIError {
	if s.txManager == nil {
		panic("txManager is nil")
	}

	return s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		txSvc := &registrationFlowSvcImpl{
			repos:           f,
			mediatorFactory: s.mediatorFactory,
			txManager:       s.txManager,
		}
		return fn(txCtx, txSvc)
	})
}

func (s *registrationFlowSvcImpl) ListRegistrationFlows(ctx context.Context, params domain.ListRegistrationFlowsParams) (*domain.ListRegistrationFlowsResult, *apierror.APIError) {
	ctx, span := registrationFlowSvcTracer.Start(ctx, "service.registration_flow.list")
	defer span.End()

	identity, ok := appctx.Identity[*types.Identity](ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainAccount, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	params.AccountID = identity.Target.AccountID

	return s.repos.NewRegistrationFlowRepo().List(ctx, params)
}

func (s *registrationFlowSvcImpl) GetRegistrationFlow(ctx context.Context, registrationFlowID string) (*domain.RegistrationFlow, *apierror.APIError) {
	ctx, span := registrationFlowSvcTracer.Start(ctx, "service.registration_flow.get")
	defer span.End()

	identity, ok := appctx.Identity[*types.Identity](ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainAccount, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return s.repos.NewRegistrationFlowRepo().Get(ctx, identity.Target.AccountID, registrationFlowID)
}

func (s *registrationFlowSvcImpl) CreateRegistrationFlow(ctx context.Context, params domain.CreateRegistrationFlowParams) (*domain.RegistrationFlow, *apierror.APIError) {
	ctx, span := registrationFlowSvcTracer.Start(ctx, "service.registration_flow.create")
	defer span.End()

	identity, ok := appctx.Identity[*types.Identity](ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainAccount, types.ActionUpdate); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	flowID, apiErr := id.GenID(id.RegistrationFlowIDPrefix, nil)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	params.AccountID = identity.Target.AccountID

	if apiErr := checkCustomerRefs(ctx, s.repos, params.AccountID, registrationFlowTermRefs(params.PaymentTermIDs, params.ShippingTermIDs)); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	meds := s.mediators()

	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, identity)
	if apiErr != nil {
		return nil, apiErr
	}

	switch domain.RecoveryPoint(idempotencyKey.RecoveryPoint) {
	case domain.RecoveryPointFinished:
		cached, err := idemresponse.UnmarshalCachedResponse[domain.RegistrationFlow](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.RegistrationFlow
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *registrationFlowSvcImpl) *apierror.APIError {
			txRepo := txSvc.repos.NewRegistrationFlowRepo()

			created, apiErr := txRepo.Create(txCtx, flowID, params)
			if apiErr != nil {
				return apiErr
			}
			result = created

			changes := audit.ComputeChanges(nil, created)

			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionCreate,
				ResourceType: constants.ObjectTypeRegistrationFlow,
				ResourceID:   created.ID,
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

func (s *registrationFlowSvcImpl) UpdateRegistrationFlow(ctx context.Context, params domain.UpdateRegistrationFlowParams) (*domain.RegistrationFlow, *apierror.APIError) {
	ctx, span := registrationFlowSvcTracer.Start(ctx, "service.registration_flow.update")
	defer span.End()

	identity, ok := appctx.Identity[*types.Identity](ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainAccount, types.ActionUpdate); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	params.AccountID = identity.Target.AccountID

	if apiErr := checkCustomerRefs(ctx, s.repos, params.AccountID, registrationFlowTermRefs(params.PaymentTermIDs, params.ShippingTermIDs)); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	meds := s.mediators()

	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, identity)
	if apiErr != nil {
		return nil, apiErr
	}

	switch domain.RecoveryPoint(idempotencyKey.RecoveryPoint) {
	case domain.RecoveryPointFinished:
		cached, err := idemresponse.UnmarshalCachedResponse[domain.RegistrationFlow](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.RegistrationFlow
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *registrationFlowSvcImpl) *apierror.APIError {
			txRepo := txSvc.repos.NewRegistrationFlowRepo()

			old, apiErr := txRepo.Get(txCtx, params.AccountID, params.RegistrationFlowID)
			if apiErr != nil {
				return apiErr
			}

			updated, apiErr := txRepo.Update(txCtx, params)
			if apiErr != nil {
				return apiErr
			}
			result = updated

			changes := audit.ComputeChanges(old, updated)

			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionUpdate,
				ResourceType: constants.ObjectTypeRegistrationFlow,
				ResourceID:   updated.ID,
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

func (s *registrationFlowSvcImpl) DeleteRegistrationFlow(ctx context.Context, registrationFlowID string) *apierror.APIError {
	ctx, span := registrationFlowSvcTracer.Start(ctx, "service.registration_flow.delete")
	defer span.End()

	identity, ok := appctx.Identity[*types.Identity](ctx)
	if !ok || identity == nil {
		return tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainAccount, types.ActionUpdate); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	accountID := identity.Target.AccountID

	registrationFlow, apiErr := s.repos.NewRegistrationFlowRepo().Get(ctx, accountID, registrationFlowID)
	if apiErr != nil {
		if apierror.HasCode(apiErr, apierror.CodeResourceNotFound) {
			wasDeleted, deletedCheckErr := s.repos.NewDeletedRecordRepo().ExistsInAccount(ctx, constants.DeletedRecordResourceTypeRegistrationFlow, registrationFlowID, accountID)
			if deletedCheckErr != nil {
				return tracing.Trace(span, deletedCheckErr)
			}
			if wasDeleted {
				return tracing.Trace(span, apierror.NewGoneError("This registration flow has already been deleted and can no longer be modified."))
			}
		}
		return tracing.Trace(span, apiErr)
	}

	apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *registrationFlowSvcImpl) *apierror.APIError {
		if apiErr := txSvc.repos.NewDeletedRecordRepo().CreateInAccount(txCtx, constants.DeletedRecordResourceTypeRegistrationFlow, registrationFlow.ID, accountID, registrationFlow); apiErr != nil {
			return apiErr
		}

		if apiErr := txSvc.repos.NewRegistrationFlowRepo().Delete(txCtx, domain.DeleteRegistrationFlowParams{
			AccountID:          accountID,
			RegistrationFlowID: registrationFlowID,
		}); apiErr != nil {
			return apiErr
		}

		changes := audit.ComputeChanges(registrationFlow, (*domain.RegistrationFlow)(nil))

		if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
			ServiceName:  domain.ServiceName,
			Action:       constants.AuditActionDelete,
			ResourceType: constants.ObjectTypeRegistrationFlow,
			ResourceID:   registrationFlow.ID,
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

func (s *registrationFlowSvcImpl) GetRegistrationFlowBySlug(ctx context.Context, slug string) (*domain.RegistrationFlow, *apierror.APIError) {
	ctx, span := registrationFlowSvcTracer.Start(ctx, "service.registration_flow.get_by_slug")
	defer span.End()

	identity, ok := appctx.Identity[*types.Identity](ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsAuthenticated(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	// Resolve account from slug
	account, apiErr := s.repos.NewAccountRepo().GetBySlug(ctx, slug)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	flows, apiErr := s.repos.NewRegistrationFlowRepo().GetByAccountID(ctx, account.ID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if len(flows) == 0 {
		return nil, tracing.Trace(span, apierror.NewNotFoundError("Registration flow not found."))
	}

	return flows[0], nil
}

func (s *registrationFlowSvcImpl) RegisterCustomer(ctx context.Context, params domain.RegisterCustomerParams) *apierror.APIError {
	ctx, span := registrationFlowSvcTracer.Start(ctx, "service.registration_flow.register_customer")
	defer span.End()

	identity, ok := appctx.Identity[*types.Identity](ctx)
	if !ok || identity == nil {
		return tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	// Registration links the actor to a customer account as a user, which an API key or agent is not.
	if apiErr := identity.CheckHasUserActor(); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	userID := identity.Actor.ID

	// Resolve account from slug
	account, apiErr := s.repos.NewAccountRepo().GetBySlug(ctx, params.AccountSlug)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	meds := s.mediators()

	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, identity)
	if apiErr != nil {
		return apiErr
	}

	switch domain.RecoveryPoint(idempotencyKey.RecoveryPoint) {
	case domain.RecoveryPointFinished:
		_, err := idemresponse.UnmarshalCachedResponse[any](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return nil

	case domain.RecoveryPointStarted:
		if params.IsExistingCustomer {
			return s.registerExistingCustomer(ctx, span, account.ID, userID, params, meds, idempotencyKey.TypeID)
		}
		return s.registerNewCustomer(ctx, span, account.ID, userID, params, meds, idempotencyKey.TypeID)

	default:
		return tracing.Trace(span, apierror.NewInvariantViolationError("Unexpected recovery point: "+idempotencyKey.RecoveryPoint))
	}
}

func (s *registrationFlowSvcImpl) registerExistingCustomer(
	ctx context.Context,
	span trace.Span,
	ownerAccountID, userID string,
	params domain.RegisterCustomerParams,
	meds domain.Mediators,
	idempotencyKeyTypeID string,
) *apierror.APIError {
	customerAccountID, apiErr := findExistingCustomer(ctx, s.repos.NewCustomerRegistrationRepo(), ownerAccountID, params)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	customer, apiErr := s.repos.NewCustomerRepo().Get(ctx, ownerAccountID, customerAccountID, nil)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *registrationFlowSvcImpl) *apierror.APIError {
		txRepo := txSvc.repos.NewCustomerRegistrationRepo()

		accountUserID, apiErr := id.GenID(id.AccountUserIDPrefix, nil)
		if apiErr != nil {
			return apiErr
		}

		if apiErr := txRepo.CreateAccountUserLink(txCtx, accountUserID, customerAccountID, userID); apiErr != nil {
			return apiErr
		}

		if apiErr := publishRegistrationAudit(txCtx, txSvc.repos, ownerAccountID, customerAccountID, accountUserID, false); apiErr != nil {
			return apiErr
		}

		// Notify the seller's customer-service group that a buyer joined an existing customer account. Emitted in-tx so it commits atomically with the registration.
		if apiErr := event.NewCustomerRegisteredPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), messaging.CustomerRegisteredData{
			SellerAccountID:    ownerAccountID,
			CustomerAccountID:  customerAccountID,
			CustomerName:       customer.Name,
			CustomerNumber:     customer.Number,
			RegistrantUserID:   userID,
			IsExistingCustomer: true,
		}); apiErr != nil {
			return apiErr
		}

		return txSvc.mediators().Idempotency.CacheSuccessResponse(txCtx, idempotencyKeyTypeID, nil)
	})

	if apiErr != nil {
		return meds.Idempotency.CacheErrorResponse(ctx, idempotencyKeyTypeID, apiErr)
	}

	return nil
}

// registrationFlowTermRefs lists the terms a flow would offer its registrants, who then choose one for their customer record.
func registrationFlowTermRefs(paymentTermIDs, shippingTermIDs []string) customerRefs {
	var refs customerRefs
	for _, termID := range paymentTermIDs {
		refs.add(customerRefPaymentTerm, &termID, nil, "payment_term_ids")
	}
	for _, termID := range shippingTermIDs {
		refs.add(customerRefShippingTerm, &termID, nil, "shipping_term_ids")
	}
	return refs
}

func (s *registrationFlowSvcImpl) registerNewCustomer(
	ctx context.Context,
	span trace.Span,
	ownerAccountID, userID string,
	params domain.RegisterCustomerParams,
	meds domain.Mediators,
	idempotencyKeyTypeID string,
) *apierror.APIError {
	data := params.CustomerData

	// Validate required fields for new customer
	if data.Name == nil || *data.Name == "" {
		return tracing.Trace(span, apierror.NewValidationError("Customer name is required."))
	}
	if data.Address == nil {
		return tracing.Trace(span, apierror.NewValidationError("Address is required."))
	}
	if data.ShippingTermID == nil || *data.ShippingTermID == "" {
		return tracing.Trace(span, apierror.NewValidationError("Shipping term is required."))
	}
	if data.PaymentTermID == nil || *data.PaymentTermID == "" {
		return tracing.Trace(span, apierror.NewValidationError("Payment term is required."))
	}
	if data.CustomerGroupID == nil || *data.CustomerGroupID == "" {
		return tracing.Trace(span, apierror.NewValidationError("Customer group is required."))
	}

	// The registrant chooses these, and they are stored on the seller's customer, so each must be the seller's.
	if apiErr := checkCustomerRefs(ctx, s.repos, ownerAccountID, customerRefs{
		{kind: customerRefAccountGroup, id: *data.CustomerGroupID, param: "customer_group_id"},
		{kind: customerRefPaymentTerm, id: *data.PaymentTermID, param: "payment_term_id"},
		{kind: customerRefShippingTerm, id: *data.ShippingTermID, param: "shipping_term_id"},
	}); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	customerRepo := s.repos.NewCustomerRegistrationRepo()

	// Get user email for branding
	email, apiErr := customerRepo.GetUserEmailByID(ctx, userID)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	possibleDuplicates := findPossibleDuplicateCustomers(ctx, customerRepo, ownerAccountID, email, data.Address)

	sysPropertyID, apiErr := id.GenID(id.SysPropertyIDPrefix, nil)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	var customerNumber string

	apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *registrationFlowSvcImpl) *apierror.APIError {
		txRepo := txSvc.repos.NewCustomerRegistrationRepo()

		// Queue behind the seller's other customer number writers, so a number someone is typing in
		// right now is not handed out here as well.
		if apiErr := txSvc.repos.NewCustomerRepo().LockNumbers(txCtx, ownerAccountID); apiErr != nil {
			return apiErr
		}

		// Reserve the number inside the transaction. Two people completing registration at
		// the same moment used to read the same counter and both be given that number.
		nextNum, apiErr := txRepo.AllocateNextCustomerNumber(txCtx, sysPropertyID, ownerAccountID)
		if apiErr != nil {
			return apiErr
		}
		customerNumber = strconv.FormatInt(nextNum, 10)

		// Create the customer account with all associated entities
		addressParams := domain.CustomerRegistrationAddressParams{
			Name:        data.Address.Name,
			StreetLine1: data.Address.StreetLine1,
			StreetLine2: data.Address.StreetLine2,
			Locality:    data.Address.Locality,
			State:       data.Address.State,
			PostalCode:  data.Address.PostalCode,
			Country:     data.Address.Country,
		}

		customerAccountID, apiErr := txRepo.CreateNewCustomerAccount(txCtx, domain.CreateNewCustomerAccountParams{
			AccountID:       ownerAccountID,
			CustomerName:    strings.TrimSpace(*data.Name),
			Email:           email,
			Phone:           data.Phone,
			Address:         addressParams,
			ShippingTermID:  *data.ShippingTermID,
			PaymentTermID:   *data.PaymentTermID,
			CustomerGroupID: *data.CustomerGroupID,
			CustomerNumber:  customerNumber,
		})
		if apiErr != nil {
			return apiErr
		}

		// Link the user to the new customer account
		accountUserID, apiErr := id.GenID(id.AccountUserIDPrefix, nil)
		if apiErr != nil {
			return apiErr
		}

		if apiErr := txRepo.CreateAccountUserLink(txCtx, accountUserID, customerAccountID, userID); apiErr != nil {
			return apiErr
		}

		if apiErr := publishRegistrationAudit(txCtx, txSvc.repos, ownerAccountID, customerAccountID, accountUserID, true); apiErr != nil {
			return apiErr
		}

		// Notify the seller's customer-service group that a new customer registered on the portal. Emitted in-tx so it commits atomically with the registration.
		if apiErr := event.NewCustomerRegisteredPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), messaging.CustomerRegisteredData{
			SellerAccountID:    ownerAccountID,
			CustomerAccountID:  customerAccountID,
			CustomerName:       strings.TrimSpace(*data.Name),
			CustomerNumber:     customerNumber,
			RegistrantUserID:   userID,
			PossibleDuplicates: possibleDuplicates,
		}); apiErr != nil {
			return apiErr
		}

		return txSvc.mediators().Idempotency.CacheSuccessResponse(txCtx, idempotencyKeyTypeID, nil)
	})

	if apiErr != nil {
		return meds.Idempotency.CacheErrorResponse(ctx, idempotencyKeyTypeID, apiErr)
	}

	return nil
}

// publishRegistrationAudit records a self-registration in the seller's audit log as the registrant's doing:
// the customer when one was created, and the registrant's account user under it.
func publishRegistrationAudit(txCtx context.Context, repos domain.RepoFactory, ownerAccountID, customerAccountID, accountUserID string, customerCreated bool) *apierror.APIError {
	identity, ok := appctx.Identity[*types.Identity](txCtx)
	if !ok || identity == nil {
		return apierror.NewInvariantViolationError("Identity not found in context.")
	}
	// The registrant targets no account of their own yet. This copy only scopes the events to the seller; nothing authorizes against it.
	scoped := *identity
	scoped.Target = &types.IdentityTarget{AccountID: ownerAccountID}
	auditCtx := appctx.WithIdentity(txCtx, &scoped)
	publisher := audit.NewPublisher()

	if customerCreated {
		customer, apiErr := repos.NewCustomerRepo().Get(txCtx, ownerAccountID, customerAccountID, customerAuditIncludes(nil))
		if apiErr != nil {
			return apiErr
		}
		if apiErr := publisher.Publish(auditCtx, repos.NewOutboxRepo(), audit.EventData{
			ServiceName:  domain.ServiceName,
			Action:       constants.AuditActionCreate,
			ResourceType: constants.ObjectTypeCustomer,
			ResourceID:   customer.ID,
			Changes:      audit.ComputeChanges(nil, customer),
			Metadata:     map[string]any{"source": "registration"},
		}); apiErr != nil {
			return apiErr
		}
	}

	accountUser, apiErr := repos.NewAccountUserRepo().GetDetailByAccountAndID(txCtx, customerAccountID, accountUserID, nil)
	if apiErr != nil {
		return apiErr
	}
	return publisher.Publish(auditCtx, repos.NewOutboxRepo(), audit.EventData{
		ServiceName:      domain.ServiceName,
		Action:           constants.AuditActionCreate,
		ResourceType:     constants.ObjectTypeAccountUser,
		ResourceID:       accountUser.ID,
		Changes:          audit.ComputeChanges(nil, accountUser),
		RootResourceType: constants.ObjectTypeCustomer,
		RootResourceID:   customerAccountID,
		Metadata:         map[string]any{"source": "registration"},
	})
}
