package repository

import (
	gosql "database/sql"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/apikit/pagination"
)

const scanningStationListColumns = `ss.id, ss.name, ss.notes, ss.scanning_station_type_code, ss.label_size_code, ss.label_type_code, ` +
	`ss.material_check_required, ss.department_id, d.name, d.created_at, d.updated_at, ss.account_id, ss.created_at, ss.updated_at`

// scanningStationListFilter is one page of the scanning station listing.
type scanningStationListFilter struct {
	accountID string
	// search is a LIKE pattern, already escaped and wrapped; empty for none.
	search        string
	departmentIDs []string
	cursor        *pagination.StringCursor
	limit         int32
}

// buildScanningStationListQuery emits only the predicates the caller supplied, so an absent filter costs nothing. It
// pages on the account's (account_id, created_at, id) index; name and department filters are residual on it, since
// an account holds a few dozen stations.
func buildScanningStationListQuery(f scanningStationListFilter) (string, []any) {
	var lf listFilter
	lf.add("ss.account_id = ?", f.accountID)
	if f.search != "" {
		lf.add("ss.name LIKE ?", f.search)
	}
	lf.in("ss.department_id", f.departmentIDs)
	order := lf.keyset(f.cursor, "ss.created_at", "ss.id")
	args := append(lf.args, f.limit)
	return "SELECT " + scanningStationListColumns + "\nFROM scanning_station ss\nLEFT JOIN department d ON d.id = ss.department_id" +
		lf.whereSQL() + "\nORDER BY " + order + "\nLIMIT ?", args
}

// scanScanningStationListRows reads rows in scanningStationListColumns order.
func scanScanningStationListRows(rows *gosql.Rows) ([]*domain.ScanningStation, error) {
	var out []*domain.ScanningStation
	for rows.Next() {
		var (
			ss                                    domain.ScanningStation
			typeCode                              string
			notes, labelSize, labelType, deptName gosql.NullString
			materialCheck                         bool
			deptCreatedAt, deptUpdatedAt          gosql.NullTime
		)
		if err := rows.Scan(&ss.ID, &ss.Name, &notes, &typeCode, &labelSize, &labelType, &materialCheck, &ss.DepartmentID,
			&deptName, &deptCreatedAt, &deptUpdatedAt, &ss.AccountID, &ss.CreatedAt, &ss.UpdatedAt); err != nil {
			return nil, err
		}
		ss.Type = constants.ScanningStationType(typeCode)
		ss.OperatorRequirement = boolToOperatorRequirement(materialCheck)
		ss.Notes = nullStringPtr(notes)
		ss.LabelSizeCode = nullStringPtr(labelSize)
		ss.LabelTypeCode = nullStringPtr(labelType)
		ss.DepartmentName = deptName.String
		if deptCreatedAt.Valid {
			ss.DepartmentCreatedAt = &deptCreatedAt.Time
		}
		if deptUpdatedAt.Valid {
			ss.DepartmentUpdatedAt = &deptUpdatedAt.Time
		}
		out = append(out, &ss)
	}
	return out, rows.Err()
}
