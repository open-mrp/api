package repository

import (
	"context"
	gosql "database/sql"
	"errors"
	"sort"
	"time"

	"github.com/shopspring/decimal"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/augno/api/shared/db"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/id"
	"github.com/augno/api/shared/pagination"
	"github.com/augno/api/shared/tracing"
)

var batchRepoTracer = tracing.GetTracer("core-service.batch_repository")

type batchRepoImpl struct {
	queries *sqlc.Queries
}

func NewBatchRepo(queries *sqlc.Queries) domain.BatchRepo {
	return &batchRepoImpl{queries: queries}
}

func batchScannedAt(b *domain.Batch) time.Time {
	if b.ScannedAt != nil {
		return *b.ScannedAt
	}
	return b.CreatedAt
}

func batchID(b *domain.Batch) string { return b.ID }

// mapBatchRow converts a GetBatchRow into a domain.Batch.
func mapBatchRow(row sqlc.GetBatchRow) *domain.Batch {
	quantity, _ := decimal.NewFromString(row.QuantityValue)

	b := &domain.Batch{
		ID: row.ID,
		Item: domain.LightItem{
			ID:  row.ItemID,
			SKU: row.ItemSku,
		},
		Quantity: domain.BatchQuantity{
			ID:      row.QuantityID,
			Measure: quantity,
			Unit: domain.LightUnit{
				ID:           row.QuantityUnitID,
				Abbreviation: row.QuantityUnitAbbreviation,
				Type:         row.QuantityUnitType,
			},
		},
		ClosedAt:  db.TimeFromNullTime(row.ClosedAt),
		ScannedAt: db.TimeFromNullTime(row.ScannedAt),
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}

	if row.SecondsQuantityID.Valid {
		secMeasure, _ := decimal.NewFromString(row.SecondsQuantityValue.String)
		b.Seconds = &domain.BatchQuantity{
			ID:      row.SecondsQuantityID.String,
			Measure: secMeasure,
			Unit: domain.LightUnit{
				ID:           row.SecondsUnitID.String,
				Abbreviation: row.SecondsUnitAbbreviation.String,
				Type:         row.SecondsUnitType.String,
			},
		}
	}

	if row.WasteQuantityID.Valid {
		wasteMeasure, _ := decimal.NewFromString(row.WasteQuantityValue.String)
		b.Waste = &domain.BatchQuantity{
			ID:      row.WasteQuantityID.String,
			Measure: wasteMeasure,
			Unit: domain.LightUnit{
				ID:           row.WasteUnitID.String,
				Abbreviation: row.WasteUnitAbbreviation.String,
				Type:         row.WasteUnitType.String,
			},
		}
	}

	if row.ScanningStationID.Valid {
		b.ScanningStation = &domain.LightScanningStation{
			ID:   row.ScanningStationID.String,
			Name: row.ScanningStationName.String,
		}
	}

	if row.DepartmentID.Valid {
		b.DepartmentID = &row.DepartmentID.String
	}
	if row.DepartmentName.Valid {
		b.DepartmentName = &row.DepartmentName.String
	}

	if row.ProductionStepID.Valid {
		b.ProductionStep = &domain.LightProductionStep{
			ID:   row.ProductionStepID.String,
			Name: row.ProductionStepName.String,
		}
	}

	if row.ProductionRunID2.Valid {
		b.ProductionRun = &domain.LightProductionRun{
			ID:     row.ProductionRunID2.String,
			Number: row.ProductionRunNumber.String,
		}
	}

	return b
}

// mapBaseBatchRow converts a GetBatchBaseRow into a domain.BaseBatch.
func mapBaseBatchRow(row sqlc.GetBatchBaseRow) *domain.BaseBatch {
	quantity, _ := decimal.NewFromString(row.QuantityValue)

	b := &domain.BaseBatch{
		ID: row.ID,
		Item: domain.LightItem{
			ID:  row.ItemID,
			SKU: row.ItemSku,
		},
		Quantity: domain.BatchQuantity{
			ID:      row.QuantityID,
			Measure: quantity,
			Unit: domain.LightUnit{
				ID:           row.QuantityUnitID,
				Abbreviation: row.QuantityUnitAbbreviation,
				Type:         row.QuantityUnitType,
			},
		},
		ProductionRunID: db.StringFromNullString(row.ProductionRunID),
		ClosedAt:        db.TimeFromNullTime(row.ClosedAt),
		ScannedAt:       db.TimeFromNullTime(row.ScannedAt),
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
	}

	if row.SecondsQuantityID.Valid {
		secMeasure, _ := decimal.NewFromString(row.SecondsQuantityValue.String)
		b.Seconds = &domain.BatchQuantity{
			ID:      row.SecondsQuantityID.String,
			Measure: secMeasure,
			Unit: domain.LightUnit{
				ID:           row.SecondsUnitID.String,
				Abbreviation: row.SecondsUnitAbbreviation.String,
				Type:         row.SecondsUnitType.String,
			},
		}
	}

	if row.WasteQuantityID.Valid {
		wasteMeasure, _ := decimal.NewFromString(row.WasteQuantityValue.String)
		b.Waste = &domain.BatchQuantity{
			ID:      row.WasteQuantityID.String,
			Measure: wasteMeasure,
			Unit: domain.LightUnit{
				ID:           row.WasteUnitID.String,
				Abbreviation: row.WasteUnitAbbreviation.String,
				Type:         row.WasteUnitType.String,
			},
		}
	}

	if row.ScanningStationID.Valid {
		b.ScanningStation = &domain.LightScanningStation{
			ID:   row.ScanningStationID.String,
			Name: row.ScanningStationName.String,
		}
	}

	if row.DepartmentID.Valid {
		b.DepartmentID = &row.DepartmentID.String
	}
	if row.DepartmentName.Valid {
		b.DepartmentName = &row.DepartmentName.String
	}

	if row.ProductionStepID.Valid {
		b.ProductionStep = &domain.LightProductionStep{
			ID:   row.ProductionStepID.String,
			Name: row.ProductionStepName.String,
		}
	}

	if row.ProductionRunID2.Valid {
		b.ProductionRun = &domain.LightProductionRun{
			ID:     row.ProductionRunID2.String,
			Number: row.ProductionRunNumber.String,
		}
	}

	return b
}

