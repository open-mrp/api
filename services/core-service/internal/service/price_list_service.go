package service

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/shopspring/decimal"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/audit"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/id"
	"github.com/open-mrp/api/shared/idemresponse"
	apierror "github.com/open-mrp/apikit/apierror"
	"github.com/open-mrp/apikit/appctx"
	"github.com/open-mrp/apikit/tracing"
)

var priceListSvcTracer = tracing.GetTracer("core-service.price_list_service")

type priceListSvcImpl struct {
	repos           domain.RepoFactory
	mediatorFactory domain.MediatorFactory
	txManager       TransactionManager
}

type PriceListSvcConfig struct {
	// Repos (required) is the repository factory.
	Repos domain.RepoFactory

	// MediatorFactory (required) builds the mediators used by this service.
	MediatorFactory domain.MediatorFactory

	// TxManager (required) wraps multi-step operations in database transactions.
	TxManager TransactionManager
}

func (c *PriceListSvcConfig) validate() error {
	if c.Repos == nil {
		return fmt.Errorf("price list service: repos is required")
	}
	if c.MediatorFactory == nil {
		return fmt.Errorf("price list service: mediator factory is required")
	}
	if c.TxManager == nil {
		return fmt.Errorf("price list service: tx manager is required")
	}
	return nil
}

func NewPriceListSvc(config *PriceListSvcConfig) domain.PriceListSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}
	return &priceListSvcImpl{repos: config.Repos, mediatorFactory: config.MediatorFactory, txManager: config.TxManager}
}

func (s *priceListSvcImpl) mediators() domain.Mediators {
	return s.mediatorFactory.Build(s.repos)
}

func (s *priceListSvcImpl) withTx(ctx context.Context, fn func(context.Context, *priceListSvcImpl) *apierror.APIError) *apierror.APIError {
	return s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		return fn(txCtx, &priceListSvcImpl{repos: f, mediatorFactory: s.mediatorFactory, txManager: s.txManager})
	})
}

// priceListIdentity is the caller, a member of the account the lists belong to; each method checks its discounts
// permission. Portal users never see price lists.
func priceListIdentity(ctx context.Context, read bool) (*types.Identity, *apierror.APIError) {
	identity, ok := appctx.Identity[*types.Identity](ctx)
	if !ok || identity == nil {
		return nil, apierror.NewInvariantViolationError("Identity not found in context.")
	}
	check := identity.CheckIsInternalActor
	if read {
		check = identity.CheckIsInternalActorForRead
	}
	if apiErr := check(); apiErr != nil {
		return nil, apiErr
	}
	return identity, nil
}

// priceListIdempotentWrite runs fn once per idempotency key in a transaction and caches its result.
func priceListIdempotentWrite[T any](ctx context.Context, s *priceListSvcImpl, identity *types.Identity,
	fn func(context.Context, *priceListSvcImpl) (*T, *apierror.APIError)) (*T, *apierror.APIError) {
	meds := s.mediators()
	key, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, identity)
	if apiErr != nil {
		return nil, apiErr
	}
	switch domain.RecoveryPoint(key.RecoveryPoint) {
	case domain.RecoveryPointFinished:
		cached, err := idemresponse.UnmarshalCachedResponse[T](ctx, key.ResponseCode, key.ResponseBody)
		if err != nil {
			return nil, apierror.NewInternalError(err, "Issue unmarshalling cached response.")
		}
		return cached.Data, cached.Error
	case domain.RecoveryPointStarted:
		var result *T
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *priceListSvcImpl) *apierror.APIError {
			out, apiErr := fn(txCtx, txSvc)
			if apiErr != nil {
				return apiErr
			}
			result = out
			return txSvc.mediators().Idempotency.CacheSuccessResponse(txCtx, key.TypeID, result)
		})
		if apiErr != nil {
			return nil, meds.Idempotency.CacheErrorResponse(ctx, key.TypeID, apiErr)
		}
		return result, nil
	default:
		return nil, apierror.NewInvariantViolationError("Unexpected recovery point: " + key.RecoveryPoint)
	}
}

