package resourceregistry

import (
	"context"
	"github.com/open-mrp/apikit/include"

	"github.com/open-mrp/api/services/api-gateway/internal/resourceloaders"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
)

func init() {
	include.Register(&include.Definition{
		ObjectType: constants.ObjectTypeContactMatch,
		Load:       resourceloaders.LoadContactMatches,
		Subs: []include.SubField{
			// account_user is eagerly built by the find-by-email service (the contact lives in a related account that the account-scoped account_user batch-get can't reach)
			// and stashed in LoadMeta. Populate gates it on ?include=account_user; ExtractRefs then lets the resolver recurse into account_user.user/role/work_center via the
			// AccountUser definition's own subs (which load by id from global tables).
			{
				Key:         "account_user",
				Target:      constants.ObjectTypeAccountUser,
				Cardinality: include.CardinalityOnePtr,
				ExtractRefs: extractAccountUserRefFromContactMatch,
				Populate:    populateAccountUserOnContactMatch,
			},
			// account is lazy: the service stashes the contact's account_id (generic across customer/supplier/partner/self) and LoadAccounts hydrates it on ?include=account.
			{
				Key:         "account",
				Target:      constants.ObjectTypeAccount,
				Cardinality: include.CardinalityOnePtr,
				ExtractIDs:  extractAccountIDFromContactMatch,
				Populate:    populateAccountOnContactMatch,
			},
		},
	})
}

func populateAccountUserOnContactMatch(ctx context.Context, parent any, _ map[string]any) {
	cm := parent.(*apiresource.ContactMatch)
	v, ok := include.GetLoadMeta(ctx).Get(constants.ObjectTypeContactMatch, cm.ID, "account_user")
	if !ok {
		return
	}
	cm.AccountUser = v.(*apiresource.AccountUser)
}

func extractAccountUserRefFromContactMatch(_ context.Context, parent any) []any {
	cm := parent.(*apiresource.ContactMatch)
	if cm.AccountUser == nil {
		return nil
	}
	return []any{cm.AccountUser}
}

func extractAccountIDFromContactMatch(ctx context.Context, parent any) []string {
	cm := parent.(*apiresource.ContactMatch)
	id, _ := include.GetLoadMeta(ctx).GetString(constants.ObjectTypeContactMatch, cm.ID, "account_id")
	if id == "" {
		return nil
	}
	return []string{id}
}

func populateAccountOnContactMatch(ctx context.Context, parent any, loaded map[string]any) {
	cm := parent.(*apiresource.ContactMatch)
	id, _ := include.GetLoadMeta(ctx).GetString(constants.ObjectTypeContactMatch, cm.ID, "account_id")
	if id == "" {
		return
	}
	if v, ok := loaded[id]; ok {
		cm.Account = v.(*apiresource.Account)
	}
}
