package repository

import (
	"context"
	gosql "database/sql"
	"strconv"
	"strings"
	"time"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	"github.com/open-mrp/apikit/db"
	"github.com/open-mrp/apikit/pagination"
	"github.com/open-mrp/apikit/tracing"
)

var manufacturingMethodRepoTracer = tracing.GetTracer("core-service.manufacturing_method_repository")

type manufacturingMethodRepoImpl struct {
	queries *sqlc.Queries
}

func NewManufacturingMethodRepo(queries *sqlc.Queries) domain.ManufacturingMethodRepo {
	return &manufacturingMethodRepoImpl{queries: queries}
}

func mmNullString(s *string) gosql.NullString {
	if s == nil {
		return gosql.NullString{}
	}
	return gosql.NullString{String: *s, Valid: true}
}

func mmStringPtr(s gosql.NullString) *string {
	if !s.Valid {
		return nil
	}
	v := s.String
	return &v
}

// percentFromFactor is a stored fraction (0.15) as a whole-number percent ("15").
func percentFromFactor(factor string) string {
	return scaleDecimal(factor, 2)
}

// factorFromPercent is a whole-number percent ("15") as the stored fraction ("0.15").
func factorFromPercent(percent string) string {
	return scaleDecimal(percent, -2)
}

// scaleDecimal moves a decimal string's point by shift places (right when positive) without float rounding, trimming
// trailing zeros.
func scaleDecimal(v string, shift int) string {
	v = strings.TrimSpace(v)
	neg := strings.HasPrefix(v, "-")
	v = strings.TrimPrefix(strings.TrimPrefix(v, "-"), "+")
	intPart, frac, _ := strings.Cut(v, ".")
	digits := intPart + frac
	point := len(intPart) + shift
	for point < 0 {
		digits = "0" + digits
		point++
	}
	for point > len(digits) {
		digits += "0"
	}
	whole := strings.TrimLeft(digits[:point], "0")
	if whole == "" {
		whole = "0"
	}
	fraction := strings.TrimRight(digits[point:], "0")
	out := whole
	if fraction != "" {
		out += "." + fraction
	}
	if neg && out != "0" {
		out = "-" + out
	}
	return out
}

func methodFromRow(r sqlc.GetManufacturingMethodsByIDsRow) *domain.ManufacturingMethod {
	m := &domain.ManufacturingMethod{
		ID:        r.ID,
		AccountID: r.AccountID,
		ItemID:    r.ItemID,
		Version:   r.Version,
		Status:    constants.ManufacturingMethodStatus(r.StatusCode),
		OutputQuantity: domain.QuantityValue{
			Value: r.OutputQuantityValue, UnitID: r.OutputQuantityUnitID, UnitType: r.OutputQuantityUnitType,
		},
		SalesOrderLineID: mmStringPtr(r.SalesOrderLineID),
		CreatedAt:        r.CreatedAt,
		UpdatedAt:        r.UpdatedAt,
	}
	if r.ActivatedAt.Valid {
		t := r.ActivatedAt.Time
		m.ActivatedAt = &t
	}
	return m
}

func (r *manufacturingMethodRepoImpl) GetByIDs(ctx context.Context, accountID string, ids []string) ([]*domain.ManufacturingMethod, *apierror.APIError) {
	ctx, span := manufacturingMethodRepoTracer.Start(ctx, "repository.manufacturing_method.get_by_ids")
	defer span.End()

	if len(ids) == 0 {
		return nil, nil
	}
	rows, err := r.queries.GetManufacturingMethodsByIDs(ctx, sqlc.GetManufacturingMethodsByIDsParams{AccountID: accountID, Ids: ids})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	byID := make(map[string]*domain.ManufacturingMethod, len(rows))
	for _, row := range rows {
		byID[row.ID] = methodFromRow(row)
	}
	out := make([]*domain.ManufacturingMethod, 0, len(rows))
	for _, id := range ids {
		m, ok := byID[id]
		if !ok {
			continue
		}
		delete(byID, id)
		if apiErr := r.embedMembers(ctx, m); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		out = append(out, m)
	}
	return out, nil
}

