package repository

import (
	gosql "database/sql"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/apikit/db"
	"github.com/open-mrp/apikit/pagination"
)

const machineListColumns = `m.id, m.name, m.serial_number, m.notes, m.department_id, d.name, d.created_at, d.updated_at, ` +
	`m.production_step_id, m.is_active, m.created_at, m.updated_at`

// machineListFilter is one page of the machine listing.
type machineListFilter struct {
	accountID string
	// search is the search term; empty for none.
	search        string
	departmentIDs []string
	isActive      *bool
	cursor        *pagination.StringCursor
	limit         int32
}

// buildMachineListQuery emits only the predicates the caller supplied. Without a search it pages on the account's
// (account_id, created_at, id) index. A search matches names starting with the term and ranks the machine named
// exactly the term first (tier 0), then the other matches (tier 1), newest first within a tier; the tier leads the
// cursor.
func buildMachineListQuery(f machineListFilter) (string, []any) {
	var lf listFilter
	lf.add("m.account_id = ?", f.accountID)
	lf.in("m.department_id", f.departmentIDs)
	if f.isActive != nil {
		lf.add("m.is_active = ?", *f.isActive)
	}

	tierSQL, order := "0", ""
	var selectArgs []any
	if f.search == "" {
		order = lf.keyset(f.cursor, "m.created_at", "m.id")
	} else {
		tierSQL = "(m.name <> ?)"
		selectArgs = []any{f.search}
		lf.add("m.name LIKE ?", db.EscapeLike(f.search)+"%")
		order = lf.tieredKeyset(f.cursor, tierSQL, f.search, "m.created_at", "m.id")
	}

	args := append(append(selectArgs, lf.args...), f.limit)
	return "SELECT " + machineListColumns + ", " + tierSQL + " AS match_tier\nFROM machine m\nJOIN department d ON d.id = m.department_id" +
		lf.whereSQL() + "\nORDER BY " + order + "\nLIMIT ?", args
}

// tieredKeyset pages on (tier, atCol, idCol): tier ascending, then newest first, all reversed after a backward
// cursor. tierSQL takes one argument, tierArg, and is selected AS match_tier. A cursor without a tier is from tier 0.
func (f *listFilter) tieredKeyset(cursor *pagination.StringCursor, tierSQL string, tierArg any, atCol, idCol string) string {
	if cursor == nil {
		return "match_tier ASC, " + atCol + " DESC, " + idCol + " DESC"
	}
	tier := 0
	if cursor.MatchTier != nil {
		tier = *cursor.MatchTier
	}
	if cursor.Direction == pagination.DirectionBackward {
		f.add("("+tierSQL+" < ? OR ("+tierSQL+" = ? AND ("+atCol+" > ? OR ("+atCol+" = ? AND "+idCol+" > ?))))",
			tierArg, tier, tierArg, tier, cursor.OccurredAt, cursor.OccurredAt, cursor.ID)
		return "match_tier DESC, " + atCol + " ASC, " + idCol + " ASC"
	}
	f.add("("+tierSQL+" > ? OR ("+tierSQL+" = ? AND ("+atCol+" < ? OR ("+atCol+" = ? AND "+idCol+" < ?))))",
		tierArg, tier, tierArg, tier, cursor.OccurredAt, cursor.OccurredAt, cursor.ID)
	return "match_tier ASC, " + atCol + " DESC, " + idCol + " DESC"
}

// machineListRow is one row of buildMachineListQuery.
type machineListRow struct {
	machine *domain.Machine
	tier    int32
}

// scanMachineListRows reads rows in machineListColumns order, then the match tier.
func scanMachineListRows(rows *gosql.Rows) ([]machineListRow, error) {
	var out []machineListRow
	for rows.Next() {
		var (
			m                            domain.Machine
			notes, stepID                gosql.NullString
			deptID, deptName             string
			deptCreatedAt, deptUpdatedAt gosql.NullTime
			isActive                     bool
			tier                         int32
		)
		if err := rows.Scan(&m.ID, &m.Name, &m.SerialNumber, &notes, &deptID, &deptName, &deptCreatedAt, &deptUpdatedAt,
			&stepID, &isActive, &m.CreatedAt, &m.UpdatedAt, &tier); err != nil {
			return nil, err
		}
		m.Notes = nullStringPtr(notes)
		m.ProductionStepID = nullStringPtr(stepID)
		m.DepartmentID, m.DepartmentName = &deptID, &deptName
		if deptCreatedAt.Valid {
			m.DepartmentCreatedAt = &deptCreatedAt.Time
		}
		if deptUpdatedAt.Valid {
			m.DepartmentUpdatedAt = &deptUpdatedAt.Time
		}
		m.Status = constants.ArchiveStatusFromIsActive(isActive)
		out = append(out, machineListRow{machine: &m, tier: tier})
	}
	return out, rows.Err()
}