func (s *priceListSvcImpl) publish(ctx context.Context, action constants.AuditAction, objectType constants.ObjectType, resourceID string, changes []audit.FieldChange) *apierror.APIError {
	return audit.NewPublisher().Publish(ctx, s.repos.NewOutboxRepo(), audit.EventData{
		ServiceName:  domain.ServiceName,
		Action:       action,
		ResourceType: objectType,
		ResourceID:   resourceID,
		Changes:      changes,
	})
}

// notFoundOrGone is a missing record's error: 410 when it was deleted, else 404.
func (s *priceListSvcImpl) notFoundOrGone(ctx context.Context, kind constants.DeletedRecordResourceType, accountID, recordID, noun string) *apierror.APIError {
	gone, apiErr := s.repos.NewDeletedRecordRepo().ExistsInAccount(ctx, kind, recordID, accountID)
	if apiErr != nil {
		return apiErr
	}
	if gone {
		return apierror.NewGoneError(fmt.Sprintf("This %s has been deleted.", noun))
	}
	return apierror.NewNotFoundError(fmt.Sprintf("%s not found.", strings.ToUpper(noun[:1])+noun[1:]))
}

func (s *priceListSvcImpl) getList(ctx context.Context, accountID, listID string) (*domain.PriceList, *apierror.APIError) {
	lists, apiErr := s.repos.NewPriceListRepo().GetByIDs(ctx, accountID, []string{listID})
	if apiErr != nil {
		return nil, apiErr
	}
	if len(lists) == 0 {
		return nil, s.notFoundOrGone(ctx, constants.DeletedRecordResourceTypePriceList, accountID, listID, "price list")
	}
	return lists[0], nil
}

func (s *priceListSvcImpl) getRule(ctx context.Context, accountID, listID, ruleID string) (*domain.PriceListRule, *apierror.APIError) {
	rules, apiErr := s.repos.NewPriceListRepo().GetRulesByIDs(ctx, accountID, []string{ruleID})
	if apiErr != nil {
		return nil, apiErr
	}
	if len(rules) == 0 || rules[0].PriceListID != listID {
		return nil, s.notFoundOrGone(ctx, constants.DeletedRecordResourceTypePriceListRule, accountID, ruleID, "price list rule")
	}
	return rules[0], nil
}

// --- Price lists -------------------------------------------------------------

func (s *priceListSvcImpl) ListPriceLists(ctx context.Context, params domain.ListPriceListsParams) (*domain.ListPriceListsResult, *apierror.APIError) {
	ctx, span := priceListSvcTracer.Start(ctx, "service.price_list.list")
	defer span.End()

	identity, apiErr := priceListIdentity(ctx, true)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainDiscounts, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	params.AccountID = identity.Target.AccountID
	if len(params.CustomerIDs) > 0 {
		applicable, apiErr := s.repos.NewPriceListRepo().ApplicableLists(ctx, params.AccountID, params.CustomerIDs)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		ids := []string{}
		for _, lists := range applicable {
			for listID := range lists {
				if !slices.Contains(ids, listID) {
					ids = append(ids, listID)
				}
			}
		}
		params.IDs = ids
	}
	result, apiErr := s.repos.NewPriceListRepo().List(ctx, params)
	return result, tracing.Trace(span, apiErr)
}

func (s *priceListSvcImpl) GetPriceList(ctx context.Context, listID string) (*domain.PriceList, *apierror.APIError) {
	ctx, span := priceListSvcTracer.Start(ctx, "service.price_list.get")
	defer span.End()

	identity, apiErr := priceListIdentity(ctx, true)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainDiscounts, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	list, apiErr := s.getList(ctx, identity.Target.AccountID, listID)
	return list, tracing.Trace(span, apiErr)
}

func (s *priceListSvcImpl) BatchGetPriceLists(ctx context.Context, ids []string, withRules bool) ([]*domain.PriceList, map[string]domain.PriceListRuleIDs, *apierror.APIError) {
	ctx, span := priceListSvcTracer.Start(ctx, "service.price_list.batch_get")
	defer span.End()

	identity, apiErr := priceListIdentity(ctx, true)
	if apiErr != nil {
		return nil, nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainDiscounts, types.ActionRead); apiErr != nil {
		return nil, nil, tracing.Trace(span, apiErr)
	}
	repo := s.repos.NewPriceListRepo()
	lists, apiErr := repo.GetByIDs(ctx, identity.Target.AccountID, ids)
	if apiErr != nil || !withRules || len(lists) == 0 {
		return lists, nil, tracing.Trace(span, apiErr)
	}
	listIDs := make([]string, len(lists))
	for i, l := range lists {
		listIDs[i] = l.ID
	}
	rules, apiErr := repo.FirstRuleIDs(ctx, listIDs)
	return lists, rules, tracing.Trace(span, apiErr)
}