// embedMembers loads the first operations and materials of a method for its embedded lists.
func (r *manufacturingMethodRepoImpl) embedMembers(ctx context.Context, m *domain.ManufacturingMethod) *apierror.APIError {
	ops, err := r.queries.ListMethodOperationsAfter(ctx, sqlc.ListMethodOperationsAfterParams{
		ManufacturingMethodID: gosql.NullString{String: m.ID, Valid: true}, AfterPosition: -1 << 31, Limit: domain.EmbeddedMethodMembers,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return apiErr
	}
	m.OperationIDs = make([]string, len(ops))
	for i, o := range ops {
		m.OperationIDs[i] = o.ID
	}
	if len(ops) == domain.EmbeddedMethodMembers {
		c := encodePositionCursor(ops[len(ops)-2].Position, ops[len(ops)-2].ID, pagination.DirectionForward)
		m.OperationsNextCursor = &c
	}

	mats, err := r.queries.ListMethodMaterialIDs(ctx, sqlc.ListMethodMaterialIDsParams{
		ManufacturingMethodID: gosql.NullString{String: m.ID, Valid: true}, Limit: domain.EmbeddedMethodMembers,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return apiErr
	}
	m.MaterialIDs = make([]string, len(mats))
	for i, x := range mats {
		m.MaterialIDs[i] = x.ID
	}
	m.MaterialsNextCursor = embeddedNextCursor(mats, domain.EmbeddedMethodMembers,
		func(x sqlc.ListMethodMaterialIDsRow) time.Time { return x.CreatedAt },
		func(x sqlc.ListMethodMaterialIDsRow) string { return x.ID })
	return nil
}

func (r *manufacturingMethodRepoImpl) Get(ctx context.Context, accountID, id string) (*domain.ManufacturingMethod, *apierror.APIError) {
	methods, apiErr := r.GetByIDs(ctx, accountID, []string{id})
	if apiErr != nil {
		return nil, apiErr
	}
	if len(methods) == 0 {
		return nil, apierror.NewNotFoundError("Manufacturing method not found.")
	}
	return methods[0], nil
}

func (r *manufacturingMethodRepoImpl) Lock(ctx context.Context, accountID, id string) (constants.ManufacturingMethodStatus, string, *apierror.APIError) {
	row, err := r.queries.LockManufacturingMethod(ctx, sqlc.LockManufacturingMethodParams{ID: id, AccountID: accountID})
	if err == gosql.ErrNoRows {
		return "", "", apierror.NewNotFoundError("Manufacturing method not found.")
	}
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return "", "", apiErr
	}
	return constants.ManufacturingMethodStatus(row.StatusCode), row.ItemID, nil
}

func (r *manufacturingMethodRepoImpl) NextVersion(ctx context.Context, itemID string) (int32, *apierror.APIError) {
	v, err := r.queries.MaxManufacturingMethodVersion(ctx, itemID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return 0, apiErr
	}
	return int32(v) + 1, nil // #nosec G115 -- versions count an item's methods
}

func (r *manufacturingMethodRepoImpl) Insert(ctx context.Context, m *domain.ManufacturingMethod) *apierror.APIError {
	var activatedAt gosql.NullTime
	if m.ActivatedAt != nil {
		activatedAt = gosql.NullTime{Time: *m.ActivatedAt, Valid: true}
	}
	err := r.queries.InsertManufacturingMethod(ctx, sqlc.InsertManufacturingMethodParams{
		ID:                   m.ID,
		AccountID:            m.AccountID,
		ItemID:               m.ItemID,
		Version:              m.Version,
		StatusCode:           string(m.Status),
		OutputQuantityValue:  m.OutputQuantity.Value,
		OutputQuantityUnitID: m.OutputQuantity.UnitID,
		ActivatedAt:          activatedAt,
	})
	return db.MapSQLError(err)
}

func (r *manufacturingMethodRepoImpl) UpdateOutput(ctx context.Context, accountID, id string, q domain.QuantityValue) *apierror.APIError {
	return db.MapSQLError(r.queries.UpdateManufacturingMethodOutput(ctx, sqlc.UpdateManufacturingMethodOutputParams{
		ID: id, AccountID: accountID, OutputQuantityValue: q.Value, OutputQuantityUnitID: q.UnitID,
	}))
}

func (r *manufacturingMethodRepoImpl) SetStatus(ctx context.Context, accountID, id string, status constants.ManufacturingMethodStatus, activatedAt *time.Time) *apierror.APIError {
	var at gosql.NullTime
	if activatedAt != nil {
		at = gosql.NullTime{Time: *activatedAt, Valid: true}
	}
	return db.MapSQLError(r.queries.UpdateManufacturingMethodStatus(ctx, sqlc.UpdateManufacturingMethodStatusParams{
		ID: id, AccountID: accountID, StatusCode: string(status), ActivatedAt: at,
	}))
}

func (r *manufacturingMethodRepoImpl) Delete(ctx context.Context, accountID, id string) *apierror.APIError {
	return db.MapSQLError(r.queries.DeleteManufacturingMethodRow(ctx, sqlc.DeleteManufacturingMethodRowParams{ID: id, AccountID: accountID}))
}

func (r *manufacturingMethodRepoImpl) ActiveStandardIDsForItem(ctx context.Context, accountID, itemID string) ([]string, *apierror.APIError) {
	ids, err := r.queries.ListActiveStandardMethodIDsForItem(ctx, sqlc.ListActiveStandardMethodIDsForItemParams{ItemID: itemID, AccountID: accountID})
	return ids, db.MapSQLError(err)
}

func (r *manufacturingMethodRepoImpl) DefaultMethodIDs(ctx context.Context, accountID string, itemIDs []string) (map[string]string, *apierror.APIError) {
	if len(itemIDs) == 0 {
		return map[string]string{}, nil
	}
	rows, err := r.queries.ListDefaultMethodIDsForItems(ctx, sqlc.ListDefaultMethodIDsForItemsParams{AccountID: accountID, ItemIds: itemIDs})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, apiErr
	}
	out := make(map[string]string, len(rows))
	for _, row := range rows {
		if _, ok := out[row.ItemID]; !ok {
			out[row.ItemID] = row.ID
		}
	}
	return out, nil
}

