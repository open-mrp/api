package service

import (
	"context"
	"fmt"
	"hash/fnv"
	"slices"
	"sort"
	"strings"

	types "github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/shared/appctx"
	"github.com/augno/api/shared/audit"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/id"
	"github.com/augno/api/shared/idempotency"
	"github.com/augno/api/shared/tracing"
)

type PropertySvcConfig struct {
	// Repos (required) is the repository factory.
	Repos domain.RepoFactory

	// MediatorFactory (required) builds the mediators used by this service.
	MediatorFactory domain.MediatorFactory

	// JobSvcFactory (required) builds the job service the async bulk operations settle through.
	JobSvcFactory domain.JobSvcFactory

	// TxManager (required) wraps multi-step operations in database transactions.
	TxManager TransactionManager
}

func (c *PropertySvcConfig) validate() error {
	if c.Repos == nil {
		return fmt.Errorf("property service: repos is required")
	}
	if c.MediatorFactory == nil {
		return fmt.Errorf("property service: mediator factory is required")
	}
	if c.JobSvcFactory == nil {
		return fmt.Errorf("property service: job service factory is required")
	}
	if c.TxManager == nil {
		return fmt.Errorf("property service: tx manager is required")
	}
	return nil
}

func NewPropertySvc(config *PropertySvcConfig) domain.PropertySvc {
	if err := config.validate(); err != nil {
		panic(err)
	}
	return &propertySvcImpl{
		repos:           config.Repos,
		mediatorFactory: config.MediatorFactory,
		jobSvcFactory:   config.JobSvcFactory,
		txManager:       config.TxManager,
	}
}

type propertySvcImpl struct {
	repos           domain.RepoFactory
	mediatorFactory domain.MediatorFactory
	jobSvcFactory   domain.JobSvcFactory
	txManager       TransactionManager
}

var propertySvcTracer = tracing.GetTracer("core-service.property_service")

func (s *propertySvcImpl) mediators() domain.Mediators {
	return s.mediatorFactory.Build(s.repos)
}

func (s *propertySvcImpl) withTx(ctx context.Context, fn func(context.Context, *propertySvcImpl) *apierror.APIError) *apierror.APIError {
	if s.txManager == nil {
		panic("txManager is nil")
	}
	return s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		txSvc := &propertySvcImpl{
			repos:           f,
			mediatorFactory: s.mediatorFactory,
			jobSvcFactory:   s.jobSvcFactory,
			txManager:       s.txManager,
		}
		return fn(txCtx, txSvc)
	})
}

func (s *propertySvcImpl) BatchGetPropertiesByIDs(ctx context.Context, ids []string) ([]*domain.Property, *apierror.APIError) {
	ctx, span := propertySvcTracer.Start(ctx, "service.property.batch_get_by_ids")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}
	meds := s.mediators()
	if apiErr := authorizeCatalogBatchRead(ctx, identity, span, meds, func() *apierror.APIError {
		return identity.CheckHasPermission(types.PermissionDomainProperties, types.ActionRead)
	}); apiErr != nil {
		return nil, apiErr
	}
	if len(ids) == 0 {
		return nil, nil
	}

	properties, apiErr := s.repos.NewPropertyRepo().GetByIDs(ctx, identity.Target.AccountID, ids)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if err := s.populatePropertyAttributes(ctx, identity.Target.AccountID, properties); err != nil {
		return nil, tracing.Trace(span, err)
	}

	return properties, nil
}

func (s *propertySvcImpl) ListProperties(ctx context.Context, params domain.ListPropertiesParams, includes []string) (*domain.ListPropertiesResult, *apierror.APIError) {
	ctx, span := propertySvcTracer.Start(ctx, "service.property.list")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}
	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainProperties, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if !identity.IsTargetAccountSet() {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("The Augno-Account-ID header is required."))
	}

	params.AccountID = identity.Target.AccountID

	result, apiErr := s.repos.NewPropertyRepo().List(ctx, params)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if slices.Contains(includes, "attributes") {
		if err := s.populatePropertyAttributes(ctx, params.AccountID, result.Properties); err != nil {
			return nil, tracing.Trace(span, err)
		}
	}

	return result, nil
}