// CreatePriceList adds a list to the caller's account. The account's legacy prices are mirrored first, since its orders
// are priced from lists from then on.
func (s *priceListSvcImpl) CreatePriceList(ctx context.Context, params domain.CreatePriceListParams) (*domain.PriceList, *apierror.APIError) {
	ctx, span := priceListSvcTracer.Start(ctx, "service.price_list.create")
	defer span.End()

	identity, apiErr := priceListIdentity(ctx, false)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainDiscounts, types.ActionCreate); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	list := &domain.PriceList{
		AccountID:   identity.Target.AccountID,
		Name:        strings.TrimSpace(params.Name),
		Description: params.Description,
		Status:      constants.ArchiveStatusActive,
		Currency:    constants.CurrencyUSD,
	}
	if list.Name == "" {
		return nil, tracing.Trace(span, apierror.NewFieldError("name", apierror.CodeInvalidFormat, "A price list needs a name."))
	}
	if params.Status != nil {
		list.Status = *params.Status
	}
	if params.Currency != nil {
		currency, apiErr := domain.ValidateCurrency("currency", *params.Currency)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		list.Currency = currency
	}
	listID, apiErr := id.GenID(id.PriceListIDPrefix, nil)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	list.ID = listID

	result, apiErr := priceListIdempotentWrite(ctx, s, identity, func(txCtx context.Context, txSvc *priceListSvcImpl) (*domain.PriceList, *apierror.APIError) {
		if apiErr := ensurePriceListsSynced(txCtx, txSvc.repos, list.AccountID); apiErr != nil {
			return nil, apiErr
		}
		repo := txSvc.repos.NewPriceListRepo()
		if apiErr := repo.Insert(txCtx, list); apiErr != nil {
			return nil, apiErr
		}
		created, apiErr := txSvc.getList(txCtx, list.AccountID, list.ID)
		if apiErr != nil {
			return nil, apiErr
		}
		return created, txSvc.publish(txCtx, constants.AuditActionCreate, constants.ObjectTypePriceList, created.ID, audit.ComputeChanges(nil, created))
	})
	return result, tracing.Trace(span, apiErr)
}

func (s *priceListSvcImpl) UpdatePriceList(ctx context.Context, params domain.UpdatePriceListParams) (*domain.PriceList, *apierror.APIError) {
	ctx, span := priceListSvcTracer.Start(ctx, "service.price_list.update")
	defer span.End()

	identity, apiErr := priceListIdentity(ctx, false)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainDiscounts, types.ActionUpdate); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if params.Name != nil && strings.TrimSpace(*params.Name) == "" {
		return nil, tracing.Trace(span, apierror.NewFieldError("name", apierror.CodeInvalidFormat, "A price list needs a name."))
	}
	accountID := identity.Target.AccountID
	result, apiErr := priceListIdempotentWrite(ctx, s, identity, func(txCtx context.Context, txSvc *priceListSvcImpl) (*domain.PriceList, *apierror.APIError) {
		old, apiErr := txSvc.getList(txCtx, accountID, params.PriceListID)
		if apiErr != nil {
			return nil, apiErr
		}
		next := *old
		if params.Name != nil {
			next.Name = strings.TrimSpace(*params.Name)
		}
		switch {
		case params.ClearDescription:
			next.Description = nil
		case params.Description != nil:
			next.Description = params.Description
		}
		if params.Status != nil {
			next.Status = *params.Status
		}
		if apiErr := txSvc.repos.NewPriceListRepo().Update(txCtx, &next); apiErr != nil {
			return nil, apiErr
		}
		updated, apiErr := txSvc.getList(txCtx, accountID, params.PriceListID)
		if apiErr != nil {
			return nil, apiErr
		}
		return updated, txSvc.publish(txCtx, constants.AuditActionUpdate, constants.ObjectTypePriceList, updated.ID, audit.ComputeChanges(old, updated))
	})
	return result, tracing.Trace(span, apiErr)
}