// mapForwardBatchRow converts a ListBatchesByScanningStationForwardRow into a domain.Batch.
func mapForwardBatchRow(row sqlc.ListBatchesByScanningStationForwardRow) *domain.Batch {
	quantity, _ := decimal.NewFromString(row.QuantityValue)

	b := &domain.Batch{
		ID: row.ID,
		Item: domain.LightItem{
			ID:  row.ItemID,
			SKU: row.ItemSku,
		},
		Quantity: domain.BatchQuantity{
			ID:      row.QuantityID,
			Measure: quantity,
			Unit: domain.LightUnit{
				ID:           row.QuantityUnitID,
				Abbreviation: row.QuantityUnitAbbreviation,
				Type:         row.QuantityUnitType,
			},
		},
		ClosedAt:  db.TimeFromNullTime(row.ClosedAt),
		ScannedAt: db.TimeFromNullTime(row.ScannedAt),
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}

	if row.SecondsQuantityID.Valid {
		secMeasure, _ := decimal.NewFromString(row.SecondsQuantityValue.String)
		b.Seconds = &domain.BatchQuantity{
			ID:      row.SecondsQuantityID.String,
			Measure: secMeasure,
			Unit: domain.LightUnit{
				ID:           row.SecondsUnitID.String,
				Abbreviation: row.SecondsUnitAbbreviation.String,
				Type:         row.SecondsUnitType.String,
			},
		}
	}

	if row.WasteQuantityID.Valid {
		wasteMeasure, _ := decimal.NewFromString(row.WasteQuantityValue.String)
		b.Waste = &domain.BatchQuantity{
			ID:      row.WasteQuantityID.String,
			Measure: wasteMeasure,
			Unit: domain.LightUnit{
				ID:           row.WasteUnitID.String,
				Abbreviation: row.WasteUnitAbbreviation.String,
				Type:         row.WasteUnitType.String,
			},
		}
	}

	if row.ScanningStationID.Valid {
		b.ScanningStation = &domain.LightScanningStation{
			ID:   row.ScanningStationID.String,
			Name: row.ScanningStationName.String,
		}
	}

	if row.ProductionStepID.Valid {
		b.ProductionStep = &domain.LightProductionStep{
			ID:   row.ProductionStepID.String,
			Name: row.ProductionStepName.String,
		}
	}

	if row.ProductionRunID2.Valid {
		b.ProductionRun = &domain.LightProductionRun{
			ID:     row.ProductionRunID2.String,
			Number: row.ProductionRunNumber.String,
		}
	}

	return b
}