func (r *manufacturingMethodRepoImpl) ItemsWithActiveMethods(ctx context.Context, accountID string, itemIDs []string) (map[string]bool, *apierror.APIError) {
	out := map[string]bool{}
	if len(itemIDs) == 0 {
		return out, nil
	}
	ids, err := r.queries.ItemsWithActiveMethods(ctx, sqlc.ItemsWithActiveMethodsParams{AccountID: accountID, ItemIds: itemIDs})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, apiErr
	}
	for _, id := range ids {
		out[id] = true
	}
	return out, nil
}

// --- list ---

const manufacturingMethodColumns = `m.id, m.account_id, m.item_id, m.version, m.status_code, m.output_quantity_value, ` +
	`m.output_quantity_unit_id, u.unit_dimension_code, m.sales_order_line_id, m.activated_at, m.created_at, m.updated_at`

// buildManufacturingMethodListPage chooses one page of an account's methods, newest first. item_ids[] rides its own
// key; statuses and work centers are residual.
func buildManufacturingMethodListPage(params domain.ListManufacturingMethodsParams, dir pagination.Direction, cursor *pagination.StringCursor, limit int32) keysetPage {
	page := keysetPage{
		table: "manufacturing_method", alias: "m", sortColumn: "created_at",
		createdIndex: "manufacturing_method_account_id_created_at_id_idx",
		filters: []keysetFilter{
			{column: "m.item_id", index: "manufacturing_method_item_id_created_at_id_idx", values: params.ItemIDs},
		},
		where: []string{"m.account_id = ?"},
		args:  []any{params.AccountID},
		desc:  dir != pagination.DirectionBackward,
		limit: limit,
	}
	if len(params.Statuses) > 0 {
		page.where = append(page.where, "m.status_code IN ("+placeholders(len(params.Statuses))+")")
		page.args = append(page.args, stringArgs(enumStrings(params.Statuses))...)
	}
	if len(params.WorkCenterIDs) > 0 {
		page.where = append(page.where, "m.id IN (SELECT ps.manufacturing_method_id FROM production_step ps"+
			" WHERE ps.account_id = ? AND ps.department_id IN ("+placeholders(len(params.WorkCenterIDs))+"))")
		page.args = append(page.args, params.AccountID)
		page.args = append(page.args, stringArgs(params.WorkCenterIDs)...)
	}
	if params.Created.After != nil {
		page.where = append(page.where, "m.created_at > ?")
		page.args = append(page.args, *params.Created.After)
	}
	if params.Created.Before != nil {
		page.where = append(page.where, "m.created_at < ?")
		page.args = append(page.args, *params.Created.Before)
	}
	if cursor != nil {
		cmp := "<"
		if dir == pagination.DirectionBackward {
			cmp = ">"
		}
		page.where = append(page.where, "(m.created_at "+cmp+" ? OR (m.created_at = ? AND m.id "+cmp+" ?))")
		page.args = append(page.args, cursor.OccurredAt, cursor.OccurredAt, cursor.ID)
	}
	return page
}

func (r *manufacturingMethodRepoImpl) List(ctx context.Context, params domain.ListManufacturingMethodsParams) (*domain.ListManufacturingMethodsResult, *apierror.APIError) {
	ctx, span := manufacturingMethodRepoTracer.Start(ctx, "repository.manufacturing_method.list")
	defer span.End()

	dir := pagination.DirectionForward
	var cursor *pagination.StringCursor
	var cursorDir *pagination.Direction
	if params.Cursor != nil {
		cur, err := pagination.DecodeStringCursor(*params.Cursor)
		if err != nil {
			return nil, apierror.NewParameterInvalidError("cursor", "Invalid pagination cursor.")
		}
		cursor, cursorDir, dir = &cur, &cur.Direction, cur.Direction
	}

	page := buildManufacturingMethodListPage(params, dir, cursor, params.Limit+1)
	orderBy := " ORDER BY m.created_at DESC, m.id DESC"
	if dir == pagination.DirectionBackward {
		orderBy = " ORDER BY m.created_at ASC, m.id ASC"
	}
	pageSQL, args := page.sql()
	query := "SELECT " + manufacturingMethodColumns + " FROM (" + pageSQL + ") page JOIN manufacturing_method m ON m.id = page.id" +
		" JOIN unit u ON u.id = m.output_quantity_unit_id" + orderBy
	rows, err := r.queries.DB().QueryContext(ctx, query, args...)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	defer func() { _ = rows.Close() }()
	var methods []*domain.ManufacturingMethod
	for rows.Next() {
		var row sqlc.GetManufacturingMethodsByIDsRow
		if err := rows.Scan(&row.ID, &row.AccountID, &row.ItemID, &row.Version, &row.StatusCode, &row.OutputQuantityValue,
			&row.OutputQuantityUnitID, &row.OutputQuantityUnitType, &row.SalesOrderLineID, &row.ActivatedAt, &row.CreatedAt, &row.UpdatedAt); err != nil {
			return nil, tracing.Trace(span, db.MapSQLError(err))
		}
		methods = append(methods, methodFromRow(row))
	}
	if apiErr := db.MapSQLError(rows.Err()); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	result, pageInfo := pagination.BuildPageString(methods, params.Limit, cursorDir,
		func(m *domain.ManufacturingMethod) time.Time { return m.CreatedAt },
		func(m *domain.ManufacturingMethod) string { return m.ID })
	for _, m := range result {
		if apiErr := r.embedMembers(ctx, m); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}
	return &domain.ListManufacturingMethodsResult{Methods: result, PageInfo: pageInfo}, nil
}

