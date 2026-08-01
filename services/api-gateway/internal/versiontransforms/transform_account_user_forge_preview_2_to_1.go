// Package versiontransforms registers the version.Transformer chain that the api-gateway uses to keep older API versions working. The package is blank-imported from cmd/run.go so its init() fires before the routers are built.
package versiontransforms

import (
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/version"
)

func init() {
	version.Register(&accountUserForgePreview2To1{})
}

// accountUserForgePreview2To1 downgrades account_user payloads from 1.0.forge-preview.2 to 1.0.forge-preview.1.
//
// preview.2 made `user` an expandable full User object and removed the profile fields (name, email, username, image_url) that account_user had duplicated from the user. preview.1 carried those fields on account_user itself and exposed `user` as an always-present entity reference. The downgrade hoists the profile fields off the expanded user — forced onto account_user root responses via ForcedIncludes — and rebuilds the entity reference. Account users embedded in other resources (sales_rep, responsible_user, shipped_by) are downgraded to the preview.1 shape too, but their profile fields are null unless the caller expanded the nested user.
type accountUserForgePreview2To1 struct{}

func (t *accountUserForgePreview2To1) FromVersion() version.APIVersion {
	return version.V1_0_Forge_Preview2
}

func (t *accountUserForgePreview2To1) ToVersion() version.APIVersion {
	return version.V1_0_Forge_Preview1
}

func (t *accountUserForgePreview2To1) ObjectTypes() []constants.ObjectType {
	return []constants.ObjectType{
		constants.ObjectTypeAccountUser,
		constants.ObjectTypeCustomer,
		constants.ObjectTypeTerritory,
		constants.ObjectTypeTransaction,
		constants.ObjectTypeTransactionSummary,
		constants.ObjectTypeProductionRun,
		constants.ObjectTypeShipment,
		constants.ObjectTypeSettlement,
		constants.ObjectTypePurchaseOrder,
		constants.ObjectTypeContactMatch,
		constants.ObjectTypeProductionScheduleWeekRelease,
	}
}

func (t *accountUserForgePreview2To1) ForcedIncludes(objectType constants.ObjectType) []string {
	switch objectType {
	case constants.ObjectTypeAccountUser:
		return []string{"user"}
	case constants.ObjectTypeTransaction, constants.ObjectTypeProductionRun:
		// preview.1 inlined responsible_user on these resources without an include; force the expansion (and its user, so the profile fields can be hoisted) to keep preview.1 responses populated.
		return []string{"responsible_user", "responsible_user.user"}
	case constants.ObjectTypeCustomer:
		// Nested keys apply only when the caller requested the parent path (defaults.sales_rep), restoring the profile fields preview.1 carried on the expanded sales rep.
		return []string{"defaults.sales_rep.user"}
	case constants.ObjectTypeTerritory:
		return []string{"sales_rep.user"}
	case constants.ObjectTypeSettlement:
		return []string{"responsible_user.user"}
	case constants.ObjectTypeShipment:
		return []string{"shipped_by.user"}
	case constants.ObjectTypeContactMatch:
		return []string{"account_user.user"}
	}
	return nil
}

func (t *accountUserForgePreview2To1) Transform(_ constants.ObjectType, data map[string]any) map[string]any {
	downgradeAccountUsersIn(data)
	return data
}

func (t *accountUserForgePreview2To1) TransformRequest(_ constants.ObjectType, data map[string]any) map[string]any {
	// Request shapes did not change between preview.1 and preview.2.
	return data
}

// downgradeAccountUsersIn walks the payload and rewrites every embedded account_user object in place — single resources, list envelopes, and account users nested in other resources.
func downgradeAccountUsersIn(node any) {
	switch v := node.(type) {
	case map[string]any:
		if v["object"] == string(constants.ObjectTypeAccountUser) {
			downgradeAccountUser(v)
		}
		for _, child := range v {
			downgradeAccountUsersIn(child)
		}
	case []any:
		for _, child := range v {
			downgradeAccountUsersIn(child)
		}
	}
}

// downgradeAccountUser converts one account_user object from the preview.2 shape to the preview.1 shape: profile fields hoisted to the top level and `user` demoted from a full User object to an entity reference.
func downgradeAccountUser(au map[string]any) {
	user, _ := au["user"].(map[string]any)

	var name, email, username, imageURL any
	if user != nil {
		name = user["name"]
		email = user["email"]
		username = user["username"]
		imageURL = user["image_url"]
	}

	au["name"] = name
	au["email"] = email
	au["username"] = username
	au["image_url"] = imageURL

	if user != nil {
		au["user"] = map[string]any{
			"id":     user["id"],
			"object": string(constants.ObjectTypeEntity),
			"type":   string(constants.ObjectTypeUser),
			"name":   name,
			"handle": email,
		}
	} else {
		au["user"] = nil
	}
}