// DeletePriceList removes a list and its rules while no customer or account group is assigned it.
func (s *priceListSvcImpl) DeletePriceList(ctx context.Context, listID string) (*domain.PriceList, *apierror.APIError) {
	ctx, span := priceListSvcTracer.Start(ctx, "service.price_list.delete")
	defer span.End()

	identity, apiErr := priceListIdentity(ctx, false)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainDiscounts, types.ActionDelete); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	accountID := identity.Target.AccountID
	var deleted *domain.PriceList
	apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *priceListSvcImpl) *apierror.APIError {
		list, apiErr := txSvc.getList(txCtx, accountID, listID)
		if apiErr != nil {
			return apiErr
		}
		if apiErr := domain.PriceListReferences.RequireUnused(txCtx, txSvc.repos.NewReferenceRepo(), listID); apiErr != nil {
			return apiErr
		}
		if apiErr := txSvc.repos.NewDeletedRecordRepo().CreateInAccount(txCtx, constants.DeletedRecordResourceTypePriceList, list.ID, accountID, list); apiErr != nil {
			return apiErr
		}
		if apiErr := txSvc.repos.NewPriceListRepo().Delete(txCtx, accountID, listID); apiErr != nil {
			return apiErr
		}
		deleted = list
		return txSvc.publish(txCtx, constants.AuditActionDelete, constants.ObjectTypePriceList, list.ID, audit.ComputeChanges(list, (*domain.PriceList)(nil)))
	})
	return deleted, tracing.Trace(span, apiErr)
}

// --- Rules -------------------------------------------------------------------

func (s *priceListSvcImpl) ListPriceListRules(ctx context.Context, params domain.ListPriceListRulesParams) (*domain.ListPriceListRulesResult, *apierror.APIError) {
	ctx, span := priceListSvcTracer.Start(ctx, "service.price_list.list_rules")
	defer span.End()

	identity, apiErr := priceListIdentity(ctx, true)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainDiscounts, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	params.AccountID = identity.Target.AccountID
	if _, apiErr := s.getList(ctx, params.AccountID, params.PriceListID); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	result, apiErr := s.repos.NewPriceListRepo().ListRules(ctx, params)
	return result, tracing.Trace(span, apiErr)
}

func (s *priceListSvcImpl) GetPriceListRule(ctx context.Context, listID, ruleID string) (*domain.PriceListRule, *apierror.APIError) {
	ctx, span := priceListSvcTracer.Start(ctx, "service.price_list.get_rule")
	defer span.End()

	identity, apiErr := priceListIdentity(ctx, true)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainDiscounts, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if _, apiErr := s.getList(ctx, identity.Target.AccountID, listID); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	rule, apiErr := s.getRule(ctx, identity.Target.AccountID, listID, ruleID)
	return rule, tracing.Trace(span, apiErr)
}

func (s *priceListSvcImpl) BatchGetPriceListRules(ctx context.Context, ids []string) ([]*domain.PriceListRule, *apierror.APIError) {
	ctx, span := priceListSvcTracer.Start(ctx, "service.price_list.batch_get_rules")
	defer span.End()

	identity, apiErr := priceListIdentity(ctx, true)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainDiscounts, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	rules, apiErr := s.repos.NewPriceListRepo().GetRulesByIDs(ctx, identity.Target.AccountID, ids)
	return rules, tracing.Trace(span, apiErr)
}

