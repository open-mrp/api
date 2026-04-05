package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/shared/appctx"
	"github.com/augno/api/shared/audit"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/id"
	"github.com/augno/api/shared/idempotency"
	"github.com/augno/api/shared/tracing"
)

var productionStepSvcTracer = tracing.GetTracer("core-service.production_step_service")

type productionStepSvcImpl struct {
	repos           domain.RepoFactory
	mediatorFactory domain.MediatorFactory
	txManager       TransactionManager
}

type ProductionStepSvcConfig struct {
	Repos           domain.RepoFactory
	MediatorFactory domain.MediatorFactory
	TxManager       TransactionManager
}

func (c *ProductionStepSvcConfig) validate() error {
	if c.Repos == nil {
		return fmt.Errorf("production step service: repos is required")
	}
	if c.MediatorFactory == nil {
		return fmt.Errorf("production step service: mediator factory is required")
	}
	if c.TxManager == nil {
		return fmt.Errorf("production step service: tx manager is required")
	}
	return nil
}

func NewProductionStepSvc(config *ProductionStepSvcConfig) domain.ProductionStepSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}
	return &productionStepSvcImpl{
		repos:           config.Repos,
		mediatorFactory: config.MediatorFactory,
		txManager:       config.TxManager,
	}
}

func (s *productionStepSvcImpl) mediators() domain.Mediators {
	return s.mediatorFactory.Build(s.repos)
}

func (s *productionStepSvcImpl) withTx(ctx context.Context, fn func(context.Context, *productionStepSvcImpl) *apierror.APIError) *apierror.APIError {
	return s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		txSvc := &productionStepSvcImpl{
			repos:           f,
			mediatorFactory: s.mediatorFactory,
			txManager:       s.txManager,
		}
		return fn(txCtx, txSvc)
	})
}

func (s *productionStepSvcImpl) ListProductionSteps(ctx context.Context, params domain.ListProductionStepsParams) (*domain.ListProductionStepsResult, *apierror.APIError) {
	ctx, span := productionStepSvcTracer.Start(ctx, "service.production_step.list")
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

	params.AccountID = identity.Target.AccountID

	return s.repos.NewProductionStepRepo().List(ctx, params)
}

func (s *productionStepSvcImpl) GetProductionStep(ctx context.Context, stepID string) (*domain.ProductionStep, *apierror.APIError) {
	ctx, span := productionStepSvcTracer.Start(ctx, "service.production_step.get")
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

	return s.repos.NewProductionStepRepo().Get(ctx, identity.Target.AccountID, stepID)
}