// --- operations ---

// encodePositionCursor is a list cursor over a method's operations, which are ordered by position.
func encodePositionCursor(position int32, id string, dir pagination.Direction) string {
	return pagination.EncodeValueCursor(pagination.ValueCursor{Value: strconv.Itoa(int(position)), ID: id, Direction: dir})
}

func (r *manufacturingMethodRepoImpl) ListOperationIDs(ctx context.Context, methodID string) ([]string, *apierror.APIError) {
	ids, err := r.queries.ListMethodOperationIDs(ctx, gosql.NullString{String: methodID, Valid: true})
	return ids, db.MapSQLError(err)
}

func (r *manufacturingMethodRepoImpl) ListOperations(ctx context.Context, params domain.ListMethodMembersParams) (*domain.ListMethodOperationsResult, *apierror.APIError) {
	ctx, span := manufacturingMethodRepoTracer.Start(ctx, "repository.manufacturing_method.list_operations")
	defer span.End()

	methodID := gosql.NullString{String: params.MethodID, Valid: true}
	var cursorDir *pagination.Direction
	type posRow struct {
		id       string
		position int32
	}
	var page []posRow
	if params.Cursor == nil {
		rows, err := r.queries.ListMethodOperationsAfter(ctx, sqlc.ListMethodOperationsAfterParams{
			ManufacturingMethodID: methodID, AfterPosition: -1 << 31, Limit: params.Limit + 1,
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		for _, x := range rows {
			page = append(page, posRow{x.ID, x.Position})
		}
	} else {
		cur, err := pagination.DecodeValueCursor(*params.Cursor)
		position, convErr := strconv.Atoi(cur.Value)
		if err != nil || convErr != nil {
			return nil, apierror.NewParameterInvalidError("cursor", "Invalid pagination cursor.")
		}
		cursorDir = &cur.Direction
		if cur.Direction == pagination.DirectionBackward {
			rows, err := r.queries.ListMethodOperationsBefore(ctx, sqlc.ListMethodOperationsBeforeParams{
				ManufacturingMethodID: methodID, BeforePosition: int32(position), Limit: params.Limit + 1, // #nosec G115 -- positions are small
			})
			if apiErr := db.MapSQLError(err); apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			for _, x := range rows {
				page = append(page, posRow{x.ID, x.Position})
			}
		} else {
			rows, err := r.queries.ListMethodOperationsAfter(ctx, sqlc.ListMethodOperationsAfterParams{
				ManufacturingMethodID: methodID, AfterPosition: int32(position), Limit: params.Limit + 1, // #nosec G115 -- positions are small
			})
			if apiErr := db.MapSQLError(err); apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			for _, x := range rows {
				page = append(page, posRow{x.ID, x.Position})
			}
		}
	}

	page, pageInfo := buildValuePage(page, params.Limit, cursorDir,
		func(x posRow) string { return encodePositionCursor(x.position, x.id, pagination.DirectionForward) },
		func(x posRow) string { return encodePositionCursor(x.position, x.id, pagination.DirectionBackward) })
	ids := make([]string, len(page))
	for i, x := range page {
		ids[i] = x.id
	}
	ops, apiErr := r.GetOperationsByIDs(ctx, params.AccountID, ids)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return &domain.ListMethodOperationsResult{Operations: ops, PageInfo: pageInfo}, nil
}

// buildValuePage trims a limit+1 page read in cursor direction and computes its page info, like
// pagination.BuildPageString, for lists whose cursors aren't timestamps.
func buildValuePage[T any](items []T, limit int32, cursorDir *pagination.Direction, next, prev func(T) string) ([]T, pagination.PageInfo) {
	if len(items) == 0 {
		return items, pagination.PageInfo{}
	}
	hasExtra := len(items) > int(limit)
	if hasExtra {
		items = items[:limit]
	}
	var pi pagination.PageInfo
	switch {
	case cursorDir == nil || *cursorDir == pagination.DirectionForward:
		pi.HasNextPage = hasExtra
		pi.HasPrevPage = cursorDir != nil
	default:
		for i, j := 0, len(items)-1; i < j; i, j = i+1, j-1 {
			items[i], items[j] = items[j], items[i]
		}
		pi.HasNextPage = true
		pi.HasPrevPage = hasExtra
	}
	if pi.HasNextPage {
		c := next(items[len(items)-1])
		pi.NextCursor = &c
	}
	if pi.HasPrevPage {
		c := prev(items[0])
		pi.PrevCursor = &c
	}
	return items, pi
}

func operationFromRow(r sqlc.GetMethodOperationsByIDsRow) *domain.MethodOperation {
	op := &domain.MethodOperation{
		ID:                  r.ID,
		MethodID:            r.ManufacturingMethodID.String,
		Position:            r.Position,
		Sequence:            constants.OperationSequence(r.SequenceCode),
		Type:                constants.OperationType(r.TypeCode),
		WorkCenterID:        mmStringPtr(r.DepartmentID),
		ScanningStationID:   mmStringPtr(r.ScanningStationID),
		LevelingPercent:     percentFromFactor(r.LevelingFactor),
		AllowancePercent:    percentFromFactor(r.Allowances),
		WorkInstructions:    mmStringPtr(r.WorkInstructions),
		LaborRateID:         r.LaborRateID,
		LaborTimeID:         r.LaborTimeID,
		OverheadRateID:      r.OverheadRateID,
		SetupTimeQuantityID: mmStringPtr(r.SetupTimeQuantityID),
		MachineTimeID:       mmStringPtr(r.MachineTimeID),
		LaborRate: unitPriceFromColumns(r.LaborRateAmount, r.LaborRateCurrency, r.LaborRatePerUnitID,
			r.LegacyLaborRateValue, r.LegacyLaborRatePerUnitID),
		OverheadRate: unitPriceFromColumns(r.OverheadRateAmount, r.OverheadRateCurrency, r.OverheadRatePerUnitID,
			r.LegacyOverheadRateValue, r.LegacyOverheadRatePerUnitID),
		MachineIDs: []string{},
		CreatedAt:  r.CreatedAt,
		UpdatedAt:  r.UpdatedAt,
	}
	if r.SetupTimeValue.Valid {
		op.SetupTime = &domain.QuantityValue{Value: r.SetupTimeValue.String, UnitID: r.SetupTimeUnitID.String, UnitType: r.SetupTimeUnitType.String}
	}
	if r.LaborTimeValue.Valid {
		op.LaborTime = &domain.RateValue{Value: r.LaborTimeValue.String, NumeratorUnitID: r.LaborTimeUnitID.String, DenominatorUnitID: r.LaborTimePerUnitID.String}
	}
	if r.MachineTimeValue.Valid {
		op.MachineTime = &domain.RateValue{Value: r.MachineTimeValue.String, NumeratorUnitID: r.MachineTimeUnitID.String, DenominatorUnitID: r.MachineTimePerUnitID.String}
	}
	return op
}

func (r *manufacturingMethodRepoImpl) GetOperationsByIDs(ctx context.Context, accountID string, ids []string) ([]*domain.MethodOperation, *apierror.APIError) {
	if len(ids) == 0 {
		return nil, nil
	}
	rows, err := r.queries.GetMethodOperationsByIDs(ctx, sqlc.GetMethodOperationsByIDsParams{AccountID: accountID, Ids: ids})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, apiErr
	}
	byID := make(map[string]*domain.MethodOperation, len(rows))
	found := make([]string, 0, len(rows))
	for _, row := range rows {
		byID[row.ID] = operationFromRow(row)
		found = append(found, row.ID)
	}
	if len(found) > 0 {
		machines, err := r.queries.ListOperationMachineIDs(ctx, found)
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, apiErr
		}
		for _, m := range machines {
			if op, ok := byID[m.OperationID]; ok {
				op.MachineIDs = append(op.MachineIDs, m.MachineID)
			}
		}
	}
	out := make([]*domain.MethodOperation, 0, len(rows))
	for _, id := range ids {
		if op, ok := byID[id]; ok {
			out = append(out, op)
			delete(byID, id)
		}
	}
	return out, nil
}