func (s *propertySvcImpl) GetProperty(ctx context.Context, propertyID string, includes []string) (*domain.Property, *apierror.APIError) {
	ctx, span := propertySvcTracer.Start(ctx, "service.property.get")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}
	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainProperties, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if !identity.IsTargetAccountSet() {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("The Augno-Account-ID header is required."))
	}

	property, apiErr := s.repos.NewPropertyRepo().Get(ctx, domain.GetPropertyParams{
		PropertyID: propertyID,
		AccountID:  identity.Target.AccountID,
	})
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if slices.Contains(includes, "attributes") {
		if err := s.populatePropertyAttributes(ctx, identity.Target.AccountID, []*domain.Property{property}); err != nil {
			return nil, tracing.Trace(span, err)
		}
	}

	return property, nil
}

func (s *propertySvcImpl) populatePropertyAttributes(ctx context.Context, accountID string, properties []*domain.Property) *apierror.APIError {
	if len(properties) == 0 {
		return nil
	}

	propertyIDs := make([]string, len(properties))
	for i, p := range properties {
		propertyIDs[i] = p.ID
	}

	attributes, apiErr := s.repos.NewAttributeRepo().ListByPropertyIDs(ctx, accountID, propertyIDs)
	if apiErr != nil {
		return apiErr
	}

	attrsByProperty := make(map[string][]*domain.Attribute)
	for _, a := range attributes {
		attrsByProperty[a.PropertyID] = append(attrsByProperty[a.PropertyID], a)
	}

	for _, p := range properties {
		p.Attributes = attrsByProperty[p.ID]
	}

	return nil
}

func (s *propertySvcImpl) CreateProperty(ctx context.Context, params domain.CreatePropertyParams, includes []string) (*domain.Property, *apierror.APIError) {
	ctx, span := propertySvcTracer.Start(ctx, "service.property.create")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}
	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainProperties, types.ActionCreate); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if !identity.IsTargetAccountSet() {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("The Augno-Account-ID header is required."))
	}

	propertyID, apiErr := id.GenID(id.PropertyIDPrefix, nil)
	if apiErr != nil {
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
		cached, err := idempotency.UnmarshalCachedResponse[domain.Property](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.Property
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *propertySvcImpl) *apierror.APIError {
			txRepo := txSvc.repos.NewPropertyRepo()

			created, apiErr := txRepo.Create(txCtx, propertyID, params)
			if apiErr != nil {
				return apiErr
			}
			result = created

			changes := audit.ComputeChanges(nil, created)

			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionCreate,
				ResourceType: constants.ObjectTypeProperty,
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

		if slices.Contains(includes, "attributes") {
			if err := s.populatePropertyAttributes(ctx, params.AccountID, []*domain.Property{result}); err != nil {
				return nil, tracing.Trace(span, err)
			}
		}

		return result, nil

	default:
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Unexpected recovery point: "+idempotencyKey.RecoveryPoint))
	}
}

func (s *propertySvcImpl) UpdateProperty(ctx context.Context, params domain.UpdatePropertyParams, includes []string) (*domain.Property, *apierror.APIError) {
	ctx, span := propertySvcTracer.Start(ctx, "service.property.update")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}
	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainProperties, types.ActionUpdate); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if !identity.IsTargetAccountSet() {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("The Augno-Account-ID header is required."))
	}

	params.AccountID = identity.Target.AccountID

	meds := s.mediators()

	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, identity)
	if apiErr != nil {
		return nil, apiErr
	}

	switch domain.RecoveryPoint(idempotencyKey.RecoveryPoint) {
	case domain.RecoveryPointFinished:
		cached, err := idempotency.UnmarshalCachedResponse[domain.Property](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.Property
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *propertySvcImpl) *apierror.APIError {
			txRepo := txSvc.repos.NewPropertyRepo()

			old, apiErr := txRepo.Get(txCtx, domain.GetPropertyParams{
				PropertyID: params.PropertyID,
				AccountID:  params.AccountID,
			})
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
				ResourceType: constants.ObjectTypeProperty,
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

		if slices.Contains(includes, "attributes") {
			if err := s.populatePropertyAttributes(ctx, params.AccountID, []*domain.Property{result}); err != nil {
				return nil, tracing.Trace(span, err)
			}
		}

		return result, nil

	default:
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Unexpected recovery point: "+idempotencyKey.RecoveryPoint))
	}
}

