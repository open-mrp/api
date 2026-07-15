package service

import (
	"context"
	"sort"

	"github.com/augno/api/services/core-service/internal/domain"
	apierror "github.com/augno/api/shared/errors"
	"github.com/shopspring/decimal"
)

// itemTypeMaterial is the item_type_code for raw materials — the leaf inputs the
// batch algorithm reserves against (matches legacy consumedItem.type === 'material').
const itemTypeMaterial = "material"

// batchQuantityPrecision is the decimal precision the final batch quantity is rounded
// to, removing the sub-unit float noise the chained ratio divisions introduce.
const batchQuantityPrecision = 10

// blockAccum accumulates the total base-unit quantity for one material-only block
// item across all order lines, plus a representative unit to express the result in.
type blockAccum struct {
	itemID    string
	baseTotal decimal.Decimal
	unitID    string
}

// computeProductionBatchItems reproduces legacy OrderRepo.getBaseItems / getBaseBatches:
// for each order line it walks the produced item's production-flow graph, finds the
// production blocks that consume only materials, normalizes their output to a standard
// base unit, scales by the ordered quantity, and aggregates one batch per block item.
func (s *salesOrderSvcImpl) computeProductionBatchItems(ctx context.Context, accountID string, lines []domain.ProductionBatchLineInput) ([]domain.ProductionBatchItem, *apierror.APIError) {
	flowRepo := s.repos.NewProductionFlowRepo()
	stepQueryRepo := s.repos.NewProductionStepQueryRepo()

	type lineFlow struct {
		line domain.ProductionBatchLineInput
		flow []*domain.ProductionFlowStep
	}
	lineFlows := make([]lineFlow, 0, len(lines))
	unitIDSet := make(map[string]struct{})

	for _, line := range lines {
		flow, apiErr := assembleFlowForItem(ctx, flowRepo, stepQueryRepo, accountID, line.ProducedItemID)
		if apiErr != nil {
			return nil, apiErr
		}
		if len(flow) == 0 {
			continue // No production flow for this item → no batches (matches legacy).
		}
		lineFlows = append(lineFlows, lineFlow{line: line, flow: flow})
		unitIDSet[line.OrderedUnitID] = struct{}{}
		for _, st := range flow {
			unitIDSet[st.Production.Quantity.Unit.ID] = struct{}{}
			for _, c := range st.Consumptions {
				unitIDSet[c.Quantity.Unit.ID] = struct{}{}
			}
		}
	}
	if len(lineFlows) == 0 {
		return nil, nil
	}

	unitIDs := make([]string, 0, len(unitIDSet))
	for id := range unitIDSet {
		unitIDs = append(unitIDs, id)
	}
	factors, apiErr := s.repos.NewUnitConversionRepo().GetUnitFactors(ctx, accountID, unitIDs)
	if apiErr != nil {
		return nil, apiErr
	}

	blocks := make(map[string]*blockAccum)
	for _, lf := range lineFlows {
		orderedBase := toBaseMeasureRaw(lf.line.OrderedMeasure, lf.line.OrderedUnitID, factors)
		computeMaterialOnlyBlocks(lf.flow, lf.line.ProducedItemID, orderedBase, factors, blocks)
	}

	items := make([]domain.ProductionBatchItem, 0, len(blocks))
	for _, acc := range blocks {
		measure := acc.baseTotal
		if f, ok := factors[acc.unitID]; ok {
			measure = f.FromBase(acc.baseTotal)
		}
		// Round away the sub-nano-unit noise from the chained ratio divisions (e.g. 1/6)
		// so batch quantities land on clean values while preserving real fractions.
		items = append(items, domain.ProductionBatchItem{ItemID: acc.itemID, Measure: measure.Round(batchQuantityPrecision), UnitID: acc.unitID})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ItemID < items[j].ItemID })
	return items, nil
}