func operationMoneyParams(op *domain.MethodOperation) (laborAmount, laborCurrency, laborPerUnit, overheadAmount, overheadCurrency, overheadPerUnit gosql.NullString) {
	laborAmount, laborCurrency, laborPerUnit = unitPriceParams(op.LaborRate)
	overheadAmount, overheadCurrency, overheadPerUnit = unitPriceParams(op.OverheadRate)
	return
}

func (r *manufacturingMethodRepoImpl) InsertOperation(ctx context.Context, accountID, name string, op *domain.MethodOperation) *apierror.APIError {
	la, lc, lu, oa, oc, ou := operationMoneyParams(op)
	err := r.queries.InsertMethodOperation(ctx, sqlc.InsertMethodOperationParams{
		ID:                    op.ID,
		Name:                  name,
		LevelingFactor:        factorFromPercent(op.LevelingPercent),
		Allowances:            factorFromPercent(op.AllowancePercent),
		LaborRateID:           op.LaborRateID,
		LaborTimeID:           op.LaborTimeID,
		OverheadRateID:        op.OverheadRateID,
		LaborRateAmount:       la,
		LaborRateCurrency:     lc,
		LaborRatePerUnitID:    lu,
		OverheadRateAmount:    oa,
		OverheadRateCurrency:  oc,
		OverheadRatePerUnitID: ou,
		ScanningStationID:     mmNullString(op.ScanningStationID),
		DepartmentID:          mmNullString(op.WorkCenterID),
		AccountID:             accountID,
		ManufacturingMethodID: gosql.NullString{String: op.MethodID, Valid: true},
		Position:              op.Position,
		SequenceCode:          string(op.Sequence),
		TypeCode:              string(op.Type),
		SetupTimeQuantityID:   mmNullString(op.SetupTimeQuantityID),
		MachineTimeID:         mmNullString(op.MachineTimeID),
		WorkInstructions:      mmNullString(op.WorkInstructions),
	})
	return db.MapSQLError(err)
}

