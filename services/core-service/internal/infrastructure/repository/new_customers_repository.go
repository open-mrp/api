package repository

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/db"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/pagination"
	"github.com/open-mrp/api/shared/tracing"
)

func (r *salesReportRepoImpl) BuyerSummariesReady(ctx context.Context) (bool, *apierror.APIError) {
	sync, apiErr := NewSalesFactRepo(r.queries).GetBuyerSummarySync(ctx)
	if apiErr != nil {
		return false, apiErr
	}
	return sync.LastCompletedAt != nil, nil
}

// GetNewCustomers reads the customers added in the window from account_relation and their first order and
// lifetime sales from sales_buyer_summary, so the cost is the window's customers, not their sales history.
// A customer with no summary has never ordered and is left out, as the legacy report left it out.
func (r *salesReportRepoImpl) GetNewCustomers(ctx context.Context, params domain.ListNewCustomersParams) (*domain.NewCustomerPage, *apierror.APIError) {
	ctx, span := salesReportRepoTracer.Start(ctx, "repository.sales_report.get_new_customers")
	defer span.End()

	cursor, apiErr := keysetCursor(params.Cursor)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	var sb strings.Builder
	args := []any{params.AccountID, params.StartsAt, params.EndsAt}
	sb.WriteString(`SELECT ar.counterparty_account_id, ar.external_number, COALESCE(NULLIF(ar.alias, ''), a.name, ''), ag.name,
    g.locality, g.state, u.name, CAST(s.total_invoiced AS CHAR), s.first_ordered_at, ar.created_at
FROM account_relation ar
JOIN sales_buyer_summary s ON s.account_id = ar.owner_account_id AND s.buyer_account_id = ar.counterparty_account_id
JOIN account a ON a.id = ar.counterparty_account_id
LEFT JOIN account_group ag ON ag.id = ar.account_group_id
LEFT JOIN address ad ON ad.id = ar.default_shipping_address_id
LEFT JOIN geolocation g ON g.id = ad.geolocation_id
LEFT JOIN account_user au ON au.id = ar.default_sales_rep_id
LEFT JOIN ` + "`user`" + ` u ON u.id = au.user_id
WHERE ar.owner_account_id = ? AND ar.account_relation_role_code = 'customer'
  AND ar.created_at >= ? AND ar.created_at <= ?`)
	if len(params.CustomerGroupIDs) > 0 {
		in := placeholders(len(params.CustomerGroupIDs))
		sb.WriteString(` AND (ar.account_group_id IN (` + in + `) OR EXISTS (SELECT 1 FROM account_relation_price_group pg
    WHERE pg.account_relation_id = ar.id AND pg.account_group_id IN (` + in + `)))`)
		args = append(args, stringsToAny(params.CustomerGroupIDs)...)
		args = append(args, stringsToAny(params.CustomerGroupIDs)...)
	}
	if len(params.SalesRepIDs) > 0 {
		sb.WriteString(` AND ar.default_sales_rep_id IN (` + placeholders(len(params.SalesRepIDs)) + `)`)
		args = append(args, stringsToAny(params.SalesRepIDs)...)
	}
	seek, seekArgs, order := keysetPredicate(cursor, "s.first_ordered_at", "ar.counterparty_account_id")
	if seek != "" {
		sb.WriteString(" AND " + seek)
		args = append(args, seekArgs...)
	}
	sb.WriteString(" ORDER BY s.first_ordered_at " + order + ", ar.counterparty_account_id " + order + " LIMIT ?")
	args = append(args, params.Limit+1)

	rows, err := r.queries.DB().QueryContext(ctx, sb.String(), args...)
	if err != nil {
		return nil, tracing.Trace(span, db.MapSQLError(err))
	}
	defer func() { _ = rows.Close() }()
	page := &domain.NewCustomerPage{}
	for rows.Next() {
		var (
			c                        domain.NewCustomer
			group, locality, st, rep sql.NullString
			total                    sql.NullString
		)
		if err := rows.Scan(&c.CustomerID, &c.CustomerNumber, &c.CustomerName, &group, &locality, &st, &rep, &total, &c.FirstOrderedAt, &c.CustomerCreatedAt); err != nil {
			return nil, tracing.Trace(span, db.MapSQLError(err))
		}
		c.CustomerGroupName = nullStringPtr(group)
		c.SalesRepName = nullStringPtr(rep)
		c.Location = joinLocation(locality, st)
		c.TotalInvoiced = exactDecimal(total)
		page.Customers = append(page.Customers, c)
	}
	if err := rows.Err(); err != nil {
		return nil, tracing.Trace(span, db.MapSQLError(err))
	}
	page.Customers, page.PageInfo = pagination.BuildPageString(page.Customers, params.Limit, cursorDirection(cursor),
		func(c domain.NewCustomer) time.Time { return c.FirstOrderedAt },
		func(c domain.NewCustomer) string { return c.CustomerID })
	return page, nil
}

// joinLocation is "locality, state" with any blank part left out, nil when both are, as the legacy report showed it.
func joinLocation(parts ...sql.NullString) *string {
	var kept []string
	for _, p := range parts {
		if p.Valid && p.String != "" {
			kept = append(kept, p.String)
		}
	}
	if len(kept) == 0 {
		return nil
	}
	s := strings.Join(kept, ", ")
	return &s
}