func (s *priceListSvcImpl) CreatePriceListRule(ctx context.Context, params domain.CreatePriceListRuleParams) (*domain.PriceListRule, *apierror.APIError) {
	ctx, span := priceListSvcTracer.Start(ctx, "service.price_list.create_rule")
	defer span.End()

	identity, apiErr := priceListIdentity(ctx, false)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainDiscounts, types.ActionCreate); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	accountID := identity.Target.AccountID
	price, apiErr := normalizeRulePrice(params.Rule.Price)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	scope := normalizeRuleScope(params.Rule.Scope)
	ruleID, apiErr := id.GenID(id.PriceListRuleIDPrefix, nil)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	result, apiErr := priceListIdempotentWrite(ctx, s, identity, func(txCtx context.Context, txSvc *priceListSvcImpl) (*domain.PriceListRule, *apierror.APIError) {
		if _, apiErr := txSvc.getList(txCtx, accountID, params.PriceListID); apiErr != nil {
			return nil, apiErr
		}
		if apiErr := txSvc.checkRuleRefs(txCtx, accountID, scope, price, nil); apiErr != nil {
			return nil, apiErr
		}
		repo := txSvc.repos.NewPriceListRepo()
		if apiErr := repo.InsertRule(txCtx, &domain.PriceListRule{
			ID: ruleID, PriceListID: params.PriceListID, AccountID: accountID, Scope: scope, Price: price,
		}); apiErr != nil {
			return nil, apiErr
		}
		created, apiErr := txSvc.getRule(txCtx, accountID, params.PriceListID, ruleID)
		if apiErr != nil {
			return nil, apiErr
		}
		return created, txSvc.publish(txCtx, constants.AuditActionCreate, constants.ObjectTypePriceListRule, created.ID, audit.ComputeChanges(nil, created))
	})
	return result, tracing.Trace(span, apiErr)
}

func (s *priceListSvcImpl) UpdatePriceListRule(ctx context.Context, params domain.UpdatePriceListRuleParams) (*domain.PriceListRule, *apierror.APIError) {
	ctx, span := priceListSvcTracer.Start(ctx, "service.price_list.update_rule")
	defer span.End()

	identity, apiErr := priceListIdentity(ctx, false)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainDiscounts, types.ActionUpdate); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	accountID := identity.Target.AccountID
	var price *domain.PriceListPrice
	if params.Price != nil {
		p, apiErr := normalizeRulePrice(*params.Price)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		price = &p
	}

	result, apiErr := priceListIdempotentWrite(ctx, s, identity, func(txCtx context.Context, txSvc *priceListSvcImpl) (*domain.PriceListRule, *apierror.APIError) {
		old, apiErr := txSvc.getRule(txCtx, accountID, params.PriceListID, params.RuleID)
		if apiErr != nil {
			return nil, apiErr
		}
		next := *old
		if params.ItemIDs != nil {
			next.Scope.ItemIDs = *params.ItemIDs
		}
		if params.ProductLineIDs != nil {
			next.Scope.ProductLineIDs = *params.ProductLineIDs
		}
		if params.CategoryIDs != nil {
			next.Scope.CategoryIDs = *params.CategoryIDs
		}
		if params.AttributeIDs != nil {
			next.Scope.AttributeIDs = *params.AttributeIDs
		}
		next.Scope = normalizeRuleScope(next.Scope)
		if price != nil {
			next.Price = *price
		}
		if apiErr := txSvc.checkRuleRefs(txCtx, accountID, next.Scope, next.Price, old); apiErr != nil {
			return nil, apiErr
		}
		if apiErr := txSvc.repos.NewPriceListRepo().UpdateRule(txCtx, &next); apiErr != nil {
			return nil, apiErr
		}
		updated, apiErr := txSvc.getRule(txCtx, accountID, params.PriceListID, params.RuleID)
		if apiErr != nil {
			return nil, apiErr
		}
		return updated, txSvc.publish(txCtx, constants.AuditActionUpdate, constants.ObjectTypePriceListRule, updated.ID, audit.ComputeChanges(old, updated))
	})
	return result, tracing.Trace(span, apiErr)
}

// DeletePriceListRule removes a rule. Orders already priced by it keep their prices.
func (s *priceListSvcImpl) DeletePriceListRule(ctx context.Context, listID, ruleID string) (*domain.PriceListRule, *apierror.APIError) {
	ctx, span := priceListSvcTracer.Start(ctx, "service.price_list.delete_rule")
	defer span.End()

	identity, apiErr := priceListIdentity(ctx, false)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainDiscounts, types.ActionDelete); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	accountID := identity.Target.AccountID
	var deleted *domain.PriceListRule
	apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *priceListSvcImpl) *apierror.APIError {
		if _, apiErr := txSvc.getList(txCtx, accountID, listID); apiErr != nil {
			return apiErr
		}
		rule, apiErr := txSvc.getRule(txCtx, accountID, listID, ruleID)
		if apiErr != nil {
			return apiErr
		}
		if apiErr := txSvc.repos.NewDeletedRecordRepo().CreateInAccount(txCtx, constants.DeletedRecordResourceTypePriceListRule, rule.ID, accountID, rule); apiErr != nil {
			return apiErr
		}
		if apiErr := txSvc.repos.NewPriceListRepo().DeleteRule(txCtx, ruleID); apiErr != nil {
			return apiErr
		}
		deleted = rule
		return txSvc.publish(txCtx, constants.AuditActionDelete, constants.ObjectTypePriceListRule, rule.ID, audit.ComputeChanges(rule, (*domain.PriceListRule)(nil)))
	})
	return deleted, tracing.Trace(span, apiErr)
}