func (s *productionStepSvcImpl) CreateProductionStep(ctx context.Context, params domain.CreateProductionStepParams) (*domain.ProductionStep, *apierror.APIError) {
	ctx, span := productionStepSvcTracer.Start(ctx, "service.production_step.create")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainProductionSteps, types.ActionCreate); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	params.AccountID = identity.Target.AccountID

	// Generate IDs.
	stepID, apiErr := id.GenID(id.ProductionStepIDPrefix, nil)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	laborRateID, apiErr := id.GenID(id.RateIDPrefix, nil)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	laborTimeID, apiErr := id.GenID(id.RateIDPrefix, nil)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	overheadRateID, apiErr := id.GenID(id.RateIDPrefix, nil)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	productionID, apiErr := id.GenID(id.ProductionIDPrefix, nil)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	quantityID, apiErr := id.GenID(id.QuantityIDPrefix, nil)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	meds := s.mediators()

	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, identity)
	if apiErr != nil {
		return nil, apiErr
	}

	switch domain.RecoveryPoint(idempotencyKey.RecoveryPoint) {
	case domain.RecoveryPointFinished:
		cached, err := idempotency.UnmarshalCachedResponse[domain.ProductionStep](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.ProductionStep
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *productionStepSvcImpl) *apierror.APIError {
			txRepo := txSvc.repos.NewProductionStepRepo()

			// Insert rates.
			if apiErr := txRepo.InsertRate(txCtx, laborRateID, params.LaborRate); apiErr != nil {
				return apiErr
			}
			if apiErr := txRepo.InsertRate(txCtx, laborTimeID, params.LaborTime); apiErr != nil {
				return apiErr
			}
			if apiErr := txRepo.InsertRate(txCtx, overheadRateID, params.OverheadRate); apiErr != nil {
				return apiErr
			}

			// Insert the production step.
			if apiErr := txRepo.InsertStep(txCtx, stepID, params.Name, params.Notes, params.LevelingFactor, params.Allowances, laborRateID, laborTimeID, overheadRateID, params.ScanningStationID, params.DepartmentID, params.AccountID); apiErr != nil {
				return apiErr
			}

			// Insert production quantity and production output.
			if apiErr := txRepo.InsertQuantity(txCtx, quantityID, params.Production.QuantityValue, params.Production.QuantityUnitID); apiErr != nil {
				return apiErr
			}
			if apiErr := txRepo.InsertProduction(txCtx, productionID, params.Production.ItemID, quantityID, stepID); apiErr != nil {
				return apiErr
			}

			// Create consumptions.
			consumptionRepo := txSvc.repos.NewConsumptionRepo()
			for _, cp := range params.Consumptions {
				consumptionID, apiErr := id.GenID(id.ConsumptionIDPrefix, nil)
				if apiErr != nil {
					return apiErr
				}
				cQuantityID, apiErr := id.GenID(id.QuantityIDPrefix, nil)
				if apiErr != nil {
					return apiErr
				}
				wasteQuantityID, apiErr := id.GenID(id.QuantityIDPrefix, nil)
				if apiErr != nil {
					return apiErr
				}
				_, apiErr = consumptionRepo.Create(txCtx, consumptionID, cQuantityID, wasteQuantityID, domain.CreateConsumptionParams{
					AccountID:           params.AccountID,
					ProductionStepID:    stepID,
					ItemID:              cp.ItemID,
					QuantityValue:       cp.QuantityValue,
					QuantityUnitID:      cp.QuantityUnitID,
					WasteQuantityValue:  cp.WasteQuantityValue,
					WasteQuantityUnitID: cp.WasteQuantityUnitID,
					Instructions:        cp.Instructions,
				})
				if apiErr != nil {
					return apiErr
				}
			}

			// Re-fetch the full step to return.
			fetched, apiErr := txRepo.Get(txCtx, params.AccountID, stepID)
			if apiErr != nil {
				return apiErr
			}
			result = fetched

			changes := audit.ComputeChanges(nil, result)
			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionCreate,
				ResourceType: constants.ObjectTypeProductionStep,
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

		// Link the production flow outside the main transaction.
		if apiErr := meds.ProductionFlow.LinkFlow(ctx, stepID, params.AccountID); apiErr != nil {
			// Non-fatal: log but don't fail the create.
			_ = apiErr
		}

		return result, nil

	default:
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Unexpected recovery point: "+idempotencyKey.RecoveryPoint))
	}
}

// validLaborTimeUnits are the accepted abbreviations for the labor time numerator unit.
var validLaborTimeUnits = map[string]bool{
	"hr":     true,
	"minute": true,
	"min":    true,
	"second": true,
	"sec":    true,
	"day":    true,
}

