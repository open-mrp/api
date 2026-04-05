package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/augno/api/shared/db"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/pagination"
	"github.com/augno/api/shared/tracing"
)

var productionStepRepoTracer = tracing.GetTracer("core-service.production_step_repository")

type productionStepRepoImpl struct {
	queries *sqlc.Queries
}

func NewProductionStepRepo(queries *sqlc.Queries) domain.ProductionStepRepo {
	return &productionStepRepoImpl{queries: queries}
}

func psCreatedAt(ps *domain.ProductionStep) time.Time { return ps.CreatedAt }
func psID(ps *domain.ProductionStep) string           { return ps.ID }

func mapRateFromRow(id, value sql.NullString, numID, numAbbr, numType, denID, denAbbr, denType sql.NullString) *domain.ProductionStepRate {
	if !id.Valid {
		return nil
	}
	return &domain.ProductionStepRate{
		ID:    id.String,
		Value: value.String,
		NumeratorUnit: domain.LightUnit{
			ID:           numID.String,
			Abbreviation: numAbbr.String,
			Type:         numType.String,
		},
		DenominatorUnit: domain.LightUnit{
			ID:           denID.String,
			Abbreviation: denAbbr.String,
			Type:         denType.String,
		},
	}
}

func mapListForwardRow(row sqlc.ListProductionStepsForwardRow) *domain.ProductionStep {
	step := &domain.ProductionStep{
		ID:             row.ID,
		Name:           row.Name,
		Notes:          nullStringToPtr(row.Notes),
		LevelingFactor: row.LevelingFactor,
		Allowances:     row.Allowances,
		DepartmentID:   nullStringToPtr(row.DepartmentID),
		CreatedAt:      row.CreatedAt,
		UpdatedAt:      row.UpdatedAt,
		Production: &domain.Production{
			ID:              row.ProductionID,
			ItemID:          row.ProducedItemID,
			ItemSKU:         row.ProducedItemSku,
			ItemDescription: nullStringToPtr(row.ProducedItemDescription),
			ItemTypeCode:    row.ProducedItemTypeCode,
			Quantity: domain.Quantity{
				ID:               row.ProducedQuantityID,
				Value:            row.ProducedQuantityValue,
				UnitID:           row.ProducedUnitID,
				UnitAbbreviation: row.ProducedUnitAbbreviation,
				UnitType:         row.ProducedUnitType,
			},
			ProductionStepID: row.ID,
			CreatedAt:        row.ProductionCreatedAt,
			UpdatedAt:        row.ProductionUpdatedAt,
		},
		LaborRate: mapRateFromRow(
			row.LaborRateID, row.LaborRateValue,
			row.LaborRateNumUnitID, row.LaborRateNumUnitAbbr, row.LaborRateNumUnitType,
			row.LaborRateDenUnitID, row.LaborRateDenUnitAbbr, row.LaborRateDenUnitType,
		),
		LaborTime: mapRateFromRow(
			row.LaborTimeID, row.LaborTimeValue,
			row.LaborTimeNumUnitID, row.LaborTimeNumUnitAbbr, row.LaborTimeNumUnitType,
			row.LaborTimeDenUnitID, row.LaborTimeDenUnitAbbr, row.LaborTimeDenUnitType,
		),
		OverheadRate: mapRateFromRow(
			row.OverheadRateID, row.OverheadRateValue,
			row.OverheadRateNumUnitID, row.OverheadRateNumUnitAbbr, row.OverheadRateNumUnitType,
			row.OverheadRateDenUnitID, row.OverheadRateDenUnitAbbr, row.OverheadRateDenUnitType,
		),
	}

	if row.ScanningStationID.Valid {
		step.ScanningStation = &domain.LightScanningStation{
			ID:   row.ScanningStationID.String,
			Name: row.ScanningStationName.String,
		}
	}

	return step
}

