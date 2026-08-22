package versiontransforms

import (
	"testing"

	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/version"
)

func accountUserPreview2Payload() map[string]any {
	return map[string]any{
		"id":     "acus_123",
		"object": "account_user",
		"user": map[string]any{
			"id":                "us_123",
			"object":            "user",
			"email":             "jane@example.com",
			"name":              "Jane Doe",
			"username":          "jane",
			"image_url":         "/v1/core/users/us_123/photo",
			"email_verified_at": nil,
			"created_at":        "2026-01-01T00:00:00Z",
			"updated_at":        "2026-01-02T00:00:00Z",
		},
		"status":     "active",
		"role":       nil,
		"department": nil,
		"created_at": "2026-01-01T00:00:00Z",
		"updated_at": "2026-01-02T00:00:00Z",
	}
}

func TestTransform_SingleAccountUser_Downgrade(t *testing.T) {
	t.Parallel()
	tr := &accountUserForgePreview2To1{}

	result := tr.Transform(constants.ObjectTypeAccountUser, accountUserPreview2Payload())

	if result["name"] != "Jane Doe" {
		t.Errorf("Expected hoisted name 'Jane Doe', got %v", result["name"])
	}
	if result["email"] != "jane@example.com" {
		t.Errorf("Expected hoisted email, got %v", result["email"])
	}
	if result["username"] != "jane" {
		t.Errorf("Expected hoisted username, got %v", result["username"])
	}
	if result["image_url"] != "/v1/core/users/us_123/photo" {
		t.Errorf("Expected hoisted image_url, got %v", result["image_url"])
	}

	user, ok := result["user"].(map[string]any)
	if !ok {
		t.Fatalf("Expected user to be an entity object, got %v", result["user"])
	}
	if user["id"] != "us_123" {
		t.Errorf("Expected entity id us_123, got %v", user["id"])
	}
	if user["object"] != "entity" {
		t.Errorf("Expected entity object type, got %v", user["object"])
	}
	if user["type"] != "user" {
		t.Errorf("Expected entity type user, got %v", user["type"])
	}
	if user["name"] != "Jane Doe" {
		t.Errorf("Expected entity name, got %v", user["name"])
	}
	if user["handle"] != "jane@example.com" {
		t.Errorf("Expected entity handle to be the email, got %v", user["handle"])
	}
}

func TestTransform_ListEnvelope_Downgrade(t *testing.T) {
	t.Parallel()
	tr := &accountUserForgePreview2To1{}

	payload := map[string]any{
		"object": "list",
		"data":   []any{accountUserPreview2Payload(), accountUserPreview2Payload()},
	}

	result := tr.Transform(constants.ObjectTypeAccountUser, payload)

	data := result["data"].([]any)
	for i, item := range data {
		au := item.(map[string]any)
		if au["name"] != "Jane Doe" {
			t.Errorf("item %d: expected hoisted name, got %v", i, au["name"])
		}
		user := au["user"].(map[string]any)
		if user["object"] != "entity" {
			t.Errorf("item %d: expected entity user reference, got %v", i, user["object"])
		}
	}
}

func TestTransform_AccountUserWithoutExpandedUser(t *testing.T) {
	t.Parallel()
	tr := &accountUserForgePreview2To1{}

	payload := accountUserPreview2Payload()
	payload["user"] = nil

	result := tr.Transform(constants.ObjectTypeAccountUser, payload)

	for _, key := range []string{"name", "email", "username", "image_url", "user"} {
		v, present := result[key]
		if !present {
			t.Errorf("Expected key %q to be present", key)
		}
		if v != nil {
			t.Errorf("Expected %q to be null without an expanded user, got %v", key, v)
		}
	}
}

func TestTransform_NestedAccountUser_Downgrade(t *testing.T) {
	t.Parallel()
	tr := &accountUserForgePreview2To1{}

	payload := map[string]any{
		"id":               "txn_123",
		"object":           "transaction",
		"responsible_user": accountUserPreview2Payload(),
	}

	result := tr.Transform(constants.ObjectTypeTransaction, payload)

	ru := result["responsible_user"].(map[string]any)
	if ru["name"] != "Jane Doe" {
		t.Errorf("Expected nested account user name hoisted, got %v", ru["name"])
	}
	user := ru["user"].(map[string]any)
	if user["object"] != "entity" {
		t.Errorf("Expected nested user demoted to entity, got %v", user["object"])
	}
}

