package repository

import (
	"context"

	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
)

// customerSearchMatchCap is the most customers a search may match for the list to be read from its
// matches; a search matching more is dense enough that walking the customers in list order and testing
// each meets a page sooner.
const customerSearchMatchCap = 1000

// customerSearchMatches returns the accounts of the owner's customers that a search (a LIKE pattern)
// matches on name, alias, number, notes, or support email. ok is false past customerSearchMatchCap.
//
// Each column is read on its own: tested inside the list's walk, the search reads every customer's
// account and branding rows before a rare match fills the page. Names and support emails are on the
// customer's own account, shared by every owner, so their keys are scanned whole and the matches kept
// only where the owner has the customer. Notes are TEXT and in no key, so the relation table is scanned
// once for the relation's three columns, which beats fetching each of the owner's relations by key.
func customerSearchMatches(ctx context.Context, q sqlc.DBTX, ownerID, like string) (ids []string, ok bool, err error) {
	ids, err = selectStrings(ctx, q, `SELECT m.id FROM (
    SELECT na.id FROM account na FORCE INDEX (account_name_idx) WHERE na.name LIKE ?
    UNION
    SELECT mar.counterparty_account_id FROM account_relation mar FORCE INDEX (PRIMARY)
    WHERE mar.owner_account_id = ? AND mar.account_relation_role_code = 'customer'
        AND (mar.alias LIKE ? OR mar.external_number LIKE ? OR mar.notes LIKE ?)
    UNION
    SELECT mab.owner_account_id FROM account_branding mab FORCE INDEX (account_branding_support_email_idx) WHERE mab.support_email LIKE ?
) m
JOIN account_relation cr FORCE INDEX (account_relation_owner_account_id_counterparty_account_id_ac_key)
    ON cr.owner_account_id = ? AND cr.counterparty_account_id = m.id AND cr.account_relation_role_code = 'customer'
LIMIT ?`, like, ownerID, like, like, like, like, ownerID, customerSearchMatchCap+1)
	if err != nil || len(ids) > customerSearchMatchCap {
		return nil, false, err
	}
	return ids, true, nil
}