func mapListBackwardRow(row sqlc.ListProductionStepsBackwardRow) *domain.ProductionStep {
	step := &domain.ProductionStep{
		ID:             row.ID,
		Name:           row.Name,
		Notes:          nullStringToPtr(row.Notes),
		LevelingFactor: row.LevelingFactor,
		Allowances:     row.Allowances,
		DepartmentID:   nullStringToPtr(row.DepartmentID),
		CreatedAt:      row.CreatedAt,
		UpdatedAt:      row.UpdatedAt,
		Production: &domain.Production{
			ID:              row.ProductionID,
			ItemID:          row.ProducedItemID,
			ItemSKU:         row.ProducedItemSku,
			ItemDescription: nullStringToPtr(row.ProducedItemDescription),
			ItemTypeCode:    row.ProducedItemTypeCode,
			Quantity: domain.Quantity{
				ID:               row.ProducedQuantityID,
				Value:            row.ProducedQuantityValue,
				UnitID:           row.ProducedUnitID,
				UnitAbbreviation: row.ProducedUnitAbbreviation,
				UnitType:         row.ProducedUnitType,
			},
			ProductionStepID: row.ID,
			CreatedAt:        row.ProductionCreatedAt,
			UpdatedAt:        row.ProductionUpdatedAt,
		},
		LaborRate: mapRateFromRow(
			row.LaborRateID, row.LaborRateValue,
			row.LaborRateNumUnitID, row.LaborRateNumUnitAbbr, row.LaborRateNumUnitType,
			row.LaborRateDenUnitID, row.LaborRateDenUnitAbbr, row.LaborRateDenUnitType,
		),
		LaborTime: mapRateFromRow(
			row.LaborTimeID, row.LaborTimeValue,
			row.LaborTimeNumUnitID, row.LaborTimeNumUnitAbbr, row.LaborTimeNumUnitType,
			row.LaborTimeDenUnitID, row.LaborTimeDenUnitAbbr, row.LaborTimeDenUnitType,
		),
		OverheadRate: mapRateFromRow(
			row.OverheadRateID, row.OverheadRateValue,
			row.OverheadRateNumUnitID, row.OverheadRateNumUnitAbbr, row.OverheadRateNumUnitType,
			row.OverheadRateDenUnitID, row.OverheadRateDenUnitAbbr, row.OverheadRateDenUnitType,
		),
	}

	if row.ScanningStationID.Valid {
		step.ScanningStation = &domain.LightScanningStation{
			ID:   row.ScanningStationID.String,
			Name: row.ScanningStationName.String,
		}
	}

	return step
}

func mapGetFullRow(row sqlc.GetProductionStepFullRow) *domain.ProductionStep {
	step := &domain.ProductionStep{
		ID:             row.ID,
		Name:           row.Name,
		Notes:          nullStringToPtr(row.Notes),
		LevelingFactor: row.LevelingFactor,
		Allowances:     row.Allowances,
		DepartmentID:   nullStringToPtr(row.DepartmentID),
		CreatedAt:      row.CreatedAt,
		UpdatedAt:      row.UpdatedAt,
		Production: &domain.Production{
			ID:              row.ProductionID,
			ItemID:          row.ProducedItemID,
			ItemSKU:         row.ProducedItemSku,
			ItemDescription: nullStringToPtr(row.ProducedItemDescription),
			ItemTypeCode:    row.ProducedItemTypeCode,
			Quantity: domain.Quantity{
				ID:               row.ProducedQuantityID,
				Value:            row.ProducedQuantityValue,
				UnitID:           row.ProducedUnitID,
				UnitAbbreviation: row.ProducedUnitAbbreviation,
				UnitType:         row.ProducedUnitType,
			},
			ProductionStepID: row.ID,
			CreatedAt:        row.ProductionCreatedAt,
			UpdatedAt:        row.ProductionUpdatedAt,
		},
		LaborRate: mapRateFromRow(
			row.LaborRateID, row.LaborRateValue,
			row.LaborRateNumUnitID, row.LaborRateNumUnitAbbr, row.LaborRateNumUnitType,
			row.LaborRateDenUnitID, row.LaborRateDenUnitAbbr, row.LaborRateDenUnitType,
		),
		LaborTime: mapRateFromRow(
			row.LaborTimeID, row.LaborTimeValue,
			row.LaborTimeNumUnitID, row.LaborTimeNumUnitAbbr, row.LaborTimeNumUnitType,
			row.LaborTimeDenUnitID, row.LaborTimeDenUnitAbbr, row.LaborTimeDenUnitType,
		),
		OverheadRate: mapRateFromRow(
			row.OverheadRateID, row.OverheadRateValue,
			row.OverheadRateNumUnitID, row.OverheadRateNumUnitAbbr, row.OverheadRateNumUnitType,
			row.OverheadRateDenUnitID, row.OverheadRateDenUnitAbbr, row.OverheadRateDenUnitType,
		),
	}

	if row.ScanningStationID.Valid {
		step.ScanningStation = &domain.LightScanningStation{
			ID:   row.ScanningStationID.String,
			Name: row.ScanningStationName.String,
		}
	}

	return step
}