func (r *manufacturingMethodRepoImpl) UpdateOperation(ctx context.Context, accountID string, op *domain.MethodOperation) *apierror.APIError {
	la, lc, lu, oa, oc, ou := operationMoneyParams(op)
	err := r.queries.UpdateMethodOperation(ctx, sqlc.UpdateMethodOperationParams{
		ID:                    op.ID,
		AccountID:             accountID,
		LevelingFactor:        factorFromPercent(op.LevelingPercent),
		Allowances:            factorFromPercent(op.AllowancePercent),
		LaborRateID:           op.LaborRateID,
		LaborTimeID:           op.LaborTimeID,
		OverheadRateID:        op.OverheadRateID,
		LaborRateAmount:       la,
		LaborRateCurrency:     lc,
		LaborRatePerUnitID:    lu,
		OverheadRateAmount:    oa,
		OverheadRateCurrency:  oc,
		OverheadRatePerUnitID: ou,
		ScanningStationID:     mmNullString(op.ScanningStationID),
		DepartmentID:          mmNullString(op.WorkCenterID),
		SequenceCode:          string(op.Sequence),
		SetupTimeQuantityID:   mmNullString(op.SetupTimeQuantityID),
		MachineTimeID:         mmNullString(op.MachineTimeID),
		WorkInstructions:      mmNullString(op.WorkInstructions),
	})
	return db.MapSQLError(err)
}

func (r *manufacturingMethodRepoImpl) SetOperationPosition(ctx context.Context, accountID, id string, position int32) *apierror.APIError {
	return db.MapSQLError(r.queries.SetMethodOperationPosition(ctx, sqlc.SetMethodOperationPositionParams{ID: id, AccountID: accountID, Position: position}))
}

func (r *manufacturingMethodRepoImpl) SetOperationMachines(ctx context.Context, accountID, operationID string, machineIDs []string) *apierror.APIError {
	if err := r.queries.DeleteOperationMachines(ctx, operationID); err != nil {
		return db.MapSQLError(err)
	}
	for _, id := range machineIDs {
		if err := r.queries.InsertOperationMachine(ctx, sqlc.InsertOperationMachineParams{AccountID: accountID, OperationID: operationID, MachineID: id}); err != nil {
			return db.MapSQLError(err)
		}
	}
	return nil
}

func (r *manufacturingMethodRepoImpl) DeleteOperation(ctx context.Context, accountID string, op *domain.MethodOperation) *apierror.APIError {
	if err := r.queries.DeleteOperationMachines(ctx, op.ID); err != nil {
		return db.MapSQLError(err)
	}
	if err := r.queries.DeleteMethodOperationRow(ctx, sqlc.DeleteMethodOperationRowParams{ID: op.ID, AccountID: accountID}); err != nil {
		return db.MapSQLError(err)
	}
	rates := []string{op.LaborRateID, op.LaborTimeID, op.OverheadRateID}
	if op.MachineTimeID != nil {
		rates = append(rates, *op.MachineTimeID)
	}
	if err := r.queries.DeleteMethodRates(ctx, rates); err != nil {
		return db.MapSQLError(err)
	}
	if op.SetupTimeQuantityID != nil {
		return db.MapSQLError(r.queries.DeleteMethodQuantities(ctx, []string{*op.SetupTimeQuantityID}))
	}
	return nil
}

func (r *manufacturingMethodRepoImpl) CountOperationMaterials(ctx context.Context, operationID string) (int64, *apierror.APIError) {
	n, err := r.queries.CountOperationMaterials(ctx, gosql.NullString{String: operationID, Valid: true})
	return n, db.MapSQLError(err)
}