func (s *propertySvcImpl) DeleteProperty(ctx context.Context, propertyID string) *apierror.APIError {
	ctx, span := propertySvcTracer.Start(ctx, "service.property.delete")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}
	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainProperties, types.ActionDelete); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	if !identity.IsTargetAccountSet() {
		return tracing.Trace(span, apierror.NewAuthenticationError("The Augno-Account-ID header is required."))
	}

	accountID := identity.Target.AccountID

	property, apiErr := s.repos.NewPropertyRepo().Get(ctx, domain.GetPropertyParams{
		PropertyID: propertyID,
		AccountID:  accountID,
	})
	if apiErr != nil {
		if apierror.IsNotFound(apiErr) {
			wasDeleted, deletedCheckErr := s.repos.NewDeletedRecordRepo().Exists(ctx, constants.DeletedRecordResourceTypeProperty, propertyID)
			if deletedCheckErr != nil {
				return tracing.Trace(span, deletedCheckErr)
			}
			if wasDeleted {
				return tracing.Trace(span, apierror.NewAlreadyDeletedError("This property has already been deleted and can no longer be modified."))
			}
		}
		return tracing.Trace(span, apiErr)
	}

	return s.withTx(ctx, func(txCtx context.Context, txSvc *propertySvcImpl) *apierror.APIError {
		txRepo := txSvc.repos.NewPropertyRepo()

		if apiErr := txSvc.repos.NewDeletedRecordRepo().Create(txCtx, constants.DeletedRecordResourceTypeProperty, property.ID, property); apiErr != nil {
			return apiErr
		}

		if apiErr := txRepo.DeleteAttributesByPropertyID(txCtx, propertyID, accountID); apiErr != nil {
			return apiErr
		}

		if apiErr := txRepo.Delete(txCtx, domain.DeletePropertyParams{
			PropertyID: propertyID,
			AccountID:  accountID,
		}); apiErr != nil {
			return apiErr
		}

		changes := audit.ComputeChanges(property, (*domain.Property)(nil))

		if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
			ServiceName:  domain.ServiceName,
			Action:       constants.AuditActionDelete,
			ResourceType: constants.ObjectTypeProperty,
			ResourceID:   property.ID,
			Changes:      changes,
		}); apiErr != nil {
			return apiErr
		}

		return nil
	})
}

// propertyAttributeKey builds the resolver key for a (property name, value) pair. Both
// sides are trimmed and matched case-insensitively, mirroring how the single-endpoint
// flows behave (TrimSpace on create + MySQL's case-insensitive collation on lookup).
func propertyAttributeKey(propertyName, value string) string {
	return strings.ToLower(strings.TrimSpace(propertyName)) + "\x00" + strings.ToLower(strings.TrimSpace(value))
}

// assignableAttributeColors are the nine named colors (everything except "default")
// that the dashboard assigns when an attribute is created without an explicit color.
var assignableAttributeColors = []constants.Color{
	constants.ColorBlue, constants.ColorBrown, constants.ColorGray,
	constants.ColorGreen, constants.ColorOrange, constants.ColorPink,
	constants.ColorPurple, constants.ColorRed, constants.ColorYellow,
}

// attributeColorFor deterministically picks a color for a bulk-created attribute. The
// manual create path picks one of the nine colors at random when omitted; bulk hashes
// the (property, value) pair instead so idempotent re-runs produce identical rows.
func attributeColorFor(propertyName, value string) string {
	h := fnv.New32a()
	_, _ = h.Write([]byte(strings.ToLower(propertyName) + "\x00" + strings.ToLower(value)))
	return string(assignableAttributeColors[h.Sum32()%uint32(len(assignableAttributeColors))]) // #nosec G115 -- fixed palette length
}