// BulkCreateProductionSteps creates multiple production steps in a single operation.
// Steps that already exist (by name) are updated; new ones are created. Individual row
// failures are captured in the results rather than failing the whole operation.
func (s *productionStepSvcImpl) BulkCreateProductionSteps(ctx context.Context, params domain.BulkCreateProductionStepsParams) ([]domain.BulkCreateProductionStepResult, *apierror.APIError) {
	ctx, span := productionStepSvcTracer.Start(ctx, "service.production_step.bulk_create")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainProductionSteps, types.ActionCreate); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	accountID := identity.Target.AccountID

	meds := s.mediators()

	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, identity)
	if apiErr != nil {
		return nil, apiErr
	}

	switch domain.RecoveryPoint(idempotencyKey.RecoveryPoint) {
	case domain.RecoveryPointFinished:
		cached, err := idempotency.UnmarshalCachedResponse[[]domain.BulkCreateProductionStepResult](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return *cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		// Resolve units needed for rates: dollar (numerator for labor/overhead), hr (denominator for labor/overhead).
		unitRepo := s.repos.NewUnitRepo()
		abbreviationsNeeded := []string{"$", "hr"}
		// Collect all labor time unit abbreviations from input.
		for _, step := range params.Steps {
			ltu := "hr"
			if step.LaborTimeUnit != nil && *step.LaborTimeUnit != "" {
				ltu = *step.LaborTimeUnit
			}
			abbreviationsNeeded = append(abbreviationsNeeded, ltu)
		}
		units, apiErr := unitRepo.FindByAbbreviations(ctx, accountID, abbreviationsNeeded)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}

		unitByAbbr := make(map[string]string) // abbreviation -> unit ID
		for _, u := range units {
			unitByAbbr[strings.ToLower(u.Abbreviation)] = u.ID
		}

		dollarUnitID := unitByAbbr["$"]
		hrUnitID := unitByAbbr["hr"]

		if dollarUnitID == "" || hrUnitID == "" {
			return nil, tracing.Trace(span, apierror.NewInternalError(nil, "Could not resolve required units (dollar, hr)."))
		}

		// Collect all SKUs across all rows to batch-resolve items.
		allSKUs := make(map[string]struct{})
		for _, step := range params.Steps {
			for _, c := range step.Consumptions {
				allSKUs[c.SKU] = struct{}{}
			}
			for _, p := range step.Productions {
				allSKUs[p.SKU] = struct{}{}
			}
		}
		skuList := make([]string, 0, len(allSKUs))
		for sku := range allSKUs {
			skuList = append(skuList, sku)
		}

		itemBySKU := make(map[string]domain.ItemSKUInfo) // sku -> ItemSKUInfo
		if len(skuList) > 0 {
			items, apiErr := s.repos.NewItemRepo().FetchItemsBySKU(ctx, accountID, skuList)
			if apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			for _, item := range items {
				itemBySKU[item.SKU] = item
			}
		}

		results := make([]domain.BulkCreateProductionStepResult, len(params.Steps))

		for i, input := range params.Steps {
			results[i] = s.bulkCreateSingleStep(ctx, accountID, input, itemBySKU, unitByAbbr, dollarUnitID, hrUnitID)
		}

		cacheErr := s.withTx(ctx, func(txCtx context.Context, txSvc *productionStepSvcImpl) *apierror.APIError {
			return txSvc.mediators().Idempotency.CacheSuccessResponse(txCtx, idempotencyKey.TypeID, results)
		})
		if cacheErr != nil {
			// Caching failure is not fatal; return results anyway.
			_ = cacheErr
		}

		return results, nil

	default:
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Unexpected recovery point: "+idempotencyKey.RecoveryPoint))
	}
}