// mapBackwardBatchRow converts a ListBatchesByScanningStationBackwardRow into a domain.Batch.
func mapBackwardBatchRow(row sqlc.ListBatchesByScanningStationBackwardRow) *domain.Batch {
	quantity, _ := decimal.NewFromString(row.QuantityValue)

	b := &domain.Batch{
		ID: row.ID,
		Item: domain.LightItem{
			ID:  row.ItemID,
			SKU: row.ItemSku,
		},
		Quantity: domain.BatchQuantity{
			ID:      row.QuantityID,
			Measure: quantity,
			Unit: domain.LightUnit{
				ID:           row.QuantityUnitID,
				Abbreviation: row.QuantityUnitAbbreviation,
				Type:         row.QuantityUnitType,
			},
		},
		ClosedAt:  db.TimeFromNullTime(row.ClosedAt),
		ScannedAt: db.TimeFromNullTime(row.ScannedAt),
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}

	if row.SecondsQuantityID.Valid {
		secMeasure, _ := decimal.NewFromString(row.SecondsQuantityValue.String)
		b.Seconds = &domain.BatchQuantity{
			ID:      row.SecondsQuantityID.String,
			Measure: secMeasure,
			Unit: domain.LightUnit{
				ID:           row.SecondsUnitID.String,
				Abbreviation: row.SecondsUnitAbbreviation.String,
				Type:         row.SecondsUnitType.String,
			},
		}
	}

	if row.WasteQuantityID.Valid {
		wasteMeasure, _ := decimal.NewFromString(row.WasteQuantityValue.String)
		b.Waste = &domain.BatchQuantity{
			ID:      row.WasteQuantityID.String,
			Measure: wasteMeasure,
			Unit: domain.LightUnit{
				ID:           row.WasteUnitID.String,
				Abbreviation: row.WasteUnitAbbreviation.String,
				Type:         row.WasteUnitType.String,
			},
		}
	}

	if row.ScanningStationID.Valid {
		b.ScanningStation = &domain.LightScanningStation{
			ID:   row.ScanningStationID.String,
			Name: row.ScanningStationName.String,
		}
	}

	if row.ProductionStepID.Valid {
		b.ProductionStep = &domain.LightProductionStep{
			ID:   row.ProductionStepID.String,
			Name: row.ProductionStepName.String,
		}
	}

	if row.ProductionRunID2.Valid {
		b.ProductionRun = &domain.LightProductionRun{
			ID:     row.ProductionRunID2.String,
			Number: row.ProductionRunNumber.String,
		}
	}

	return b
}