// toBaseMeasure converts a step quantity to its dimension's base measure.
func toBaseMeasure(q domain.BatchQuantity, factors map[string]domain.UnitFactors) decimal.Decimal {
	return toBaseMeasureRaw(q.Measure, q.Unit.ID, factors)
}

func toBaseMeasureRaw(measure decimal.Decimal, unitID string, factors map[string]domain.UnitFactors) decimal.Decimal {
	if f, ok := factors[unitID]; ok {
		return f.ToBase(measure)
	}
	return measure // Unknown unit → identity (degrades gracefully, as legacy does).
}

// computeMaterialOnlyBlocks runs the normalization + material-only-block traversal for a
// single line's flow, accumulating scaled block quantities into `out`. It is a pure
// function over the flow graph, so it is fully unit-testable with constructed graphs.
func computeMaterialOnlyBlocks(
	flow []*domain.ProductionFlowStep,
	targetItemID string,
	orderedBaseMeasure decimal.Decimal,
	factors map[string]domain.UnitFactors,
	out map[string]*blockAccum,
) {
	stepMap := make(map[string]*domain.ProductionFlowStep, len(flow))
	for _, st := range flow {
		stepMap[st.ID] = st
	}

	// The target step produces the ordered item; batches are generated relative to it.
	var target *domain.ProductionFlowStep
	for _, st := range flow {
		if st.Production.ProducedItem.ID == targetItemID {
			target = st
			break
		}
	}
	if target == nil {
		return
	}

	// normMap: step ID → dimensionless normalization factor.
	normMap := make(map[string]decimal.Decimal, len(flow))
	standard := decimal.NewFromInt(1) // 1 base unit of the target produced item.
	targetProdBase := toBaseMeasure(target.Production.Quantity, factors)

	for _, st := range flow {
		switch {
		case st.ID == target.ID:
			if targetProdBase.IsZero() {
				normMap[st.ID] = decimal.Zero
			} else {
				normMap[st.ID] = standard.Div(targetProdBase)
			}
		case len(st.OutStepIDs) == 0:
			normMap[st.ID] = decimal.NewFromInt(1) // Terminal step.
		default:
			// Ratio for a non-terminal step: how much of this step's output the following
			// step consumes, per unit produced here.
			following := stepMap[st.OutStepIDs[0]]
			prodBase := toBaseMeasure(st.Production.Quantity, factors)
			ratio := decimal.NewFromInt(1)
			if following != nil && !prodBase.IsZero() {
				for _, c := range following.Consumptions {
					if c.ConsumedItem.ID == st.Production.ProducedItem.ID {
						ratio = toBaseMeasure(c.Quantity, factors).Div(prodBase)
						break
					}
				}
			}
			normMap[st.ID] = ratio
		}
	}

	// Backward pass: propagate the cumulative downstream factor into each step, walking
	// back from the target through its input edges.
	backQueue := []string{target.ID}
	processed := map[string]bool{target.ID: true}
	for len(backQueue) > 0 {
		id := backQueue[0]
		backQueue = backQueue[1:]
		st := stepMap[id]
		if st == nil {
			continue
		}
		if len(st.OutStepIDs) > 0 {
			if parent, ok := normMap[st.OutStepIDs[0]]; ok {
				normMap[id] = parent.Mul(normMap[id])
			}
		}
		for _, inID := range st.InStepIDs {
			if !processed[inID] {
				processed[inID] = true
				backQueue = append(backQueue, inID)
			}
		}
	}

	// Forward pass from source steps: a step is a material-only block if it has no inputs
	// (a source) or every one of its consumptions is a material. Scale and aggregate.
	queue := make([]string, 0)
	for _, st := range flow {
		if len(st.InStepIDs) == 0 {
			queue = append(queue, st.ID)
		}
	}
	visited := make(map[string]bool)
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		if visited[id] {
			continue
		}
		visited[id] = true
		st := stepMap[id]
		if st == nil {
			continue
		}
		factor, ok := normMap[id]
		if !ok {
			continue
		}

		matCount := 0
		for _, c := range st.Consumptions {
			if c.ConsumedItem.Type == itemTypeMaterial {
				matCount++
			}
		}
		if len(st.InStepIDs) == 0 || (matCount > 0 && matCount == len(st.Consumptions)) {
			prodBase := toBaseMeasure(st.Production.Quantity, factors)
			scaled := prodBase.Mul(factor).Mul(orderedBaseMeasure)
			itemID := st.Production.ProducedItem.ID
			if acc, exists := out[itemID]; exists {
				acc.baseTotal = acc.baseTotal.Add(scaled)
			} else {
				out[itemID] = &blockAccum{itemID: itemID, baseTotal: scaled, unitID: st.Production.Quantity.Unit.ID}
			}
		}

		if len(st.OutStepIDs) > 0 && !visited[st.OutStepIDs[0]] {
			queue = append(queue, st.OutStepIDs[0])
		}
	}
}