// bulkCreateSingleStep processes a single row in the bulk create operation.
// Errors are captured in the result rather than propagated.
func (s *productionStepSvcImpl) bulkCreateSingleStep(
	ctx context.Context,
	accountID string,
	input domain.BulkCreateProductionStepInput,
	itemBySKU map[string]domain.ItemSKUInfo,
	unitByAbbr map[string]string,
	dollarUnitID, hrUnitID string,
) domain.BulkCreateProductionStepResult {
	skipResult := func(name, reason string) domain.BulkCreateProductionStepResult {
		return domain.BulkCreateProductionStepResult{
			Name:    name,
			Success: false,
			Error:   &reason,
			Action:  "skipped",
		}
	}

	// Validation: name required.
	if strings.TrimSpace(input.Name) == "" {
		return skipResult(input.Name, "Name is required")
	}

	// Validation: at least one production required.
	if len(input.Productions) == 0 {
		return skipResult(input.Name, "No productions found in row")
	}

	// Resolve scanning station by name if provided.
	var scanningStationID *string
	if input.Station != nil && *input.Station != "" {
		stationID, apiErr := s.repos.NewScanningStationRepo().FindIDByName(ctx, accountID, *input.Station)
		if apiErr != nil {
			return skipResult(input.Name, "Failed to resolve station")
		}
		if stationID == nil {
			return skipResult(input.Name, "Invalid station name")
		}
		scanningStationID = stationID
	}

	// Validate labor time unit.
	laborTimeUnit := "hr"
	if input.LaborTimeUnit != nil && *input.LaborTimeUnit != "" {
		laborTimeUnit = *input.LaborTimeUnit
	}
	if !validLaborTimeUnits[strings.ToLower(laborTimeUnit)] {
		return skipResult(input.Name, "Invalid labor time unit")
	}

	laborTimeUnitID, ok := unitByAbbr[strings.ToLower(laborTimeUnit)]
	if !ok {
		return skipResult(input.Name, "Invalid labor time unit")
	}

	// Resolve production items.
	for _, p := range input.Productions {
		if _, ok := itemBySKU[p.SKU]; !ok {
			return skipResult(input.Name, fmt.Sprintf("Missing item for production SKU: %s", p.SKU))
		}
	}

	// Resolve consumption items.
	for _, c := range input.Consumptions {
		if _, ok := itemBySKU[c.SKU]; !ok {
			return skipResult(input.Name, fmt.Sprintf("Missing item for consumption SKU: %s", c.SKU))
		}
	}

	// Determine leveling factor and allowances (defaults to "0").
	levelingFactor := "0"
	if input.LevelingFactor != nil {
		levelingFactor = *input.LevelingFactor
	}
	allowances := "0"
	if input.Allowances != nil {
		allowances = *input.Allowances
	}

	// Check if step already exists by name.
	stepRepo := s.repos.NewProductionStepRepo()
	existingStepID, apiErr := stepRepo.FindIDByName(ctx, accountID, input.Name)
	if apiErr != nil {
		return skipResult(input.Name, "Failed to check existing step")
	}

	if existingStepID != nil {
		// UPDATE existing step.
		return s.bulkUpdateExistingStep(ctx, accountID, *existingStepID, input, itemBySKU, unitByAbbr, dollarUnitID, hrUnitID, laborTimeUnitID, levelingFactor, allowances, scanningStationID)
	}

	// CREATE new step.
	return s.bulkCreateNewStep(ctx, accountID, input, itemBySKU, dollarUnitID, hrUnitID, laborTimeUnitID, levelingFactor, allowances, scanningStationID)
}