func (r *batchRepoImpl) Find(ctx context.Context, accountID, batchID string) (*domain.Batch, *apierror.APIError) {
	ctx, span := batchRepoTracer.Start(ctx, "repository.batch.find")
	defer span.End()

	row, err := r.queries.GetBatch(ctx, sqlc.GetBatchParams{
		ID:        batchID,
		AccountID: accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	batch := mapBatchRow(row)

	machineRows, err := r.queries.GetBatchMachines(ctx, batchID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	machines := make([]domain.LightMachine, len(machineRows))
	for i, m := range machineRows {
		machines[i] = domain.LightMachine{
			ID:   m.ID,
			Name: m.Name,
		}
	}
	batch.Machines = machines

	return batch, nil
}

func (r *batchRepoImpl) FindBatchFlow(ctx context.Context, accountID, batchID string) ([]domain.BatchFlowNode, *apierror.APIError) {
	ctx, span := batchRepoTracer.Start(ctx, "repository.batch.find_batch_flow")
	defer span.End()

	// BFS to collect all batch IDs in the flow graph.
	visited := map[string]bool{batchID: true}
	queue := []string{batchID}

	// outgoing and incoming adjacency maps for building nodes later.
	outgoingMap := make(map[string][]string)
	incomingMap := make(map[string][]string)

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		outgoing, err := r.queries.GetBatchFlowOutgoing(ctx, current)
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		outgoingMap[current] = outgoing

		incoming, err := r.queries.GetBatchFlowIncoming(ctx, current)
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		incomingMap[current] = incoming

		for _, neighbor := range outgoing {
			if !visited[neighbor] {
				visited[neighbor] = true
				queue = append(queue, neighbor)
			}
		}
		for _, neighbor := range incoming {
			if !visited[neighbor] {
				visited[neighbor] = true
				queue = append(queue, neighbor)
			}
		}
	}

	// For each visited batch, fetch full data and build the flow node.
	nodes := make([]domain.BatchFlowNode, 0, len(visited))
	for id := range visited {
		row, err := r.queries.GetBatch(ctx, sqlc.GetBatchParams{
			ID:        id,
			AccountID: accountID,
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}

		batch := mapBatchRow(row)

		machineRows, err := r.queries.GetBatchMachines(ctx, id)
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		machines := make([]domain.LightMachine, len(machineRows))
		for i, m := range machineRows {
			machines[i] = domain.LightMachine{ID: m.ID, Name: m.Name}
		}
		batch.Machines = machines

		out := outgoingMap[id]
		if out == nil {
			out = []string{}
		}
		in := incomingMap[id]
		if in == nil {
			in = []string{}
		}

		nodes = append(nodes, domain.BatchFlowNode{
			Batch:          *batch,
			InputBatchIDs:  in,
			OutputBatchIDs: out,
		})
	}

	return nodes, nil
}

func (r *batchRepoImpl) FindByScanningStation(ctx context.Context, params domain.ListBatchesByScanningStationParams) (*domain.ListBatchesByScanningStationResult, *apierror.APIError) {
	ctx, span := batchRepoTracer.Start(ctx, "repository.batch.find_by_scanning_station")
	defer span.End()

	searchQuery := gosql.NullString{}
	if params.Query != nil && *params.Query != "" {
		searchQuery = gosql.NullString{String: "%" + db.EscapeLike(*params.Query) + "%", Valid: true}
	}

	scanningStationID := db.NullString(params.ScanningStationID)

	var cursorDir *pagination.Direction

	if params.Cursor != nil {
		cur, err := pagination.DecodeStringCursor(*params.Cursor)
		if err != nil {
			return nil, apierror.NewValidationError("Invalid pagination cursor.")
		}
		cursorDir = &cur.Direction

		if cur.Direction == pagination.DirectionBackward {
			rows, err := r.queries.ListBatchesByScanningStationBackward(ctx, sqlc.ListBatchesByScanningStationBackwardParams{
				AccountID:         params.AccountID,
				ScanningStationID: scanningStationID,
				SearchQuery:       searchQuery,
				CursorScannedAt:   gosql.NullTime{Time: cur.OccurredAt, Valid: true},
				CursorID:          cur.ID,
				Limit:             params.Limit + 1,
			})
			if apiErr := db.MapSQLError(err); apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			batches := make([]*domain.Batch, len(rows))
			for i, row := range rows {
				batches[i] = mapBackwardBatchRow(row)
			}
			result, pageInfo := pagination.BuildPageString(batches, params.Limit, cursorDir, batchScannedAt, batchID)
			return &domain.ListBatchesByScanningStationResult{Batches: result, PageInfo: pageInfo}, nil
		}

		// Forward with cursor
		rows, err := r.queries.ListBatchesByScanningStationForward(ctx, sqlc.ListBatchesByScanningStationForwardParams{
			AccountID:         params.AccountID,
			ScanningStationID: scanningStationID,
			SearchQuery:       searchQuery,
			CursorScannedAt:   gosql.NullTime{Time: cur.OccurredAt, Valid: true},
			CursorID:          gosql.NullString{String: cur.ID, Valid: true},
			Limit:             params.Limit + 1,
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		batches := make([]*domain.Batch, len(rows))
		for i, row := range rows {
			batches[i] = mapForwardBatchRow(row)
		}
		result, pageInfo := pagination.BuildPageString(batches, params.Limit, cursorDir, batchScannedAt, batchID)
		return &domain.ListBatchesByScanningStationResult{Batches: result, PageInfo: pageInfo}, nil
	}

	// No cursor — first page
	rows, err := r.queries.ListBatchesByScanningStationForward(ctx, sqlc.ListBatchesByScanningStationForwardParams{
		AccountID:         params.AccountID,
		ScanningStationID: scanningStationID,
		SearchQuery:       searchQuery,
		Limit:             params.Limit + 1,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	batches := make([]*domain.Batch, len(rows))
	for i, row := range rows {
		batches[i] = mapForwardBatchRow(row)
	}
	result, pageInfo := pagination.BuildPageString(batches, params.Limit, cursorDir, batchScannedAt, batchID)
	return &domain.ListBatchesByScanningStationResult{Batches: result, PageInfo: pageInfo}, nil
}

func (r *batchRepoImpl) FindPossibleNextSteps(ctx context.Context, accountID, scanningStationID, batchID string) ([]domain.ScanningProductionStepInfo, *apierror.APIError) {
	ctx, span := batchRepoTracer.Start(ctx, "repository.batch.find_possible_next_steps")
	defer span.End()

	// Get the starting batch.
	startBatch, err := r.queries.GetBatchFlowTraversalInfo(ctx, sqlc.GetBatchFlowTraversalInfoParams{
		ID:        batchID,
		AccountID: accountID,
	})
	if err != nil {
		if errors.Is(err, gosql.ErrNoRows) {
			return []domain.ScanningProductionStepInfo{}, nil
		}
		return nil, tracing.Trace(span, db.MapSQLError(err))
	}

	// BFS forward through batch flow, matching the Dashboard algorithm:
	// - Unclosed batches that are scanned with a production step: collect child production steps.
	// - Closed batches: follow outgoing batch flow edges and continue traversal.
	type traversalBatch struct {
		id               string
		closedAt         gosql.NullTime
		scannedAt        gosql.NullTime
		productionStepID gosql.NullString
	}

	currentBatches := []traversalBatch{{
		id:               startBatch.ID,
		closedAt:         startBatch.ClosedAt,
		scannedAt:        startBatch.ScannedAt,
		productionStepID: startBatch.ProductionStepID,
	}}

	// Use a set to deduplicate results by step ID.
	seenSteps := map[string]bool{}
	var results []domain.ScanningProductionStepInfo

	for len(currentBatches) > 0 {
		var nextBatches []traversalBatch

		for _, batch := range currentBatches {
			if !batch.closedAt.Valid {
				// Unclosed batch: check if scanned and has a production step.
				if !batch.scannedAt.Valid || !batch.productionStepID.Valid {
					continue
				}

				// Get child production steps of this batch's production step.
				childStepIDs, err := r.queries.GetProductionStepChildSteps(ctx, batch.productionStepID.String)
				if apiErr := db.MapSQLError(err); apiErr != nil {
					return nil, tracing.Trace(span, apiErr)
				}

				for _, childStepID := range childStepIDs {
					if seenSteps[childStepID] {
						continue
					}

					// Check if the child step's scanning station matches.
					stationID, err := r.queries.GetProductionStepScanningStationID(ctx, sqlc.GetProductionStepScanningStationIDParams{
						ID:        childStepID,
						AccountID: accountID,
					})
					if apiErr := db.MapSQLError(err); apiErr != nil {
						return nil, tracing.Trace(span, apiErr)
					}

					if !stationID.Valid || stationID.String != scanningStationID {
						continue
					}

					// Get step name.
					stepRow, err := r.queries.GetProductionStep(ctx, sqlc.GetProductionStepParams{
						ID:        childStepID,
						AccountID: accountID,
					})
					if apiErr := db.MapSQLError(err); apiErr != nil {
						return nil, tracing.Trace(span, apiErr)
					}

					// Count only part-type consumptions for isMultiPart.
					consumptionCount, err := r.queries.CountProductionStepPartConsumptions(ctx, gosql.NullString{String: childStepID, Valid: true})
					if apiErr := db.MapSQLError(err); apiErr != nil {
						return nil, tracing.Trace(span, apiErr)
					}

					seenSteps[childStepID] = true
					results = append(results, domain.ScanningProductionStepInfo{
						ID:          childStepID,
						Name:        stepRow.Name,
						IsMultiPart: consumptionCount > 1,
					})
				}
			} else {
				// Closed batch: follow outgoing batch flow edges.
				outgoing, err := r.queries.GetBatchFlowOutgoing(ctx, batch.id)
				if apiErr := db.MapSQLError(err); apiErr != nil {
					return nil, tracing.Trace(span, apiErr)
				}

				for _, outBatchID := range outgoing {
					outBatch, err := r.queries.GetBatchFlowTraversalInfo(ctx, sqlc.GetBatchFlowTraversalInfoParams{
						ID:        outBatchID,
						AccountID: accountID,
					})
					if err != nil {
						if errors.Is(err, gosql.ErrNoRows) {
							continue
						}
						return nil, tracing.Trace(span, db.MapSQLError(err))
					}

					nextBatches = append(nextBatches, traversalBatch{
						id:               outBatch.ID,
						closedAt:         outBatch.ClosedAt,
						scannedAt:        outBatch.ScannedAt,
						productionStepID: outBatch.ProductionStepID,
					})
				}
			}
		}

		currentBatches = nextBatches
	}

	if results == nil {
		results = []domain.ScanningProductionStepInfo{}
	}

	return results, nil
}

func (r *batchRepoImpl) FindOpenBatches(ctx context.Context, accountID string, itemIDs, productLineIDs []string) ([]domain.OpenBatchSummary, *apierror.APIError) {
	ctx, span := batchRepoTracer.Start(ctx, "repository.batch.find_open_batches")
	defer span.End()

	includeItemFilter := len(itemIDs) > 0
	if itemIDs == nil {
		itemIDs = []string{}
	}
	includeProductLineFilter := len(productLineIDs) > 0
	// The product_line_id column is nullable, so sqlc types the filter slice as NullString.
	productLineFilter := make([]gosql.NullString, len(productLineIDs))
	for i, id := range productLineIDs {
		productLineFilter[i] = gosql.NullString{String: id, Valid: true}
	}

	rows, err := r.queries.ListOpenBatches(ctx, sqlc.ListOpenBatchesParams{
		AccountID:                accountID,
		IncludeItemFilter:        includeItemFilter,
		ItemIds:                  itemIDs,
		IncludeProductLineFilter: includeProductLineFilter,
		ProductLineIds:           productLineFilter,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	summaries := make([]domain.OpenBatchSummary, len(rows))
	for i, row := range rows {
		departmentName := ""
		if row.DepartmentName.Valid {
			departmentName = row.DepartmentName.String
		}

		scanningStationID := ""
		if row.ScanningStationID.Valid {
			scanningStationID = row.ScanningStationID.String
		}

		totalCount := decimal.Zero
		if row.TotalCount != nil {
			if tc, ok := row.TotalCount.(string); ok {
				totalCount, _ = decimal.NewFromString(tc)
			}
		}

		summaries[i] = domain.OpenBatchSummary{
			DepartmentName:    departmentName,
			ItemName:          row.ItemName,
			ItemID:            row.ItemID,
			ScanningStationID: scanningStationID,
			Count:             totalCount,
			Unit:              row.UnitAbbreviation,
		}
	}

	return summaries, nil
}

func (r *batchRepoImpl) FindFurthestRightBatchInFlow(ctx context.Context, accountID, batchID string) (*domain.BaseBatch, *apierror.APIError) {
	ctx, span := batchRepoTracer.Start(ctx, "repository.batch.find_furthest_right_batch_in_flow")
	defer span.End()

	// BFS to collect all batch IDs in the flow.
	visited := map[string]bool{batchID: true}
	queue := []string{batchID}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		outgoing, err := r.queries.GetBatchFlowOutgoing(ctx, current)
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}

		for _, neighbor := range outgoing {
			if !visited[neighbor] {
				visited[neighbor] = true
				queue = append(queue, neighbor)
			}
		}

		incoming, err := r.queries.GetBatchFlowIncoming(ctx, current)
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}

		for _, neighbor := range incoming {
			if !visited[neighbor] {
				visited[neighbor] = true
				queue = append(queue, neighbor)
			}
		}
	}

	// Get all batches and filter to scanned && !closed, sort by scannedAt desc.
	type candidate struct {
		batch     *domain.BaseBatch
		scannedAt time.Time
	}
	var candidates []candidate

	for id := range visited {
		row, err := r.queries.GetBatchBase(ctx, sqlc.GetBatchBaseParams{
			ID:        id,
			AccountID: accountID,
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}

		b := mapBaseBatchRow(row)
		if b.ScannedAt != nil && b.ClosedAt == nil {
			candidates = append(candidates, candidate{batch: b, scannedAt: *b.ScannedAt})
		}
	}

	if len(candidates) == 0 {
		// Fall back to the original batch itself.
		row, err := r.queries.GetBatchBase(ctx, sqlc.GetBatchBaseParams{
			ID:        batchID,
			AccountID: accountID,
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		return mapBaseBatchRow(row), nil
	}

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].scannedAt.After(candidates[j].scannedAt)
	})

	return candidates[0].batch, nil
}

func (r *batchRepoImpl) FindNextAvailableBatchInFlow(ctx context.Context, accountID, batchID, productionStepID string) (*domain.BaseBatch, *apierror.APIError) {
	ctx, span := batchRepoTracer.Start(ctx, "repository.batch.find_next_available_batch_in_flow")
	defer span.End()

	// BFS through the flow graph following outgoing edges.
	visited := map[string]bool{batchID: true}
	queue := []string{batchID}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		row, err := r.queries.GetBatchBase(ctx, sqlc.GetBatchBaseParams{
			ID:        current,
			AccountID: accountID,
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}

		b := mapBaseBatchRow(row)

		// If the batch is scanned, not closed, and its production step is an input of the target step, return it.
		if b.ScannedAt != nil && b.ClosedAt == nil && b.ProductionStep != nil {
			count, err := r.queries.IsInputOfProductionStep(ctx, sqlc.IsInputOfProductionStepParams{
				CurrentStepID: productionStepID,
				InputStepID:   b.ProductionStep.ID,
			})
			if apiErr := db.MapSQLError(err); apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			if count > 0 {
				return b, nil
			}
		}

		// If closed, continue BFS through output batches.
		if b.ClosedAt != nil {
			outgoing, err := r.queries.GetBatchFlowOutgoing(ctx, current)
			if apiErr := db.MapSQLError(err); apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			for _, neighbor := range outgoing {
				if !visited[neighbor] {
					visited[neighbor] = true
					queue = append(queue, neighbor)
				}
			}
		}
	}

	return nil, tracing.Trace(span, apierror.NewResourceNotFoundError("No available batch found in the flow."))
}

func (r *batchRepoImpl) FindAvailableBatchesInFlow(ctx context.Context, accountID string, batchIDs []string, productionStepID string) ([]domain.BaseBatch, *apierror.APIError) {
	ctx, span := batchRepoTracer.Start(ctx, "repository.batch.find_available_batches_in_flow")
	defer span.End()

	var results []domain.BaseBatch
	for _, bid := range batchIDs {
		b, apiErr := r.FindNextAvailableBatchInFlow(ctx, accountID, bid, productionStepID)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		results = append(results, *b)
	}

	if results == nil {
		results = []domain.BaseBatch{}
	}

	return results, nil
}

func (r *batchRepoImpl) FindOutputBatches(ctx context.Context, accountID, batchID string) ([]domain.BaseBatch, *apierror.APIError) {
	ctx, span := batchRepoTracer.Start(ctx, "repository.batch.find_output_batches")
	defer span.End()

	outgoingIDs, err := r.queries.GetBatchFlowOutgoing(ctx, batchID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	results := make([]domain.BaseBatch, 0, len(outgoingIDs))
	for _, outID := range outgoingIDs {
		row, err := r.queries.GetBatchBase(ctx, sqlc.GetBatchBaseParams{
			ID:        outID,
			AccountID: accountID,
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		results = append(results, *mapBaseBatchRow(row))
	}

	return results, nil
}

func (r *batchRepoImpl) Create(ctx context.Context, batchID string, params domain.CreateBatchParams) (*domain.BaseBatch, *apierror.APIError) {
	ctx, span := batchRepoTracer.Start(ctx, "repository.batch.create")
	defer span.End()

	// Generate and insert the primary quantity.
	quantityID, apiErr := id.GenID(id.QuantityIDPrefix, nil)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	err := r.queries.InsertBatchQuantity(ctx, sqlc.InsertBatchQuantityParams{
		ID:     quantityID,
		Value:  params.Quantity.Measure.String(),
		UnitID: params.Quantity.UnitID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	// Generate and insert seconds quantity if provided.
	var secondsQuantityID gosql.NullString
	if params.Seconds != nil {
		sqID, apiErr := id.GenID(id.QuantityIDPrefix, nil)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		err = r.queries.InsertBatchQuantity(ctx, sqlc.InsertBatchQuantityParams{
			ID:     sqID,
			Value:  params.Seconds.Measure.String(),
			UnitID: params.Seconds.UnitID,
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		secondsQuantityID = gosql.NullString{String: sqID, Valid: true}
	}

	// Generate and insert waste quantity if provided.
	var wasteQuantityID gosql.NullString
	if params.Waste != nil {
		wqID, apiErr := id.GenID(id.QuantityIDPrefix, nil)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		err = r.queries.InsertBatchQuantity(ctx, sqlc.InsertBatchQuantityParams{
			ID:     wqID,
			Value:  params.Waste.Measure.String(),
			UnitID: params.Waste.UnitID,
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		wasteQuantityID = gosql.NullString{String: wqID, Valid: true}
	}

	// Insert the batch.
	err = r.queries.InsertBatch(ctx, sqlc.InsertBatchParams{
		ID:                batchID,
		AccountID:         params.AccountID,
		ItemID:            params.ItemID,
		QuantityID:        quantityID,
		SecondsQuantityID: secondsQuantityID,
		WasteQuantityID:   wasteQuantityID,
		ProductionStepID:  db.NullString(params.ProductionStepID),
		ScanningStationID: db.NullString(params.ScanningStationID),
		ProductionRunID:   db.NullString(params.ProductionRunID),
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	for _, machineID := range params.MachineIDs {
		if machineID == "" {
			continue
		}
		err = r.queries.LinkBatchMachine(ctx, sqlc.LinkBatchMachineParams{
			BatchID:   batchID,
			MachineID: machineID,
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}

	// Fetch and return the created batch.
	row, err := r.queries.GetBatchBase(ctx, sqlc.GetBatchBaseParams{
		ID:        batchID,
		AccountID: params.AccountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return mapBaseBatchRow(row), nil
}

func (r *batchRepoImpl) MarkAsScanned(ctx context.Context, accountID, batchID string) *apierror.APIError {
	ctx, span := batchRepoTracer.Start(ctx, "repository.batch.mark_as_scanned")
	defer span.End()

	err := r.queries.UpdateBatchScannedAt(ctx, sqlc.UpdateBatchScannedAtParams{
		ID:        batchID,
		AccountID: accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *batchRepoImpl) ConnectProductionStep(ctx context.Context, accountID, batchID, productionStepID string) *apierror.APIError {
	ctx, span := batchRepoTracer.Start(ctx, "repository.batch.connect_production_step")
	defer span.End()

	err := r.queries.UpdateBatchProductionStepID(ctx, sqlc.UpdateBatchProductionStepIDParams{
		ProductionStepID: gosql.NullString{String: productionStepID, Valid: true},
		ID:               batchID,
		AccountID:        accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *batchRepoImpl) ConnectScanningStation(ctx context.Context, accountID, batchID, scanningStationID string) *apierror.APIError {
	ctx, span := batchRepoTracer.Start(ctx, "repository.batch.connect_scanning_station")
	defer span.End()

	err := r.queries.UpdateBatchScanningStationID(ctx, sqlc.UpdateBatchScanningStationIDParams{
		ScanningStationID: gosql.NullString{String: scanningStationID, Valid: true},
		ID:                batchID,
		AccountID:         accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *batchRepoImpl) ConnectOneToOne(ctx context.Context, accountID, sourceBatchID, targetBatchID string, autoClose bool) *apierror.APIError {
	ctx, span := batchRepoTracer.Start(ctx, "repository.batch.connect_one_to_one")
	defer span.End()

	err := r.queries.InsertBatchFlow(ctx, sqlc.InsertBatchFlowParams{
		SourceBatchID: sourceBatchID,
		TargetBatchID: targetBatchID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	if autoClose {
		err := r.queries.UpdateBatchClosedAt(ctx, sqlc.UpdateBatchClosedAtParams{
			ID:        sourceBatchID,
			AccountID: accountID,
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return tracing.Trace(span, apiErr)
		}
	}

	return nil
}

func (r *batchRepoImpl) ConnectManyToOne(ctx context.Context, accountID string, sourceBatchIDs []string, targetBatchID string, autoClose bool) *apierror.APIError {
	ctx, span := batchRepoTracer.Start(ctx, "repository.batch.connect_many_to_one")
	defer span.End()

	for _, sourceBatchID := range sourceBatchIDs {
		err := r.queries.InsertBatchFlow(ctx, sqlc.InsertBatchFlowParams{
			SourceBatchID: sourceBatchID,
			TargetBatchID: targetBatchID,
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return tracing.Trace(span, apiErr)
		}

		if autoClose {
			err := r.queries.UpdateBatchClosedAt(ctx, sqlc.UpdateBatchClosedAtParams{
				ID:        sourceBatchID,
				AccountID: accountID,
			})
			if apiErr := db.MapSQLError(err); apiErr != nil {
				return tracing.Trace(span, apiErr)
			}
		}
	}

	return nil
}

func (r *batchRepoImpl) Close(ctx context.Context, accountID, batchID string) (*domain.BaseBatch, *apierror.APIError) {
	ctx, span := batchRepoTracer.Start(ctx, "repository.batch.close")
	defer span.End()

	err := r.queries.UpdateBatchClosedAt(ctx, sqlc.UpdateBatchClosedAtParams{
		ID:        batchID,
		AccountID: accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	row, err := r.queries.GetBatchBase(ctx, sqlc.GetBatchBaseParams{
		ID:        batchID,
		AccountID: accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return mapBaseBatchRow(row), nil
}

func (r *batchRepoImpl) CloseIfLastStep(ctx context.Context, accountID, batchID, productionStepID string) *apierror.APIError {
	ctx, span := batchRepoTracer.Start(ctx, "repository.batch.close_if_last_step")
	defer span.End()

	isLast, err := r.queries.IsLastProductionStep(ctx, productionStepID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	if isLast == 1 {
		err := r.queries.UpdateBatchClosedAt(ctx, sqlc.UpdateBatchClosedAtParams{
			ID:        batchID,
			AccountID: accountID,
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return tracing.Trace(span, apiErr)
		}
	}

	return nil
}

func (r *batchRepoImpl) CloseIfFullyUsed(ctx context.Context, accountID string, batch domain.BaseBatch, producedUnit domain.LightUnit, productionStepID string) *apierror.APIError {
	ctx, span := batchRepoTracer.Start(ctx, "repository.batch.close_if_fully_used")
	defer span.End()

	// Get output batches for the given batch.
	outputBatches, apiErr := r.FindOutputBatches(ctx, accountID, batch.ID)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	// Sum all output batch quantities (firsts + seconds + waste).
	totalUsed := decimal.Zero
	for _, ob := range outputBatches {
		totalUsed = totalUsed.Add(ob.Quantity.Measure)
		if ob.Seconds != nil {
			totalUsed = totalUsed.Add(ob.Seconds.Measure)
		}
		if ob.Waste != nil {
			totalUsed = totalUsed.Add(ob.Waste.Measure)
		}
	}

	// Compare against the batch's quantity.
	remaining := batch.Quantity.Measure.Sub(totalUsed)
	if remaining.LessThanOrEqual(decimal.Zero) {
		err := r.queries.UpdateBatchClosedAt(ctx, sqlc.UpdateBatchClosedAtParams{
			ID:        batch.ID,
			AccountID: accountID,
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return tracing.Trace(span, apiErr)
		}
	}

	return nil
}

func (r *batchRepoImpl) Delete(ctx context.Context, accountID, batchID string) (*domain.BaseBatch, *apierror.APIError) {
	ctx, span := batchRepoTracer.Start(ctx, "repository.batch.delete")
	defer span.End()

	// Fetch the batch before deleting so we can return it.
	row, err := r.queries.GetBatchBase(ctx, sqlc.GetBatchBaseParams{
		ID:        batchID,
		AccountID: accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	batch := mapBaseBatchRow(row)

	// Delete flow edges.
	err = r.queries.DeleteBatchFlowByBatchID(ctx, sqlc.DeleteBatchFlowByBatchIDParams{
		BatchID: batchID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	// Delete machine associations.
	err = r.queries.DeleteBatchesMachinesByBatchID(ctx, batchID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	// Delete the batch.
	err = r.queries.DeleteBatch(ctx, sqlc.DeleteBatchParams{
		ID:        batchID,
		AccountID: accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return batch, nil
}

func (r *batchRepoImpl) DeleteMany(ctx context.Context, accountID string, batchIDs []string) *apierror.APIError {
	ctx, span := batchRepoTracer.Start(ctx, "repository.batch.delete_many")
	defer span.End()

	if len(batchIDs) == 0 {
		return nil
	}

	// Delete flow edges for each batch.
	for _, bid := range batchIDs {
		err := r.queries.DeleteBatchFlowByBatchID(ctx, sqlc.DeleteBatchFlowByBatchIDParams{
			BatchID: bid,
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return tracing.Trace(span, apiErr)
		}

		err = r.queries.DeleteBatchesMachinesByBatchID(ctx, bid)
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return tracing.Trace(span, apiErr)
		}
	}

	// Delete all batches.
	err := r.queries.DeleteBatchesByIDs(ctx, sqlc.DeleteBatchesByIDsParams{
		Ids:       batchIDs,
		AccountID: accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

// Ensure compile-time interface compliance.
var _ domain.BatchRepo = (*batchRepoImpl)(nil)