// assembleFlowForItem builds the production-flow graph for an item directly from the
// flow repo (no permission gate), mirroring legacy ProductionFlowRepo.buildProductionFlow:
// pick a single step that produces the item and walk the account's edge graph backward
// through its input (upstream) edges, hydrating each relevant step's production,
// consumptions, and in/out edges. The traversal is deliberately backward-only and rooted
// at one producing step: following downstream (out) edges, or seeding the walk from every
// step that produces the item, would pull sibling recipes — other product variants that
// merely share an upstream step, or an alternate recipe for the same item — into this
// item's flow, generating batches for parts that are not on the ordered item's recipe.
func assembleFlowForItem(
	ctx context.Context,
	flowRepo domain.ProductionFlowRepo,
	stepQueryRepo domain.ProductionStepQueryRepo,
	accountID, itemID string,
) ([]*domain.ProductionFlowStep, *apierror.APIError) {
	initialStepIDs, apiErr := flowRepo.FindStepsByProducedItem(ctx, accountID, itemID)
	if apiErr != nil {
		return nil, apiErr
	}
	if len(initialStepIDs) == 0 {
		return nil, nil
	}
	// Root the flow at a single producing step (legacy findOneByProducedBlock). Sort so the
	// choice is deterministic when an item has more than one producing recipe.
	sort.Strings(initialStepIDs)
	rootStepID := initialStepIDs[0]

	edges, apiErr := flowRepo.GetAllStepEdgesForAccount(ctx, accountID)
	if apiErr != nil {
		return nil, apiErr
	}

	parentMap := make(map[string][]string)
	childMap := make(map[string][]string)
	for _, e := range edges {
		parentMap[e.ChildStepID] = append(parentMap[e.ChildStepID], e.ParentStepID)
		childMap[e.ParentStepID] = append(childMap[e.ParentStepID], e.ChildStepID)
	}

	// Backward-only walk from the root: the relevant set is the root plus its transitive
	// upstream ancestors (the steps that produce this item's inputs, recursively).
	relevant := make(map[string]bool)
	queue := []string{rootStepID}
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		if relevant[id] {
			continue
		}
		relevant[id] = true
		for _, p := range parentMap[id] {
			if !relevant[p] {
				queue = append(queue, p)
			}
		}
	}

	steps := make([]*domain.ProductionFlowStep, 0, len(relevant))
	for id := range relevant {
		step, apiErr := flowRepo.GetFlowStep(ctx, accountID, id)
		if apiErr != nil {
			return nil, apiErr
		}
		stepDetail, apiErr := stepQueryRepo.Find(ctx, accountID, id)
		if apiErr != nil {
			return nil, apiErr
		}
		step.Consumptions = stepDetail.Consumptions

		in := make([]string, 0)
		for _, p := range parentMap[id] {
			if relevant[p] {
				in = append(in, p)
			}
		}
		step.InStepIDs = in

		out := make([]string, 0)
		for _, c := range childMap[id] {
			if relevant[c] {
				out = append(out, c)
			}
		}
		step.OutStepIDs = out

		steps = append(steps, step)
	}
	return steps, nil
}