// bulkUpdateExistingStep updates an existing production step with new data (delete old consumptions/productions, recreate).
func (s *productionStepSvcImpl) bulkUpdateExistingStep(
	ctx context.Context,
	accountID, stepID string,
	input domain.BulkCreateProductionStepInput,
	itemBySKU map[string]domain.ItemSKUInfo,
	_ map[string]string,
	_, _, _ string,
	levelingFactor, allowances string,
	scanningStationID *string,
) domain.BulkCreateProductionStepResult {
	// Fetch old state before mutation for audit diff.
	old, fetchErr := s.repos.NewProductionStepRepo().Get(ctx, accountID, stepID)
	if fetchErr != nil {
		msg := "Update failed"
		return domain.BulkCreateProductionStepResult{
			Name:             input.Name,
			Success:          false,
			Error:            &msg,
			ProductionStepID: &stepID,
			Action:           "skipped",
		}
	}

	apiErr := s.withTx(ctx, func(txCtx context.Context, txSvc *productionStepSvcImpl) *apierror.APIError {
		txStepRepo := txSvc.repos.NewProductionStepRepo()

		// Delete existing consumptions and productions.
		if apiErr := txStepRepo.DeleteConsumptionsByStepID(txCtx, stepID); apiErr != nil {
			return apiErr
		}
		if apiErr := txStepRepo.DeleteProductionsByStepID(txCtx, stepID); apiErr != nil {
			return apiErr
		}

		// Update the step record.
		if apiErr := txStepRepo.UpdateStepFull(txCtx, stepID, accountID, levelingFactor, allowances, scanningStationID); apiErr != nil {
			return apiErr
		}

		// Recreate productions.
		for _, p := range input.Productions {
			item := itemBySKU[p.SKU]
			productionID, apiErr := id.GenID(id.ProductionIDPrefix, nil)
			if apiErr != nil {
				return apiErr
			}
			quantityID, apiErr := id.GenID(id.QuantityIDPrefix, nil)
			if apiErr != nil {
				return apiErr
			}
			if apiErr := txStepRepo.InsertQuantity(txCtx, quantityID, p.Measure, item.BaseUnitID); apiErr != nil {
				return apiErr
			}
			if apiErr := txStepRepo.InsertProduction(txCtx, productionID, item.ItemID, quantityID, stepID); apiErr != nil {
				return apiErr
			}
		}

		// Recreate consumptions.
		consumptionRepo := txSvc.repos.NewConsumptionRepo()
		for _, c := range input.Consumptions {
			item := itemBySKU[c.SKU]
			consumptionID, apiErr := id.GenID(id.ConsumptionIDPrefix, nil)
			if apiErr != nil {
				return apiErr
			}
			cQuantityID, apiErr := id.GenID(id.QuantityIDPrefix, nil)
			if apiErr != nil {
				return apiErr
			}
			wasteQuantityID, apiErr := id.GenID(id.QuantityIDPrefix, nil)
			if apiErr != nil {
				return apiErr
			}
			_, apiErr = consumptionRepo.Create(txCtx, consumptionID, cQuantityID, wasteQuantityID, domain.CreateConsumptionParams{
				AccountID:           accountID,
				ProductionStepID:    stepID,
				ItemID:              item.ItemID,
				QuantityValue:       c.Measure,
				QuantityUnitID:      item.BaseUnitID,
				WasteQuantityValue:  "0",
				WasteQuantityUnitID: item.BaseUnitID,
				Instructions:        c.Instructions,
			})
			if apiErr != nil {
				return apiErr
			}
		}

		// Re-fetch updated state for audit diff.
		updated, apiErr := txStepRepo.Get(txCtx, accountID, stepID)
		if apiErr != nil {
			return apiErr
		}

		changes := audit.ComputeChanges(old, updated)
		if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
			ServiceName:  domain.ServiceName,
			Action:       constants.AuditActionUpdate,
			ResourceType: constants.ObjectTypeProductionStep,
			ResourceID:   stepID,
			Changes:      changes,
		}); apiErr != nil {
			return apiErr
		}

		return nil
	})
	if apiErr != nil {
		msg := "Update failed"
		return domain.BulkCreateProductionStepResult{
			Name:             input.Name,
			Success:          false,
			Error:            &msg,
			ProductionStepID: &stepID,
			Action:           "skipped",
		}
	}

	// Link production flow (non-fatal).
	if linkErr := s.mediators().ProductionFlow.LinkFlow(ctx, stepID, accountID); linkErr != nil {
		_ = linkErr
	}

	return domain.BulkCreateProductionStepResult{
		Name:             input.Name,
		Success:          true,
		ProductionStepID: &stepID,
		Action:           "updated",
	}
}