func (r *manufacturingMethodRepoImpl) OperationNameTaken(ctx context.Context, accountID, name string) (bool, *apierror.APIError) {
	n, err := r.queries.ExistsOperationNamed(ctx, sqlc.ExistsOperationNamedParams{AccountID: accountID, Name: name})
	return n > 0, db.MapSQLError(err)
}

// --- materials ---

func (r *manufacturingMethodRepoImpl) ListMaterials(ctx context.Context, params domain.ListMethodMembersParams) (*domain.ListMethodMaterialsResult, *apierror.APIError) {
	ctx, span := manufacturingMethodRepoTracer.Start(ctx, "repository.manufacturing_method.list_materials")
	defer span.End()

	methodID := gosql.NullString{String: params.MethodID, Valid: true}
	type row struct {
		id        string
		createdAt time.Time
	}
	var page []row
	var cursorDir *pagination.Direction
	if params.Cursor == nil {
		rows, err := r.queries.ListMethodMaterialIDs(ctx, sqlc.ListMethodMaterialIDsParams{ManufacturingMethodID: methodID, Limit: params.Limit + 1})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		for _, x := range rows {
			page = append(page, row{x.ID, x.CreatedAt})
		}
	} else {
		cur, err := pagination.DecodeStringCursor(*params.Cursor)
		if err != nil {
			return nil, apierror.NewParameterInvalidError("cursor", "Invalid pagination cursor.")
		}
		cursorDir = &cur.Direction
		if cur.Direction == pagination.DirectionBackward {
			rows, err := r.queries.ListMethodMaterialsBefore(ctx, sqlc.ListMethodMaterialsBeforeParams{
				ManufacturingMethodID: methodID, BeforeCreatedAt: cur.OccurredAt, BeforeID: cur.ID, Limit: params.Limit + 1,
			})
			if apiErr := db.MapSQLError(err); apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			for _, x := range rows {
				page = append(page, row{x.ID, x.CreatedAt})
			}
		} else {
			rows, err := r.queries.ListMethodMaterialsAfter(ctx, sqlc.ListMethodMaterialsAfterParams{
				ManufacturingMethodID: methodID, AfterCreatedAt: cur.OccurredAt, AfterID: cur.ID, Limit: params.Limit + 1,
			})
			if apiErr := db.MapSQLError(err); apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			for _, x := range rows {
				page = append(page, row{x.ID, x.CreatedAt})
			}
		}
	}
	page, pageInfo := pagination.BuildPageString(page, params.Limit, cursorDir,
		func(x row) time.Time { return x.createdAt }, func(x row) string { return x.id })
	ids := make([]string, len(page))
	for i, x := range page {
		ids[i] = x.id
	}
	materials, apiErr := r.GetMaterialsByIDs(ctx, params.AccountID, ids)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return &domain.ListMethodMaterialsResult{Materials: materials, PageInfo: pageInfo}, nil
}

func (r *manufacturingMethodRepoImpl) GetMaterialsByIDs(ctx context.Context, accountID string, ids []string) ([]*domain.MethodMaterial, *apierror.APIError) {
	if len(ids) == 0 {
		return nil, nil
	}
	rows, err := r.queries.GetMethodMaterialsByIDs(ctx, sqlc.GetMethodMaterialsByIDsParams{AccountID: accountID, Ids: ids})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, apiErr
	}
	byID := make(map[string]*domain.MethodMaterial, len(rows))
	for _, x := range rows {
		byID[x.ID] = &domain.MethodMaterial{
			ID:               x.ID,
			MethodID:         x.ManufacturingMethodID.String,
			ItemID:           x.ItemID,
			OperationID:      mmStringPtr(x.ProductionStepID),
			Quantity:         domain.QuantityValue{Value: x.QuantityValue, UnitID: x.QuantityUnitID, UnitType: x.QuantityUnitType},
			ScrapQuantity:    domain.QuantityValue{Value: x.ScrapQuantityValue, UnitID: x.ScrapQuantityUnitID, UnitType: x.ScrapQuantityUnitType},
			Supply:           constants.Supply(x.SupplyCode),
			WorkInstructions: mmStringPtr(x.Instructions),
			QuantityID:       x.QuantityID,
			ScrapQuantityID:  x.WasteQuantityID,
			CreatedAt:        x.CreatedAt,
			UpdatedAt:        x.UpdatedAt,
		}
	}
	out := make([]*domain.MethodMaterial, 0, len(rows))
	for _, id := range ids {
		if m, ok := byID[id]; ok {
			out = append(out, m)
			delete(byID, id)
		}
	}
	return out, nil
}

func (r *manufacturingMethodRepoImpl) InsertMaterial(ctx context.Context, m *domain.MethodMaterial) *apierror.APIError {
	return db.MapSQLError(r.queries.InsertMethodMaterial(ctx, sqlc.InsertMethodMaterialParams{
		ID:                    m.ID,
		ItemID:                m.ItemID,
		QuantityID:            m.QuantityID,
		WasteQuantityID:       m.ScrapQuantityID,
		ProductionStepID:      mmNullString(m.OperationID),
		Instructions:          mmNullString(m.WorkInstructions),
		ManufacturingMethodID: gosql.NullString{String: m.MethodID, Valid: true},
		SupplyCode:            string(m.Supply),
	}))
}

