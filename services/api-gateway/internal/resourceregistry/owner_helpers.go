package resourceregistry

import (
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
)

// buildOwnerShell constructs the public Owner projection of a parent
// resource's account_id. Empty accountID -> system-owned; otherwise the
// owner advertises type=account but leaves Account nil. A full Account is
// attached only when ?include[]=<parent>.owner.account also fires — at
// which point the populateOwnerAccountOn<Resource> SubField writes the
// loaded *Account in (looking up the FK in LoadMeta, not on the Owner).
//
// We DO NOT emit a stub Account containing just {id, object}. The Account
// schema declares name/created_at/updated_at as required, and a stub would
// fail OpenAPI validation. More importantly: any field on a stub would be
// "hallucinated" (not loaded from the DB), which violates the framework's
// no-hallucination rule.
func buildOwnerShell(accountID string) *apiresource.Owner {
	if accountID == "" {
		return &apiresource.Owner{
			Object: constants.ObjectTypeOwner,
			Type:   constants.OwnerTypeSystem,
		}
	}
	return &apiresource.Owner{
		Object: constants.ObjectTypeOwner,
		Type:   constants.OwnerTypeAccount,
	}
}