// bulkCreateNewStep creates a brand new production step with rates, production, and consumptions.
func (s *productionStepSvcImpl) bulkCreateNewStep(
	ctx context.Context,
	accountID string,
	input domain.BulkCreateProductionStepInput,
	itemBySKU map[string]domain.ItemSKUInfo,
	dollarUnitID, hrUnitID, laborTimeUnitID string,
	levelingFactor, allowances string,
	scanningStationID *string,
) domain.BulkCreateProductionStepResult {
	// Get the base unit of the first production item for labor time denominator.
	firstProdItem := itemBySKU[input.Productions[0].SKU]
	laborTimeDenUnitID := firstProdItem.BaseUnitID

	// Generate IDs.
	stepID, apiErr := id.GenID(id.ProductionStepIDPrefix, nil)
	if apiErr != nil {
		msg := "Create failed"
		return domain.BulkCreateProductionStepResult{Name: input.Name, Success: false, Error: &msg, Action: "skipped"}
	}
	laborRateID, apiErr := id.GenID(id.RateIDPrefix, nil)
	if apiErr != nil {
		msg := "Create failed"
		return domain.BulkCreateProductionStepResult{Name: input.Name, Success: false, Error: &msg, Action: "skipped"}
	}
	laborTimeID, apiErr := id.GenID(id.RateIDPrefix, nil)
	if apiErr != nil {
		msg := "Create failed"
		return domain.BulkCreateProductionStepResult{Name: input.Name, Success: false, Error: &msg, Action: "skipped"}
	}
	overheadRateID, apiErr := id.GenID(id.RateIDPrefix, nil)
	if apiErr != nil {
		msg := "Create failed"
		return domain.BulkCreateProductionStepResult{Name: input.Name, Success: false, Error: &msg, Action: "skipped"}
	}

	apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *productionStepSvcImpl) *apierror.APIError {
		txStepRepo := txSvc.repos.NewProductionStepRepo()

		// Insert rates: labor rate ($/hr), overhead rate ($/hr), labor time (laborTimeUnit/productionUnit).
		if apiErr := txStepRepo.InsertRate(txCtx, laborRateID, domain.CreateRateParams{
			Value:             input.LaborRate,
			NumeratorUnitID:   dollarUnitID,
			DenominatorUnitID: hrUnitID,
		}); apiErr != nil {
			return apiErr
		}
		if apiErr := txStepRepo.InsertRate(txCtx, laborTimeID, domain.CreateRateParams{
			Value:             input.LaborTime,
			NumeratorUnitID:   laborTimeUnitID,
			DenominatorUnitID: laborTimeDenUnitID,
		}); apiErr != nil {
			return apiErr
		}
		if apiErr := txStepRepo.InsertRate(txCtx, overheadRateID, domain.CreateRateParams{
			Value:             input.OverheadRate,
			NumeratorUnitID:   dollarUnitID,
			DenominatorUnitID: hrUnitID,
		}); apiErr != nil {
			return apiErr
		}

		// Insert the production step.
		if apiErr := txStepRepo.InsertStep(txCtx, stepID, input.Name, nil, levelingFactor, allowances, laborRateID, laborTimeID, overheadRateID, scanningStationID, nil, accountID); apiErr != nil {
			return apiErr
		}

		// Insert productions.
		for _, p := range input.Productions {
			item := itemBySKU[p.SKU]
			productionID, apiErr := id.GenID(id.ProductionIDPrefix, nil)
			if apiErr != nil {
				return apiErr
			}
			quantityID, apiErr := id.GenID(id.QuantityIDPrefix, nil)
			if apiErr != nil {
				return apiErr
			}
			if apiErr := txStepRepo.InsertQuantity(txCtx, quantityID, p.Measure, item.BaseUnitID); apiErr != nil {
				return apiErr
			}
			if apiErr := txStepRepo.InsertProduction(txCtx, productionID, item.ItemID, quantityID, stepID); apiErr != nil {
				return apiErr
			}
		}

		// Insert consumptions.
		consumptionRepo := txSvc.repos.NewConsumptionRepo()
		for _, c := range input.Consumptions {
			item := itemBySKU[c.SKU]
			consumptionID, apiErr := id.GenID(id.ConsumptionIDPrefix, nil)
			if apiErr != nil {
				return apiErr
			}
			cQuantityID, apiErr := id.GenID(id.QuantityIDPrefix, nil)
			if apiErr != nil {
				return apiErr
			}
			wasteQuantityID, apiErr := id.GenID(id.QuantityIDPrefix, nil)
			if apiErr != nil {
				return apiErr
			}
			_, apiErr = consumptionRepo.Create(txCtx, consumptionID, cQuantityID, wasteQuantityID, domain.CreateConsumptionParams{
				AccountID:           accountID,
				ProductionStepID:    stepID,
				ItemID:              item.ItemID,
				QuantityValue:       c.Measure,
				QuantityUnitID:      item.BaseUnitID,
				WasteQuantityValue:  "0",
				WasteQuantityUnitID: item.BaseUnitID,
				Instructions:        c.Instructions,
			})
			if apiErr != nil {
				return apiErr
			}
		}

		// Re-fetch the created step for audit diff.
		created, apiErr := txStepRepo.Get(txCtx, accountID, stepID)
		if apiErr != nil {
			return apiErr
		}

		changes := audit.ComputeChanges(nil, created)
		if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
			ServiceName:  domain.ServiceName,
			Action:       constants.AuditActionCreate,
			ResourceType: constants.ObjectTypeProductionStep,
			ResourceID:   stepID,
			Changes:      changes,
		}); apiErr != nil {
			return apiErr
		}

		return nil
	})
	if apiErr != nil {
		msg := "Create failed"
		return domain.BulkCreateProductionStepResult{Name: input.Name, Success: false, Error: &msg, Action: "skipped"}
	}

	// Link production flow (non-fatal).
	if linkErr := s.mediators().ProductionFlow.LinkFlow(ctx, stepID, accountID); linkErr != nil {
		_ = linkErr
	}

	return domain.BulkCreateProductionStepResult{
		Name:             input.Name,
		Success:          true,
		ProductionStepID: &stepID,
		Action:           "created",
	}
}