// findOrCreatePropertiesInTx is the single property bulk-upsert primitive: it finds the
// account's properties for the given names (matched case-insensitively) and creates any
// that are missing, publishing an audit event per creation. nameCasing maps lower-cased
// name → the casing to create with. Returns every involved property keyed by
// lower-cased name, plus the set of lower-cased names that were newly created. Existing
// properties are never renamed — the properties bulk-upsert endpoint layers rename
// semantics on top itself. Must run inside an open transaction. Shared by the
// properties bulk-upsert endpoint, item-category bulk upsert, and the per-type item
// bulk upserts.
func findOrCreatePropertiesInTx(txCtx context.Context, repos domain.RepoFactory, accountID string, nameCasing map[string]string) (map[string]*domain.Property, map[string]struct{}, *apierror.APIError) {
	if len(nameCasing) == 0 {
		return map[string]*domain.Property{}, map[string]struct{}{}, nil
	}

	names := make([]string, 0, len(nameCasing))
	for lower := range nameCasing {
		names = append(names, lower)
	}

	propRepo := repos.NewPropertyRepo()
	existing, apiErr := propRepo.FindByNames(txCtx, accountID, names)
	if apiErr != nil {
		return nil, nil, apiErr
	}
	byName := make(map[string]*domain.Property, len(nameCasing))
	for _, p := range existing {
		byName[strings.ToLower(p.Name)] = p
	}

	// Create the missing ones in a deterministic order.
	missing := make([]string, 0, len(nameCasing))
	for lower := range nameCasing {
		if _, ok := byName[lower]; !ok {
			missing = append(missing, lower)
		}
	}
	sort.Strings(missing)

	createdSet := make(map[string]struct{}, len(missing))
	for _, lower := range missing {
		propID, apiErr := id.GenID(id.PropertyIDPrefix, nil)
		if apiErr != nil {
			return nil, nil, apiErr
		}
		created, apiErr := propRepo.Create(txCtx, propID, domain.CreatePropertyParams{
			AccountID: accountID,
			Name:      nameCasing[lower],
		})
		if apiErr != nil {
			return nil, nil, apiErr
		}

		changes := audit.ComputeChanges(nil, created)
		if apiErr := audit.NewPublisher().Publish(txCtx, repos.NewOutboxRepo(), audit.EventData{
			ServiceName:  domain.ServiceName,
			Action:       constants.AuditActionCreate,
			ResourceType: constants.ObjectTypeProperty,
			ResourceID:   created.ID,
			Changes:      changes,
		}); apiErr != nil {
			return nil, nil, apiErr
		}

		byName[lower] = created
		createdSet[lower] = struct{}{}
	}

	return byName, createdSet, nil
}