func (r *productionStepRepoImpl) buildListParams(params domain.ListProductionStepsParams) (searchQuery sql.NullString, includeItemFilter, includeMachineFilter, includeScanningStationFilter, includeInputStepFilter, includeOutputStepFilter bool, itemIDs, machineIDs []string, scanningStationIDs []sql.NullString, inputStepIDs, outputStepIDs []string, startDate, endDate sql.NullTime) {
	if params.Query != nil && *params.Query != "" {
		searchQuery = sql.NullString{String: *params.Query + "*", Valid: true}
	}

	includeItemFilter = len(params.ItemIDs) > 0
	itemIDs = params.ItemIDs
	if itemIDs == nil {
		itemIDs = []string{}
	}

	includeMachineFilter = len(params.MachineIDs) > 0
	machineIDs = params.MachineIDs
	if machineIDs == nil {
		machineIDs = []string{}
	}

	includeScanningStationFilter = len(params.ScanningStationIDs) > 0
	scanningStationIDs = make([]sql.NullString, len(params.ScanningStationIDs))
	for i, id := range params.ScanningStationIDs {
		scanningStationIDs[i] = sql.NullString{String: id, Valid: true}
	}
	if len(scanningStationIDs) == 0 {
		scanningStationIDs = []sql.NullString{}
	}

	includeInputStepFilter = len(params.InputStepIDs) > 0
	inputStepIDs = params.InputStepIDs
	if inputStepIDs == nil {
		inputStepIDs = []string{}
	}

	includeOutputStepFilter = len(params.OutputStepIDs) > 0
	outputStepIDs = params.OutputStepIDs
	if outputStepIDs == nil {
		outputStepIDs = []string{}
	}

	if params.StartDate != nil {
		startDate = sql.NullTime{Time: *params.StartDate, Valid: true}
	}
	if params.EndDate != nil {
		endDate = sql.NullTime{Time: *params.EndDate, Valid: true}
	}

	return
}

func (r *productionStepRepoImpl) enrichSteps(ctx context.Context, steps []*domain.ProductionStep) *apierror.APIError {
	for _, step := range steps {
		machines, apiErr := r.GetMachines(ctx, step.ID)
		if apiErr != nil {
			return apiErr
		}
		step.Machines = machines

		inSteps, apiErr := r.GetInputSteps(ctx, step.ID)
		if apiErr != nil {
			return apiErr
		}
		step.InSteps = inSteps

		outSteps, apiErr := r.GetOutputSteps(ctx, step.ID)
		if apiErr != nil {
			return apiErr
		}
		step.OutSteps = outSteps

		consumptionRows, err := r.queries.GetProductionStepConsumptions(ctx, sql.NullString{String: step.ID, Valid: true})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return apiErr
		}
		consumptions := make([]domain.Consumption, 0, len(consumptionRows))
		for _, cr := range consumptionRows {
			c := mapConsumptionFromStepRow(cr, step.ID)
			consumptions = append(consumptions, c)
		}
		step.Consumptions = consumptions
	}
	return nil
}

