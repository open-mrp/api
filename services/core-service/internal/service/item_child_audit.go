package service

import (
	"context"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/audit"
	"github.com/open-mrp/api/shared/decimalfmt"
	apierror "github.com/open-mrp/apikit/apierror"
)

// itemChildAudit snapshots an item's rates and attributes around a write. Both load only on
// include, so a product or part diff cannot see them on its own.
type itemChildAudit struct {
	accountID string
	item      *domain.Item
	rates     []rateSnapshot
}

type rateSnapshot struct {
	field  string
	rateID string
	before *domain.Rate
}

// rateAuditValue is the client-visible shape of a rate in an audit diff.
type rateAuditValue struct {
	Value             string `json:"value"`
	NumeratorUnitID   string `json:"numerator_unit_id"`
	DenominatorUnitID string `json:"denominator_unit_id"`
}

// snapshotItemChildren reads the item's attributes and, when withRates, its unit_price and unit_cost rates.
func snapshotItemChildren(ctx context.Context, repos domain.RepoFactory, accountID, itemID string, withRates bool) (*itemChildAudit, *apierror.APIError) {
	item, apiErr := repos.NewItemRepo().Get(ctx, domain.GetItemParams{AccountID: accountID, ItemID: itemID, Includes: []string{"attributes"}})
	if apiErr != nil {
		return nil, apiErr
	}
	a := &itemChildAudit{accountID: accountID, item: item}
	if !withRates {
		return a, nil
	}
	for _, r := range []rateSnapshot{{field: "unit_price", rateID: item.UnitValueID}, {field: "unit_cost", rateID: item.UnitCostID}} {
		if r.rateID == "" {
			continue
		}
		before, apiErr := repos.NewRateRepo().Get(ctx, r.rateID)
		if apiErr != nil {
			return nil, apiErr
		}
		r.before = before
		a.rates = append(a.rates, r)
	}
	return a, nil
}

// changes re-reads the snapshotted children and diffs them against the snapshot.
func (a *itemChildAudit) changes(ctx context.Context, repos domain.RepoFactory) ([]audit.FieldChange, *apierror.APIError) {
	after, apiErr := repos.NewItemRepo().Get(ctx, domain.GetItemParams{AccountID: a.accountID, ItemID: a.item.ID, Includes: []string{"attributes"}})
	if apiErr != nil {
		return nil, apiErr
	}
	changes := audit.ComputeChanges(a.item, after, "Attributes")
	for _, r := range a.rates {
		now, apiErr := repos.NewRateRepo().Get(ctx, r.rateID)
		if apiErr != nil {
			return nil, apiErr
		}
		old, cur := rateAuditOf(r.before), rateAuditOf(now)
		if old != cur {
			changes = append(changes, audit.NewFieldChange(r.field, old, cur))
		}
	}
	return changes, nil
}

func rateAuditOf(r *domain.Rate) rateAuditValue {
	return rateAuditValue{Value: decimalfmt.Rate(r.Value), NumeratorUnitID: r.NumeratorUnitID, DenominatorUnitID: r.DenominatorUnitID}
}