func (s *productionStepSvcImpl) UpdateProductionStep(ctx context.Context, params domain.UpdateProductionStepParams) (*domain.ProductionStep, *apierror.APIError) {
	ctx, span := productionStepSvcTracer.Start(ctx, "service.production_step.update")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainProductionSteps, types.ActionUpdate); apiErr != nil {
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
		cached, err := idempotency.UnmarshalCachedResponse[domain.ProductionStep](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.ProductionStep
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *productionStepSvcImpl) *apierror.APIError {
			txRepo := txSvc.repos.NewProductionStepRepo()

			old, apiErr := txRepo.Get(txCtx, params.AccountID, params.ProductionStepID)
			if apiErr != nil {
				return apiErr
			}

			// Check for name conflicts if name is changing.
			if params.Name != nil {
				exists, apiErr := txRepo.ExistsByName(txCtx, params.AccountID, *params.Name, &params.ProductionStepID)
				if apiErr != nil {
					return apiErr
				}
				if exists {
					return apierror.NewConflictErrorWithParam("A production step with this name already exists.", "name")
				}
			}

			if apiErr := txRepo.Update(txCtx, params); apiErr != nil {
				return apiErr
			}

			fetched, apiErr := txRepo.Get(txCtx, params.AccountID, params.ProductionStepID)
			if apiErr != nil {
				return apiErr
			}
			result = fetched

			changes := audit.ComputeChanges(old, result)
			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionUpdate,
				ResourceType: constants.ObjectTypeProductionStep,
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

func (s *productionStepSvcImpl) DeleteProductionStep(ctx context.Context, stepID string) *apierror.APIError {
	ctx, span := productionStepSvcTracer.Start(ctx, "service.production_step.delete")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainProductionSteps, types.ActionDelete); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	accountID := identity.Target.AccountID

	// Fetch the step before deletion for the audit diff.
	step, apiErr := s.repos.NewProductionStepRepo().Get(ctx, accountID, stepID)
	if apiErr != nil {
		if apierror.IsNotFound(apiErr) {
			wasDeleted, deletedCheckErr := s.repos.NewDeletedRecordRepo().Exists(ctx, constants.DeletedRecordResourceTypeProductionStep, stepID)
			if deletedCheckErr != nil {
				return tracing.Trace(span, deletedCheckErr)
			}
			if wasDeleted {
				return tracing.Trace(span, apierror.NewAlreadyDeletedError("This production step has already been deleted and can no longer be modified."))
			}
		}
		return tracing.Trace(span, apiErr)
	}

	// Disconnect parent-child links, then delete — atomically.
	apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *productionStepSvcImpl) *apierror.APIError {
		repo := txSvc.repos.NewProductionStepRepo()

		if apiErr := repo.DeleteParentChildLinks(txCtx, stepID); apiErr != nil {
			return apiErr
		}

		if apiErr := txSvc.repos.NewDeletedRecordRepo().Create(txCtx, constants.DeletedRecordResourceTypeProductionStep, step.ID, step); apiErr != nil {
			return apiErr
		}

		if apiErr := repo.Delete(txCtx, accountID, stepID); apiErr != nil {
			return apiErr
		}

		changes := audit.ComputeChanges(step, (*domain.ProductionStep)(nil))
		if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
			ServiceName:  domain.ServiceName,
			Action:       constants.AuditActionDelete,
			ResourceType: constants.ObjectTypeProductionStep,
			ResourceID:   step.ID,
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