func mapConsumptionFromStepRow(row sqlc.GetProductionStepConsumptionsRow, productionStepID string) domain.Consumption {
	var instructions *string
	if row.Instructions.Valid {
		instructions = &row.Instructions.String
	}
	var itemDescription *string
	if row.ConsumedItemDescription.Valid {
		itemDescription = &row.ConsumedItemDescription.String
	}
	return domain.Consumption{
		ID:              row.ID,
		ItemID:          row.ConsumedItemID,
		ItemSKU:         row.ConsumedItemSku,
		ItemDescription: itemDescription,
		ItemTypeCode:    row.ConsumedItemTypeCode,
		Quantity: domain.Quantity{
			ID:               row.ConsumptionQuantityID,
			Value:            row.ConsumptionQuantityValue,
			UnitID:           row.ConsumptionUnitID,
			UnitAbbreviation: row.ConsumptionUnitAbbreviation,
			UnitType:         row.ConsumptionUnitType,
		},
		WasteQuantity: domain.Quantity{
			ID:               row.WasteQuantityID,
			Value:            row.WasteQuantityValue,
			UnitID:           row.WasteUnitID,
			UnitAbbreviation: row.WasteUnitAbbreviation,
			UnitType:         row.WasteUnitType,
		},
		Instructions:     instructions,
		ProductionStepID: productionStepID,
		CreatedAt:        row.CreatedAt,
		UpdatedAt:        row.UpdatedAt,
	}
}