func TestTransform_NestedAccountUserInContactMatch_Downgrade(t *testing.T) {
	t.Parallel()
	tr := &accountUserForgePreview2To1{}

	payload := map[string]any{
		"object": "list",
		"data": []any{
			map[string]any{
				"id":           "acus_123",
				"object":       "contact_match",
				"email":        "jane@example.com",
				"relationship": "customer",
				"account_user": accountUserPreview2Payload(),
			},
		},
	}

	result := tr.Transform(constants.ObjectTypeContactMatch, payload)

	cm := result["data"].([]any)[0].(map[string]any)
	au := cm["account_user"].(map[string]any)
	if au["name"] != "Jane Doe" {
		t.Errorf("Expected nested contact_match account user name hoisted, got %v", au["name"])
	}
	user := au["user"].(map[string]any)
	if user["object"] != "entity" {
		t.Errorf("Expected nested user demoted to entity, got %v", user["object"])
	}
}

func TestTransform_NestedAccountUserInWeekRelease_Downgrade(t *testing.T) {
	t.Parallel()
	tr := &accountUserForgePreview2To1{}

	payload := map[string]any{
		"object": "production_schedule_week_release",
		"production_run": map[string]any{
			"id":               "pnrn_123",
			"object":           "production_run",
			"responsible_user": accountUserPreview2Payload(),
		},
	}

	result := tr.Transform(constants.ObjectTypeProductionScheduleWeekRelease, payload)

	run := result["production_run"].(map[string]any)
	ru := run["responsible_user"].(map[string]any)
	if ru["name"] != "Jane Doe" {
		t.Errorf("Expected nested account user name hoisted, got %v", ru["name"])
	}
	user := ru["user"].(map[string]any)
	if user["object"] != "entity" {
		t.Errorf("Expected nested user demoted to entity, got %v", user["object"])
	}
}

func TestTransformRequest_IsIdentity(t *testing.T) {
	t.Parallel()
	tr := &accountUserForgePreview2To1{}

	payload := map[string]any{"name": "Jane Doe", "role_id": "role_123"}
	result := tr.TransformRequest(constants.ObjectTypeAccountUser, payload)

	if result["name"] != "Jane Doe" || result["role_id"] != "role_123" {
		t.Errorf("Expected request payload unchanged, got %v", result)
	}
}

func TestForcedIncludes(t *testing.T) {
	t.Parallel()
	tr := &accountUserForgePreview2To1{}

	keys := tr.ForcedIncludes(constants.ObjectTypeAccountUser)
	if len(keys) != 1 || keys[0] != "user" {
		t.Errorf("Expected forced include [user] for account_user, got %v", keys)
	}

	keys = tr.ForcedIncludes(constants.ObjectTypeTransaction)
	if len(keys) != 2 || keys[0] != "responsible_user" || keys[1] != "responsible_user.user" {
		t.Errorf("Expected forced includes [responsible_user responsible_user.user] for transaction, got %v", keys)
	}

	keys = tr.ForcedIncludes(constants.ObjectTypeCustomer)
	if len(keys) != 1 || keys[0] != "defaults.sales_rep.user" {
		t.Errorf("Expected forced include [defaults.sales_rep.user] for customer, got %v", keys)
	}

	keys = tr.ForcedIncludes(constants.ObjectTypeContactMatch)
	if len(keys) != 1 || keys[0] != "account_user.user" {
		t.Errorf("Expected forced include [account_user.user] for contact_match, got %v", keys)
	}

	if keys := tr.ForcedIncludes(constants.ObjectTypePurchaseOrder); keys != nil {
		t.Errorf("Expected no forced includes for purchase_order, got %v", keys)
	}
}

func TestDefaultRegistry_EndToEnd(t *testing.T) {
	t.Parallel(
	// The package init() registers the transformer in the default registry;
	// exercise the same path the gateway uses.
	)

	result := version.Transform(
		version.V1_0_Forge_Preview2,
		version.V1_0_Forge_Preview1,
		constants.ObjectTypeAccountUser,
		accountUserPreview2Payload(),
	)

	if result["email"] != "jane@example.com" {
		t.Errorf("Expected default registry to apply the downgrade, got %v", result["email"])
	}

	forced := version.ForcedIncludes(version.V1_0_Forge_Preview2, version.V1_0_Forge_Preview1, constants.ObjectTypeAccountUser)
	if len(forced) != 1 || forced[0] != "user" {
		t.Errorf("Expected forced includes [user] from default registry, got %v", forced)
	}
}