// normalizeRuleScope drops blank and repeated IDs and never leaves a dimension nil.
func normalizeRuleScope(scope domain.PriceListRuleScope) domain.PriceListRuleScope {
	return domain.PriceListRuleScope{
		ItemIDs:        uniqueIDs(scope.ItemIDs),
		ProductLineIDs: uniqueIDs(scope.ProductLineIDs),
		CategoryIDs:    uniqueIDs(scope.CategoryIDs),
		AttributeIDs:   uniqueIDs(scope.AttributeIDs),
	}
}

// checkRuleRefs refuses scope IDs and units the account can't use (404 on the field), scope records that are archived
// unless the rule already held them (422 resource_archived), and scope dimensions longer than a rule keeps.
func (s *priceListSvcImpl) checkRuleRefs(ctx context.Context, accountID string, scope domain.PriceListRuleScope, price domain.PriceListPrice, old *domain.PriceListRule) *apierror.APIError {
	type dim struct {
		kind  domain.PriceListScopeKind
		param string
		ids   []string
		held  []string
	}
	var held domain.PriceListRuleScope
	if old != nil {
		held = old.Scope
	}
	dims := []dim{
		{domain.PriceListScopeItem, "scope.item_ids", scope.ItemIDs, held.ItemIDs},
		{domain.PriceListScopeProductLine, "scope.product_line_ids", scope.ProductLineIDs, held.ProductLineIDs},
		{domain.PriceListScopeCategory, "scope.category_ids", scope.CategoryIDs, held.CategoryIDs},
		{domain.PriceListScopeAttribute, "scope.attribute_ids", scope.AttributeIDs, held.AttributeIDs},
	}
	repo := s.repos.NewPriceListRepo()
	for _, d := range dims {
		if len(d.ids) > domain.MaxPriceListRuleScopeIDs {
			return apierror.NewFieldError(d.param, apierror.CodeInvalidFormat, fmt.Sprintf("A scope lists at most %d IDs per field.", domain.MaxPriceListRuleScopeIDs))
		}
		found, apiErr := repo.ScopeTargets(ctx, d.kind, accountID, d.ids)
		if apiErr != nil {
			return apiErr
		}
		for _, sid := range d.ids {
			active, ok := found[sid]
			if !ok {
				return apierror.NewNotFoundError(fmt.Sprintf("No %s found with the provided ID.", d.kind)).WithParam(d.param)
			}
			if !active && !slices.Contains(d.held, sid) {
				return domain.RequireAssignable(d.param, string(d.kind), constants.ArchiveStatusArchived)
			}
		}
	}
	units, apiErr := repo.ScopeTargets(ctx, domain.PriceListScopeUnit, accountID, price.UnitIDs())
	if apiErr != nil {
		return apiErr
	}
	for param, unitID := range rulePriceUnitParams(price) {
		if _, ok := units[unitID]; !ok {
			return apierror.NewNotFoundError("No unit found with the provided ID.").WithParam(param)
		}
	}
	return nil
}

// rulePriceUnitParams maps each request field naming a unit to the unit.
func rulePriceUnitParams(p domain.PriceListPrice) map[string]string {
	out := map[string]string{}
	if p.UnitPrice != nil {
		out["price.unit_price.per_unit_id"] = p.UnitPrice.PerUnitID
	}
	if p.UnitID != "" {
		out["price.unit_id"] = p.UnitID
	}
	for i, t := range p.Tiers {
		if t.UnitPrice != nil {
			out[fmt.Sprintf("price.tiers[%d].unit_price.per_unit_id", i)] = t.UnitPrice.PerUnitID
		}
	}
	return out
}