func (r *productionStepRepoImpl) List(ctx context.Context, params domain.ListProductionStepsParams) (*domain.ListProductionStepsResult, *apierror.APIError) {
	ctx, span := productionStepRepoTracer.Start(ctx, "repository.production_step.list")
	defer span.End()

	searchQuery, includeItemFilter, includeMachineFilter, includeScanningStationFilter, includeInputStepFilter, includeOutputStepFilter, itemIDs, machineIDs, scanningStationIDs, inputStepIDs, outputStepIDs, startDate, endDate := r.buildListParams(params)

	var cursorDir *pagination.Direction

	if params.Cursor != nil {
		cur, err := pagination.DecodeStringCursor(*params.Cursor)
		if err != nil {
			return nil, apierror.NewValidationError("Invalid pagination cursor.")
		}
		cursorDir = &cur.Direction

		if cur.Direction == pagination.DirectionBackward {
			rows, err := r.queries.ListProductionStepsBackward(ctx, sqlc.ListProductionStepsBackwardParams{
				AccountID:                    params.AccountID,
				SearchQuery:                  searchQuery,
				SearchQuery_2:                searchQuery,
				IncludeItemFilter:            includeItemFilter,
				ItemIds:                      itemIDs,
				IncludeMachineFilter:         includeMachineFilter,
				MachineIds:                   machineIDs,
				IncludeScanningStationFilter: includeScanningStationFilter,
				ScanningStationIds:           scanningStationIDs,
				IncludeInputStepFilter:       includeInputStepFilter,
				InputStepIds:                 inputStepIDs,
				IncludeOutputStepFilter:      includeOutputStepFilter,
				OutputStepIds:                outputStepIDs,
				StartDate:                    startDate,
				EndDate:                      endDate,
				CursorCreatedAt:              cur.OccurredAt,
				CursorID:                     cur.ID,
				Limit:                        params.Limit + 1,
			})
			if apiErr := db.MapSQLError(err); apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			steps := make([]*domain.ProductionStep, len(rows))
			for i, row := range rows {
				steps[i] = mapListBackwardRow(row)
			}
			result, pageInfo := pagination.BuildPageString(steps, params.Limit, cursorDir, psCreatedAt, psID)
			if apiErr := r.enrichSteps(ctx, result); apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			return &domain.ListProductionStepsResult{Steps: result, PageInfo: pageInfo}, nil
		}

		rows, err := r.queries.ListProductionStepsForward(ctx, sqlc.ListProductionStepsForwardParams{
			AccountID:                    params.AccountID,
			SearchQuery:                  searchQuery,
			SearchQuery_2:                searchQuery,
			IncludeItemFilter:            includeItemFilter,
			ItemIds:                      itemIDs,
			IncludeMachineFilter:         includeMachineFilter,
			MachineIds:                   machineIDs,
			IncludeScanningStationFilter: includeScanningStationFilter,
			ScanningStationIds:           scanningStationIDs,
			IncludeInputStepFilter:       includeInputStepFilter,
			InputStepIds:                 inputStepIDs,
			IncludeOutputStepFilter:      includeOutputStepFilter,
			OutputStepIds:                outputStepIDs,
			StartDate:                    startDate,
			EndDate:                      endDate,
			CursorCreatedAt:              sql.NullTime{Time: cur.OccurredAt, Valid: true},
			CursorID:                     sql.NullString{String: cur.ID, Valid: true},
			Limit:                        params.Limit + 1,
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		steps := make([]*domain.ProductionStep, len(rows))
		for i, row := range rows {
			steps[i] = mapListForwardRow(row)
		}
		result, pageInfo := pagination.BuildPageString(steps, params.Limit, cursorDir, psCreatedAt, psID)
		if apiErr := r.enrichSteps(ctx, result); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		return &domain.ListProductionStepsResult{Steps: result, PageInfo: pageInfo}, nil
	}

	// No cursor — first page
	rows, err := r.queries.ListProductionStepsForward(ctx, sqlc.ListProductionStepsForwardParams{
		AccountID:                    params.AccountID,
		SearchQuery:                  searchQuery,
		SearchQuery_2:                searchQuery,
		IncludeItemFilter:            includeItemFilter,
		ItemIds:                      itemIDs,
		IncludeMachineFilter:         includeMachineFilter,
		MachineIds:                   machineIDs,
		IncludeScanningStationFilter: includeScanningStationFilter,
		ScanningStationIds:           scanningStationIDs,
		IncludeInputStepFilter:       includeInputStepFilter,
		InputStepIds:                 inputStepIDs,
		IncludeOutputStepFilter:      includeOutputStepFilter,
		OutputStepIds:                outputStepIDs,
		StartDate:                    startDate,
		EndDate:                      endDate,
		Limit:                        params.Limit + 1,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	steps := make([]*domain.ProductionStep, len(rows))
	for i, row := range rows {
		steps[i] = mapListForwardRow(row)
	}
	result, pageInfo := pagination.BuildPageString(steps, params.Limit, cursorDir, psCreatedAt, psID)
	if apiErr := r.enrichSteps(ctx, result); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return &domain.ListProductionStepsResult{Steps: result, PageInfo: pageInfo}, nil
}

func (r *productionStepRepoImpl) Get(ctx context.Context, accountID, id string) (*domain.ProductionStep, *apierror.APIError) {
	ctx, span := productionStepRepoTracer.Start(ctx, "repository.production_step.get")
	defer span.End()

	row, err := r.queries.GetProductionStepFull(ctx, sqlc.GetProductionStepFullParams{
		ID:        id,
		AccountID: accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	step := mapGetFullRow(row)

	if apiErr := r.enrichSteps(ctx, []*domain.ProductionStep{step}); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return step, nil
}

func (r *productionStepRepoImpl) InsertStep(ctx context.Context, id, name string, notes *string, levelingFactor, allowances, laborRateID, laborTimeID, overheadRateID string, scanningStationID, departmentID *string, accountID string) *apierror.APIError {
	ctx, span := productionStepRepoTracer.Start(ctx, "repository.production_step.insert_step")
	defer span.End()

	err := r.queries.InsertProductionStep(ctx, sqlc.InsertProductionStepParams{
		ID:                id,
		Name:              name,
		Notes:             ptrToNullString(notes),
		LevelingFactor:    levelingFactor,
		Allowances:        allowances,
		LaborRateID:       laborRateID,
		LaborTimeID:       laborTimeID,
		OverheadRateID:    overheadRateID,
		ScanningStationID: ptrToNullString(scanningStationID),
		DepartmentID:      ptrToNullString(departmentID),
		AccountID:         accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *productionStepRepoImpl) Update(ctx context.Context, params domain.UpdateProductionStepParams) *apierror.APIError {
	ctx, span := productionStepRepoTracer.Start(ctx, "repository.production_step.update")
	defer span.End()

	result, err := r.queries.UpdateProductionStepFields(ctx, sqlc.UpdateProductionStepFieldsParams{
		ID:                    params.ProductionStepID,
		AccountID:             params.AccountID,
		Name:                  ptrToNullString(params.Name),
		LevelingFactor:        ptrToNullString(params.LevelingFactor),
		Allowances:            ptrToNullString(params.Allowances),
		UpdateScanningStation: params.ScanningStationID != nil,
		ScanningStationID:     ptrToNullString(params.ScanningStationID),
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return tracing.Trace(span, apierror.NewInternalError(err, "Failed to check rows affected."))
	}
	if rowsAffected == 0 {
		return tracing.Trace(span, apierror.NewResourceNotFoundError("Production step not found."))
	}

	return nil
}

func (r *productionStepRepoImpl) Delete(ctx context.Context, accountID, id string) *apierror.APIError {
	ctx, span := productionStepRepoTracer.Start(ctx, "repository.production_step.delete")
	defer span.End()

	result, err := r.queries.DeleteProductionStepRow(ctx, sqlc.DeleteProductionStepRowParams{
		ID:        id,
		AccountID: accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return tracing.Trace(span, apierror.NewInternalError(err, "Failed to check rows affected."))
	}
	if rowsAffected == 0 {
		return tracing.Trace(span, apierror.NewResourceNotFoundError("Production step not found."))
	}

	return nil
}

func (r *productionStepRepoImpl) ExistsByName(ctx context.Context, accountID, name string, excludeID *string) (bool, *apierror.APIError) {
	ctx, span := productionStepRepoTracer.Start(ctx, "repository.production_step.exists_by_name")
	defer span.End()

	count, err := r.queries.ExistsProductionStepByName(ctx, sqlc.ExistsProductionStepByNameParams{
		Name:      name,
		AccountID: accountID,
		ExcludeID: ptrToNullString(excludeID),
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return false, tracing.Trace(span, apiErr)
	}
	return count > 0, nil
}

func (r *productionStepRepoImpl) DeleteParentChildLinks(ctx context.Context, id string) *apierror.APIError {
	ctx, span := productionStepRepoTracer.Start(ctx, "repository.production_step.delete_parent_child_links")
	defer span.End()

	err := r.queries.DeleteProductionStepParentChildLinks(ctx, sqlc.DeleteProductionStepParentChildLinksParams{
		StepID: id,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *productionStepRepoImpl) GetInputSteps(ctx context.Context, id string) ([]domain.LightProductionStep, *apierror.APIError) {
	ctx, span := productionStepRepoTracer.Start(ctx, "repository.production_step.get_input_steps")
	defer span.End()

	rows, err := r.queries.GetProductionStepInputSteps(ctx, id)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	steps := make([]domain.LightProductionStep, len(rows))
	for i, row := range rows {
		steps[i] = domain.LightProductionStep{ID: row.ID, Name: row.Name}
	}
	return steps, nil
}

func (r *productionStepRepoImpl) GetOutputSteps(ctx context.Context, id string) ([]domain.LightProductionStep, *apierror.APIError) {
	ctx, span := productionStepRepoTracer.Start(ctx, "repository.production_step.get_output_steps")
	defer span.End()

	rows, err := r.queries.GetProductionStepOutputSteps(ctx, id)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	steps := make([]domain.LightProductionStep, len(rows))
	for i, row := range rows {
		steps[i] = domain.LightProductionStep{ID: row.ID, Name: row.Name}
	}
	return steps, nil
}

func (r *productionStepRepoImpl) GetMachines(ctx context.Context, id string) ([]domain.LightMachine, *apierror.APIError) {
	ctx, span := productionStepRepoTracer.Start(ctx, "repository.production_step.get_machines")
	defer span.End()

	rows, err := r.queries.GetProductionStepMachines(ctx, sql.NullString{String: id, Valid: true})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	machines := make([]domain.LightMachine, len(rows))
	for i, row := range rows {
		machines[i] = domain.LightMachine{ID: row.ID, Name: row.Name}
	}
	return machines, nil
}

func (r *productionStepRepoImpl) FindIDByName(ctx context.Context, accountID, name string) (*string, *apierror.APIError) {
	ctx, span := productionStepRepoTracer.Start(ctx, "repository.production_step.find_id_by_name")
	defer span.End()

	id, err := r.queries.FindProductionStepIDByName(ctx, sqlc.FindProductionStepIDByNameParams{
		Name:      name,
		AccountID: accountID,
	})
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}
	return &id, nil
}

func (r *productionStepRepoImpl) DeleteConsumptionsByStepID(ctx context.Context, stepID string) *apierror.APIError {
	ctx, span := productionStepRepoTracer.Start(ctx, "repository.production_step.delete_consumptions_by_step_id")
	defer span.End()

	stepIDArg := sql.NullString{String: stepID, Valid: stepID != ""}

	// Delete associated quantity records first.
	err := r.queries.DeleteConsumptionQuantitiesByStepID(ctx, stepIDArg)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	// Delete consumption rows.
	err = r.queries.DeleteConsumptionsByStepID(ctx, stepIDArg)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *productionStepRepoImpl) DeleteProductionsByStepID(ctx context.Context, stepID string) *apierror.APIError {
	ctx, span := productionStepRepoTracer.Start(ctx, "repository.production_step.delete_productions_by_step_id")
	defer span.End()

	stepIDArg := sql.NullString{String: stepID, Valid: stepID != ""}

	// Delete associated quantity records first.
	err := r.queries.DeleteProductionQuantitiesByStepID(ctx, stepIDArg)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	// Delete production rows.
	err = r.queries.DeleteProductionsByStepID(ctx, stepIDArg)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *productionStepRepoImpl) UpdateStepFull(ctx context.Context, id, accountID, levelingFactor, allowances string, scanningStationID *string) *apierror.APIError {
	ctx, span := productionStepRepoTracer.Start(ctx, "repository.production_step.update_step_full")
	defer span.End()

	err := r.queries.UpdateProductionStepFull(ctx, sqlc.UpdateProductionStepFullParams{
		ID:                id,
		AccountID:         accountID,
		LevelingFactor:    levelingFactor,
		Allowances:        allowances,
		ScanningStationID: ptrToNullString(scanningStationID),
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *productionStepRepoImpl) GetRate(ctx context.Context, rateID string) (*domain.ProductionStepRate, *apierror.APIError) {
	// Rates are fetched as part of the full production step query — this is a fallback.
	return nil, apierror.NewInternalError(nil, "GetRate not implemented as a standalone query; rates are fetched inline.")
}

func (r *productionStepRepoImpl) InsertRate(ctx context.Context, id string, params domain.CreateRateParams) *apierror.APIError {
	ctx, span := productionStepRepoTracer.Start(ctx, "repository.production_step.insert_rate")
	defer span.End()

	err := r.queries.InsertRateForProductionStep(ctx, sqlc.InsertRateForProductionStepParams{
		ID:                id,
		Value:             params.Value,
		NumeratorUnitID:   params.NumeratorUnitID,
		DenominatorUnitID: params.DenominatorUnitID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *productionStepRepoImpl) InsertQuantity(ctx context.Context, id, value, unitID string) *apierror.APIError {
	ctx, span := productionStepRepoTracer.Start(ctx, "repository.production_step.insert_quantity")
	defer span.End()

	err := r.queries.InsertProductionQuantity(ctx, sqlc.InsertProductionQuantityParams{
		ID:     id,
		Value:  value,
		UnitID: unitID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *productionStepRepoImpl) InsertProduction(ctx context.Context, id, itemID, quantityID, productionStepID string) *apierror.APIError {
	ctx, span := productionStepRepoTracer.Start(ctx, "repository.production_step.insert_production")
	defer span.End()

	err := r.queries.InsertProduction(ctx, sqlc.InsertProductionParams{
		ID:               id,
		ItemID:           itemID,
		QuantityID:       quantityID,
		ProductionStepID: sql.NullString{String: productionStepID, Valid: true},
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}
