package repository

import (
	gosql "database/sql"
	"strings"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/apikit/db"
	"github.com/open-mrp/apikit/pagination"
)

// accountUserListColumns is the SELECT list scanAccountUserListRows reads, in order.
const accountUserListColumns = `au.id, au.user_id, u.name, u.email, u.username, u.image_url, u.email_verified, ` +
	`au.role_id, au.department_id, au.status_code, au.last_used_at, au.invited_at, au.is_commission_eligible, ` +
	`au.created_at, au.updated_at`

// salesRepRoleExists holds when the member's role is a sales-rep role, which makes them commission eligible whatever
// their own flag says.
const salesRepRoleExists = `EXISTS (SELECT 1 FROM role r_comm WHERE r_comm.id = au.role_id AND r_comm.role_type_code = 'sales_rep')`

// buildAccountUserListPage chooses one page of an account's members from account_user alone. statuses[] rides the
// status key; every other filter is residual. Only the filters the caller gave are emitted. Forward pages older
// (DESC), backward pages newer.
func buildAccountUserListPage(params domain.ListAccountUsersParams, dir pagination.Direction, cursor *pagination.StringCursor, limit int32) keysetPage {
	page := keysetPage{
		table: "account_user", alias: "au", sortColumn: "created_at",
		createdIndex: "account_user_account_created_idx",
		filters: []keysetFilter{
			{column: "au.status_code", index: "account_user_account_id_status_code_created_at_id_idx", values: enumStrings(params.Statuses)},
		},
		where: []string{"au.account_id = ?"},
		args:  []any{params.AccountID},
		desc:  dir != pagination.DirectionBackward,
		limit: limit,
	}
	if params.Created.After != nil {
		page.where, page.args = append(page.where, "au.created_at >= ?"), append(page.args, *params.Created.After)
	}
	if params.Created.Before != nil {
		page.where, page.args = append(page.where, "au.created_at < ?"), append(page.args, *params.Created.Before)
	}
	if params.RoleType != nil {
		page.where = append(page.where, "EXISTS (SELECT 1 FROM role r WHERE r.id = au.role_id AND r.role_type_code = ?)")
		page.args = append(page.args, *params.RoleType)
	}
	if params.CommissionEligibility != nil {
		if *params.CommissionEligibility == constants.CommissionEligibilityEligible {
			page.where = append(page.where, "(au.is_commission_eligible = true OR "+salesRepRoleExists+")")
		} else {
			page.where = append(page.where, "(au.is_commission_eligible = false AND NOT "+salesRepRoleExists+")")
		}
	}
	if params.Query != nil && strings.TrimSpace(*params.Query) != "" {
		like := "%" + db.EscapeLike(strings.TrimSpace(*params.Query)) + "%"
		page.where = append(page.where, "EXISTS (SELECT 1 FROM `user` u_q WHERE u_q.id = au.user_id AND (u_q.name LIKE ? OR u_q.email LIKE ? OR u_q.username LIKE ?))")
		page.args = append(page.args, like, like, like)
	}
	if cursor != nil {
		cmp := "<"
		if dir == pagination.DirectionBackward {
			cmp = ">"
		}
		page.where = append(page.where, "(au.created_at "+cmp+" ? OR (au.created_at = ? AND au.id "+cmp+" ?))")
		page.args = append(page.args, cursor.OccurredAt, cursor.OccurredAt, cursor.ID)
	}
	return page
}

// buildAccountUserListQuery wraps a page from buildAccountUserListPage in the member and user columns, in list order.
func buildAccountUserListQuery(page keysetPage, dir pagination.Direction) (string, []any) {
	orderBy := " ORDER BY au.created_at DESC, au.id DESC"
	if dir == pagination.DirectionBackward {
		orderBy = " ORDER BY au.created_at ASC, au.id ASC"
	}
	pageSQL, args := page.sql()
	return "SELECT STRAIGHT_JOIN " + accountUserListColumns + " FROM (" + pageSQL + ") page" +
		" JOIN account_user au ON au.id = page.id JOIN `user` u ON u.id = au.user_id" + orderBy, args
}

// scanAccountUserListRows reads rows selecting accountUserListColumns.
func scanAccountUserListRows(rows *gosql.Rows) ([]*domain.AccountUserDetail, error) {
	var out []*domain.AccountUserDetail
	for rows.Next() {
		var d domain.AccountUserDetail
		var name, email, username, imageURL, roleID, departmentID gosql.NullString
		var emailVerified, lastUsedAt, invitedAt gosql.NullTime
		var status string
		if err := rows.Scan(&d.ID, &d.UserID, &name, &email, &username, &imageURL, &emailVerified,
			&roleID, &departmentID, &status, &lastUsedAt, &invitedAt, &d.IsCommissionEligible,
			&d.CreatedAt, &d.UpdatedAt); err != nil {
			return nil, err
		}
		d.Name = db.StringFromNullString(name)
		d.Email = db.StringFromNullString(email)
		d.Username = db.StringFromNullString(username)
		d.ImageURL = db.StringFromNullString(imageURL)
		d.EmailVerified = emailVerified.Valid
		d.RoleID = db.StringFromNullString(roleID)
		d.DepartmentID = db.StringFromNullString(departmentID)
		d.StatusCode = constants.AccountUserStatus(status)
		d.LastUsedAt = db.TimeFromNullTime(lastUsedAt)
		d.InvitedAt = db.TimeFromNullTime(invitedAt)
		out = append(out, &d)
	}
	return out, rows.Err()
}