// normalizeRulePrice validates a requested price and sorts a tiered price's tiers by min_quantity.
func normalizeRulePrice(p domain.PriceListPrice) (domain.PriceListPrice, *apierror.APIError) {
	invalid := func(param, msg string) (domain.PriceListPrice, *apierror.APIError) {
		return p, apierror.NewFieldError(param, apierror.CodeInvalidFormat, msg)
	}
	switch p.Type {
	case constants.PriceListPriceTypeFixed:
		if p.UnitPrice == nil || p.PercentOfBase != nil || p.UnitID != "" || len(p.Tiers) > 0 {
			return invalid("price.unit_price", "A fixed price takes unit_price only.")
		}
		if apiErr := checkUnitPriceAmount("price.unit_price", p.UnitPrice); apiErr != nil {
			return p, apiErr
		}
	case constants.PriceListPriceTypePercentOfBase:
		if p.PercentOfBase == nil || p.UnitPrice != nil || p.UnitID != "" || len(p.Tiers) > 0 {
			return invalid("price.percent_of_base", "A percent_of_base price takes percent_of_base only.")
		}
		if apiErr := checkPercentOfBase("price.percent_of_base", *p.PercentOfBase); apiErr != nil {
			return p, apiErr
		}
	case constants.PriceListPriceTypeTiered:
		if p.UnitID == "" || p.UnitPrice != nil || p.PercentOfBase != nil {
			return invalid("price.unit_id", "A tiered price takes unit_id and tiers only.")
		}
		if len(p.Tiers) == 0 || len(p.Tiers) > domain.MaxPriceListRuleTiers {
			return invalid("price.tiers", fmt.Sprintf("A tiered price takes 1 to %d tiers.", domain.MaxPriceListRuleTiers))
		}
		tiers := slices.Clone(p.Tiers)
		for i, t := range tiers {
			param := fmt.Sprintf("price.tiers[%d]", i)
			qty, err := decimal.NewFromString(t.MinQuantity)
			if err != nil || qty.IsNegative() || qty.Exponent() < -6 {
				return invalid(param+".min_quantity", "Must be a decimal of 0 or more, with at most 6 decimal places.")
			}
			switch {
			case (t.PercentOfBase == nil) == (t.UnitPrice == nil):
				return invalid(param, "A tier takes exactly one of percent_of_base or unit_price.")
			case t.PercentOfBase != nil:
				if apiErr := checkPercentOfBase(param+".percent_of_base", *t.PercentOfBase); apiErr != nil {
					return p, apiErr
				}
			default:
				if apiErr := checkUnitPriceAmount(param+".unit_price", t.UnitPrice); apiErr != nil {
					return p, apiErr
				}
			}
		}
		slices.SortStableFunc(tiers, func(a, b domain.PriceListTier) int {
			return parseDecimal(a.MinQuantity).Cmp(parseDecimal(b.MinQuantity))
		})
		for i := 1; i < len(tiers); i++ {
			if parseDecimal(tiers[i].MinQuantity).Equal(parseDecimal(tiers[i-1].MinQuantity)) {
				return invalid("price.tiers", "Each tier needs a different min_quantity.")
			}
		}
		p.Tiers = tiers
	default:
		return invalid("price.type", "Must be fixed, percent_of_base or tiered.")
	}
	return p, nil
}

func checkPercentOfBase(param, v string) *apierror.APIError {
	pct, err := decimal.NewFromString(v)
	if err != nil || !pct.IsPositive() || pct.Exponent() < -6 || pct.GreaterThanOrEqual(decimal.NewFromInt(100000000000000)) {
		return apierror.NewFieldError(param, apierror.CodeInvalidFormat, "Must be a decimal greater than 0, with at most 6 decimal places.")
	}
	return nil
}

func checkUnitPriceAmount(param string, p *domain.UnitPrice) *apierror.APIError {
	amount, err := decimal.NewFromString(p.Amount)
	if err != nil || amount.IsNegative() || amount.Exponent() < -6 || amount.GreaterThanOrEqual(decimal.NewFromInt(100000000000000)) {
		return apierror.NewFieldError(param+".amount", apierror.CodeInvalidFormat, "Must be a decimal of 0 or more, with at most 6 decimal places.")
	}
	return nil
}