// resolvePropertyAttributesInTx resolves every (property name, value) pair across a
// batch to an attribute ID, creating any missing properties and attributes. Returns the
// attribute resolver (keyed by propertyAttributeKey) and a property-ID map keyed by
// lower-cased property name (used to link properties onto a category). Caller-agnostic:
// shared by the per-type item bulk upserts and usable by any property/attribute batch.
func resolvePropertyAttributesInTx(txCtx context.Context, repos domain.RepoFactory, accountID string, properties []domain.UpsertItemPropertyParams) (map[string]string, map[string]string, *apierror.APIError) {
	// Collect unique property names and (name, value) pairs, keeping first-seen casing.
	// Names and values are trimmed and matched case-insensitively, mirroring the manual
	// create path (TrimSpace + case-insensitive DB collation); pairs that are empty
	// after trimming are skipped.
	nameCasing := make(map[string]string)
	type nv struct{ name, value string }
	pairSet := make(map[string]nv)
	for _, p := range properties {
		name := strings.TrimSpace(p.Name)
		value := strings.TrimSpace(p.Value)
		if name == "" || value == "" {
			continue
		}
		lower := strings.ToLower(name)
		if _, ok := nameCasing[lower]; !ok {
			nameCasing[lower] = name
		}
		pairSet[propertyAttributeKey(name, value)] = nv{name: lower, value: value}
	}
	if len(pairSet) == 0 {
		return map[string]string{}, map[string]string{}, nil
	}

	propByName, _, apiErr := findOrCreatePropertiesInTx(txCtx, repos, accountID, nameCasing)
	if apiErr != nil {
		return nil, nil, apiErr
	}

	// Load existing attributes for all involved properties, keyed by
	// (propID, lower-cased value) to mirror the DB's case-insensitive matching, and
	// count them per property so new attributes get the next 1-based sort order —
	// the same order the manual create path assigns.
	propIDs := make([]string, 0, len(propByName))
	for _, p := range propByName {
		propIDs = append(propIDs, p.ID)
	}
	existingAttrs, apiErr := repos.NewAttributeRepo().ListByPropertyIDs(txCtx, accountID, propIDs)
	if apiErr != nil {
		return nil, nil, apiErr
	}
	attrByPropValue := make(map[string]string, len(existingAttrs))
	attrCountByProp := make(map[string]int32, len(propByName))
	for _, a := range existingAttrs {
		attrByPropValue[a.PropertyID+"\x00"+strings.ToLower(strings.TrimSpace(a.Value))] = a.ID
		attrCountByProp[a.PropertyID]++
	}

	// Enforce the account-wide attribute value uniqueness the manual create path
	// enforces (ExistsByValueInAccount → 409): a value may exist under only one
	// property per account. Collect ALL offences — both conflicts with existing
	// attributes and duplicate values across properties within this request — into a
	// single validation error instead of failing on the first.
	type pendingCreate struct {
		pair nv
		prop *domain.Property
	}
	var creates []pendingCreate
	for _, pair := range pairSet {
		prop := propByName[pair.name]
		if _, ok := attrByPropValue[prop.ID+"\x00"+strings.ToLower(pair.value)]; !ok {
			creates = append(creates, pendingCreate{pair: pair, prop: prop})
		}
	}
	if len(creates) > 0 {
		sort.Slice(creates, func(i, j int) bool {
			if creates[i].pair.name != creates[j].pair.name {
				return creates[i].pair.name < creates[j].pair.name
			}
			return creates[i].pair.value < creates[j].pair.value
		})

		textSet := make(map[string]string, len(creates)) // lower value → value
		batchProps := make(map[string]map[string]struct{}, len(creates))
		for _, c := range creates {
			lowerVal := strings.ToLower(c.pair.value)
			textSet[lowerVal] = c.pair.value
			if batchProps[lowerVal] == nil {
				batchProps[lowerVal] = map[string]struct{}{}
			}
			batchProps[lowerVal][nameCasing[c.pair.name]] = struct{}{}
		}
		texts := make([]string, 0, len(textSet))
		for _, v := range textSet {
			texts = append(texts, v)
		}

		existingByText, apiErr := repos.NewAttributeRepo().FindByTextsInAccount(txCtx, accountID, texts)
		if apiErr != nil {
			return nil, nil, apiErr
		}
		dbByText := make(map[string]*domain.AttributeTextMatch, len(existingByText))
		for _, m := range existingByText {
			key := strings.ToLower(strings.TrimSpace(m.Text))
			if _, ok := dbByText[key]; !ok {
				dbByText[key] = m
			}
		}

		var problems []string
		reportedBatchDup := make(map[string]struct{})
		for _, c := range creates {
			lowerVal := strings.ToLower(c.pair.value)
			if m := dbByText[lowerVal]; m != nil && m.PropertyID != c.prop.ID {
				problems = append(problems, fmt.Sprintf("value %q already exists under property %q", c.pair.value, m.PropertyName))
				continue
			}
			if len(batchProps[lowerVal]) > 1 {
				if _, done := reportedBatchDup[lowerVal]; done {
					continue
				}
				reportedBatchDup[lowerVal] = struct{}{}
				names := make([]string, 0, len(batchProps[lowerVal]))
				for n := range batchProps[lowerVal] {
					names = append(names, n)
				}
				sort.Strings(names)
				problems = append(problems, fmt.Sprintf("value %q is used under multiple properties (%s) in this request", c.pair.value, strings.Join(names, ", ")))
			}
		}
		if len(problems) > 0 {
			return nil, nil, apierror.NewValidationError(
				"Invalid property values — " + strings.Join(problems, "; ") + ". Attribute values must be unique across the account.")
		}
	}

	// Resolve each pair, creating missing attributes.
	resolver := make(map[string]string, len(pairSet))
	for key, pair := range pairSet {
		prop := propByName[pair.name]
		pvKey := prop.ID + "\x00" + strings.ToLower(pair.value)
		attrID, ok := attrByPropValue[pvKey]
		if !ok {
			newAttrID, apiErr := id.GenID(id.AttributeIDPrefix, nil)
			if apiErr != nil {
				return nil, nil, apiErr
			}
			attrCountByProp[prop.ID]++
			if _, apiErr := repos.NewAttributeRepo().Create(txCtx, newAttrID, domain.CreateAttributeParams{
				Value:      pair.value,
				PropertyID: prop.ID,
				AccountID:  accountID,
				ColorCode:  attributeColorFor(pair.name, pair.value),
				SortOrder:  attrCountByProp[prop.ID],
			}); apiErr != nil {
				return nil, nil, apiErr
			}
			attrByPropValue[pvKey] = newAttrID
			attrID = newAttrID
		}
		resolver[key] = attrID
	}

	// Property-ID map keyed by lower-cased name, for linking properties to categories.
	propIDByName := make(map[string]string, len(propByName))
	for lower, p := range propByName {
		propIDByName[lower] = p.ID
	}

	return resolver, propIDByName, nil
}

// attributeIDsForProperties maps a single row's (name, value) properties to the
// attribute IDs produced by resolvePropertyAttributesInTx.
func attributeIDsForProperties(properties []domain.UpsertItemPropertyParams, resolver map[string]string) []string {
	if len(properties) == 0 {
		return nil
	}
	ids := make([]string, 0, len(properties))
	for _, p := range properties {
		if attrID, ok := resolver[propertyAttributeKey(p.Name, p.Value)]; ok && attrID != "" {
			ids = append(ids, attrID)
		}
	}
	return ids
}

// --- Export ---
