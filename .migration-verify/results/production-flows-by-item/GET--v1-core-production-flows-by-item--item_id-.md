# Verification: GET /v1/core/production-flows/by-item/{item_id}

## Result: Issues found and fixed

## What was compared

- **Permission checks**: Both require internal actor + `production_steps:read` permission. Go also validates target account is set. **Parity confirmed.**
- **Business logic / graph traversal**: Both find the initial step(s) producing the item, load all edges for the account, BFS backward through parent edges to collect the relevant step set, then fetch full data for each step. **Parity confirmed.**
- **DB queries**: `FindStepsByProducedItem` matches Dashboard's `findOneByProducedBlock` (Go returns all matches vs Dashboard's `findFirst` — a safe superset). `GetAllStepEdgesForAccount` matches the lightweight graph fetch. `GetProductionFlowStep` + `GetProductionStepConsumptions` match the full step data fetch. **Parity confirmed.**
- **Error handling**: Both return empty steps array when no producing step is found. **Parity confirmed.**
- **Response shape**: Both return steps with production, consumptions, in/out step refs, scanning station, leveling factor, allowances, labor rate/time, overhead rate. **Parity confirmed after fixes.**
- **Side effects**: None in either implementation (read-only endpoint). **Parity confirmed.**
- **Idempotency**: GET endpoint, no idempotency needed. **Parity confirmed.**

## Issues found and fixed

### 1. Consumption record ID was wrong (bug)
**File:** `grpc_production_flow_handler.go:49`
The consumption `Id` field was set to `c.ConsumedItem.ID` (the item ID) instead of the consumption record's own ID. Fixed by:
- Adding `ID` field to `StepConsumption` domain model
- Mapping `row.ID` (consumption record ID) in `mapStepConsumptionRow`
- Using `c.ID` instead of `c.ConsumedItem.ID` in the gRPC handler

### 2. Production record ID was wrong (bug)
**File:** `grpc_production_flow_handler.go:33`
The production `Id` field was set to `step.Production.ProducedItem.ID` (the item ID) instead of the production record's own ID. Fixed by:
- Adding `ID` field to `StepProduction` domain model
- Mapping `row.ProductionID` in `mapProductionStepRow`, `mapFindStepRow`, and `GetFlowStep`
- Using `step.Production.ID` instead of `step.Production.ProducedItem.ID` in the gRPC handler

### 3. Scanning station object type was wrong (bug)
**File:** `presenter.go:117`
The scanning station ref used `ObjectTypeProductionStep` instead of `ObjectTypeScanningStation`. Fixed to use the correct object type.

## Minor behavioral differences (acceptable)

- **FindFirst vs FindMany**: Dashboard uses `findFirst` to get a single producing step; Go uses `findMany` returning all matches. The Go approach handles the edge case of multiple steps producing the same item, which is a safe superset.
- **In/Out edge filtering**: Go filters in/out step references to only include steps within the relevant flow graph. Dashboard returns all in/out edges for each step (including edges to steps outside the flow). The Go behavior is arguably more correct for a flow-specific view.

## Files modified

- `services/core-service/internal/domain/production_step_query_models.go` — Added `ID` fields to `StepProduction` and `StepConsumption`
- `services/core-service/internal/infrastructure/repository/production_step_query_repository.go` — Map production ID and consumption ID from SQL rows
- `services/core-service/internal/infrastructure/repository/production_flow_repository.go` — Map production ID in `GetFlowStep`
- `services/core-service/internal/infrastructure/grpc/grpc_production_flow_handler.go` — Use correct IDs for production and consumption
- `services/api-gateway/endpoints/production-flows/presenter.go` — Fix scanning station object type