func (r *manufacturingMethodRepoImpl) UpdateMaterial(ctx context.Context, m *domain.MethodMaterial) *apierror.APIError {
	return db.MapSQLError(r.queries.UpdateMethodMaterial(ctx, sqlc.UpdateMethodMaterialParams{
		ID:                    m.ID,
		ManufacturingMethodID: gosql.NullString{String: m.MethodID, Valid: true},
		ItemID:                m.ItemID,
		QuantityID:            m.QuantityID,
		WasteQuantityID:       m.ScrapQuantityID,
		ProductionStepID:      mmNullString(m.OperationID),
		Instructions:          mmNullString(m.WorkInstructions),
		SupplyCode:            string(m.Supply),
	}))
}

func (r *manufacturingMethodRepoImpl) DeleteMaterial(ctx context.Context, m *domain.MethodMaterial) *apierror.APIError {
	if err := r.queries.DeleteMethodMaterialRow(ctx, sqlc.DeleteMethodMaterialRowParams{
		ID: m.ID, ManufacturingMethodID: gosql.NullString{String: m.MethodID, Valid: true},
	}); err != nil {
		return db.MapSQLError(err)
	}
	return db.MapSQLError(r.queries.DeleteMethodQuantities(ctx, []string{m.QuantityID, m.ScrapQuantityID}))
}

// --- value rows ---

func (r *manufacturingMethodRepoImpl) InsertRate(ctx context.Context, id string, v domain.RateValue) *apierror.APIError {
	return db.MapSQLError(r.queries.InsertMethodRate(ctx, sqlc.InsertMethodRateParams{
		ID: id, Value: v.Value, NumeratorUnitID: v.NumeratorUnitID, DenominatorUnitID: v.DenominatorUnitID,
	}))
}

func (r *manufacturingMethodRepoImpl) InsertQuantity(ctx context.Context, id string, v domain.QuantityValue) *apierror.APIError {
	return db.MapSQLError(r.queries.InsertMethodQuantity(ctx, sqlc.InsertMethodQuantityParams{ID: id, Value: v.Value, UnitID: v.UnitID}))
}

func (r *manufacturingMethodRepoImpl) DeleteRates(ctx context.Context, ids []string) *apierror.APIError {
	if len(ids) == 0 {
		return nil
	}
	return db.MapSQLError(r.queries.DeleteMethodRates(ctx, ids))
}

func (r *manufacturingMethodRepoImpl) DeleteQuantities(ctx context.Context, ids []string) *apierror.APIError {
	if len(ids) == 0 {
		return nil
	}
	return db.MapSQLError(r.queries.DeleteMethodQuantities(ctx, ids))
}

// --- execution projection ---

func (r *manufacturingMethodRepoImpl) InsertProduction(ctx context.Context, id, itemID, quantityID, stepID string) *apierror.APIError {
	return db.MapSQLError(r.queries.InsertMethodProduction(ctx, sqlc.InsertMethodProductionParams{
		ID: id, ItemID: itemID, QuantityID: quantityID, ProductionStepID: gosql.NullString{String: stepID, Valid: true},
	}))
}

func (r *manufacturingMethodRepoImpl) RemoveProjection(ctx context.Context, stepID string) *apierror.APIError {
	step := gosql.NullString{String: stepID, Valid: true}
	quantityIDs, err := r.queries.ListProductionQuantityIDsByStep(ctx, step)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return apiErr
	}
	if err := r.queries.DeleteProductionsOfStep(ctx, step); err != nil {
		return db.MapSQLError(err)
	}
	if apiErr := r.DeleteQuantities(ctx, quantityIDs); apiErr != nil {
		return apiErr
	}
	if err := r.queries.ClearStepLinksOfStep(ctx, stepID); err != nil {
		return db.MapSQLError(err)
	}
	if err := r.queries.ClearStepLinksFromStep(ctx, stepID); err != nil {
		return db.MapSQLError(err)
	}
	return db.MapSQLError(r.queries.ClearMachinesFromStep(ctx, step))
}

func (r *manufacturingMethodRepoImpl) ConnectSteps(ctx context.Context, upstreamID, downstreamID string) *apierror.APIError {
	return db.MapSQLError(r.queries.ConnectMethodSteps(ctx, sqlc.ConnectMethodStepsParams{DownstreamID: downstreamID, UpstreamID: upstreamID}))
}

func (r *manufacturingMethodRepoImpl) AssignMachinesToStep(ctx context.Context, accountID, stepID string, machineIDs []string) *apierror.APIError {
	if len(machineIDs) == 0 {
		return nil
	}
	return db.MapSQLError(r.queries.SetMachinesStep(ctx, sqlc.SetMachinesStepParams{
		StepID: gosql.NullString{String: stepID, Valid: true}, Ids: machineIDs, AccountID: accountID,
	}))
}
