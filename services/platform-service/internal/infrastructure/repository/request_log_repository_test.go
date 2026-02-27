package repository

import (
	"database/sql"
	"testing"
	"time"

	"github.com/augno/api/services/platform-service/internal/infrastructure/sqlc"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/db"
)

func nullStr(s string) sql.NullString {
	return sql.NullString{String: s, Valid: true}
}

func nullStrEmpty() sql.NullString {
	return sql.NullString{}
}

func baseRow() sqlc.FindRequestLogByIDRow {
	return sqlc.FindRequestLogByIDRow{
		ID:              "rlog_test123",
		Method:          "GET",
		Path:            "/v1/test",
		NormalizedRoute: "/v1/test",
		StatusCode:      200,
		LatencyUs:       1234,
		OccurredAt:      time.Now().UTC(),
		CreatedAt:       time.Now().UTC(),
		TargetAccountID: nullStr("acct_target"),
		AccountName:     nullStr("Test Account"),
	}
}

func TestMapRowToRequestLogRead_UserActor(t *testing.T) {
	row := baseRow()
	row.ActorID = nullStr("usr_abc123")
	row.ActorType = nullStr("internal")
	row.IdentityType = nullStr("user")
	row.UserEmail = nullStr("user@example.com")
	row.UserName = nullStr("John Doe")
	row.UserRoleID = nullStr("role_admin")
	row.UserRoleName = nullStr("Admin")
	row.UserRoleTypeCode = nullStr("admin")

	rl := mapRowToRequestLogRead(&row)

	if rl.Actor == nil {
		t.Fatal("expected Actor to be resolved for identity_type=user")
	}
	if rl.Actor.ObjectType != constants.ObjectTypeUser {
		t.Errorf("expected ObjectType %q, got %q", constants.ObjectTypeUser, rl.Actor.ObjectType)
	}
	if rl.Actor.ID != "usr_abc123" {
		t.Errorf("expected actor ID 'usr_abc123', got %q", rl.Actor.ID)
	}
	if rl.Actor.Email == nil || *rl.Actor.Email != "user@example.com" {
		t.Errorf("expected email 'user@example.com', got %v", rl.Actor.Email)
	}
	if rl.Actor.Name == nil || *rl.Actor.Name != "John Doe" {
		t.Errorf("expected name 'John Doe', got %v", rl.Actor.Name)
	}
	if rl.Actor.RoleID == nil || *rl.Actor.RoleID != "role_admin" {
		t.Errorf("expected role ID 'role_admin', got %v", rl.Actor.RoleID)
	}
	if rl.Actor.RoleName == nil || *rl.Actor.RoleName != "Admin" {
		t.Errorf("expected role name 'Admin', got %v", rl.Actor.RoleName)
	}
	if rl.Actor.RoleTypeCode == nil || *rl.Actor.RoleTypeCode != "admin" {
		t.Errorf("expected role type code 'admin', got %v", rl.Actor.RoleTypeCode)
	}
}

func TestMapRowToRequestLogRead_APIKeyActor(t *testing.T) {
	row := baseRow()
	row.ActorID = nullStr("apke_key_id_123")
	row.ActorType = nullStr("customer")
	row.IdentityType = nullStr("api_key")
	row.ApiKeyTypeID = nullStr("apk_type_id_456")
	row.ApiKeyName = nullStr("My API Key")
	row.ApiKeyRedactedValue = nullStr("sk_test_...abc")
	row.ApiKeyRoleID = nullStr("role_member")
	row.ApiKeyRoleName = nullStr("Member")
	row.ApiKeyRoleTypeCode = nullStr("member")

	rl := mapRowToRequestLogRead(&row)

	if rl.Actor == nil {
		t.Fatal("expected Actor to be resolved for identity_type=api_key")
	}
	if rl.Actor.ObjectType != constants.ObjectTypeAPIKey {
		t.Errorf("expected ObjectType %q, got %q", constants.ObjectTypeAPIKey, rl.Actor.ObjectType)
	}
	// Should use ApiKeyTypeID as the ID when available
	if rl.Actor.ID != "apk_type_id_456" {
		t.Errorf("expected actor ID 'apk_type_id_456', got %q", rl.Actor.ID)
	}
	if rl.Actor.Name == nil || *rl.Actor.Name != "My API Key" {
		t.Errorf("expected name 'My API Key', got %v", rl.Actor.Name)
	}
	if rl.Actor.RedactedValue == nil || *rl.Actor.RedactedValue != "sk_test_...abc" {
		t.Errorf("expected redacted value 'sk_test_...abc', got %v", rl.Actor.RedactedValue)
	}
	if rl.Actor.RoleID == nil || *rl.Actor.RoleID != "role_member" {
		t.Errorf("expected role ID 'role_member', got %v", rl.Actor.RoleID)
	}
}

func TestMapRowToRequestLogRead_APIKeyActor_FallbackToActorID(t *testing.T) {
	row := baseRow()
	row.ActorID = nullStr("apke_key_id_123")
	row.ActorType = nullStr("customer")
	row.IdentityType = nullStr("api_key")
	// No ApiKeyTypeID - should fall back to ActorID
	row.ApiKeyTypeID = nullStrEmpty()
	row.ApiKeyName = nullStr("Fallback Key")

	rl := mapRowToRequestLogRead(&row)

	if rl.Actor == nil {
		t.Fatal("expected Actor to be resolved")
	}
	if rl.Actor.ID != "apke_key_id_123" {
		t.Errorf("expected fallback to actor ID 'apke_key_id_123', got %q", rl.Actor.ID)
	}
}

func TestMapRowToRequestLogRead_NoActor_WhenIdentityTypeNil(t *testing.T) {
	row := baseRow()
	row.ActorID = nullStr("usr_abc123")
	row.ActorType = nullStr("internal")
	row.IdentityType = nullStrEmpty() // nil identity type

	rl := mapRowToRequestLogRead(&row)

	if rl.Actor != nil {
		t.Error("expected Actor to be nil when identity_type is nil")
	}
}

func TestMapRowToRequestLogRead_NoActor_WhenActorIDNil(t *testing.T) {
	row := baseRow()
	row.ActorID = nullStrEmpty() // nil actor ID
	row.ActorType = nullStr("internal")
	row.IdentityType = nullStr("user")

	rl := mapRowToRequestLogRead(&row)

	if rl.Actor != nil {
		t.Error("expected Actor to be nil when actor_id is nil")
	}
}

func TestMapRowToRequestLogRead_NoActor_WhenUnauthenticated(t *testing.T) {
	row := baseRow()
	row.ActorID = nullStr("usr_abc123")
	row.ActorType = nullStr("internal")
	row.IdentityType = nullStr("unauthenticated")

	rl := mapRowToRequestLogRead(&row)

	// "unauthenticated" doesn't match "user" or "api_key", so no actor
	if rl.Actor != nil {
		t.Error("expected Actor to be nil for unauthenticated identity_type")
	}
}

func TestMapRowToRequestLogRead_InternalActorType_WithUserIdentity(t *testing.T) {
	// This was the original bug: actor_type "internal" was used for the switch,
	// never matching "user" or "api_key". Now we switch on identity_type.
	row := baseRow()
	row.ActorID = nullStr("usr_abc123")
	row.ActorType = nullStr("internal") // was incorrectly used for switching
	row.IdentityType = nullStr("user")  // this is what should drive the switch
	row.UserEmail = nullStr("admin@example.com")
	row.UserName = nullStr("Admin User")

	rl := mapRowToRequestLogRead(&row)

	if rl.Actor == nil {
		t.Fatal("expected Actor to be resolved when identity_type=user, even with actor_type=internal")
	}
	if rl.Actor.ObjectType != constants.ObjectTypeUser {
		t.Errorf("expected ObjectType %q, got %q", constants.ObjectTypeUser, rl.Actor.ObjectType)
	}
	if rl.Actor.Email == nil || *rl.Actor.Email != "admin@example.com" {
		t.Errorf("expected email 'admin@example.com', got %v", rl.Actor.Email)
	}
}

func TestMapRowToRequestLogRead_CustomerActorType_WithAPIKeyIdentity(t *testing.T) {
	// Similar to above: actor_type "customer" should not block API key resolution
	row := baseRow()
	row.ActorID = nullStr("apke_key1")
	row.ActorType = nullStr("customer")
	row.IdentityType = nullStr("api_key")
	row.ApiKeyTypeID = nullStr("apk_type1")
	row.ApiKeyName = nullStr("Customer Key")

	rl := mapRowToRequestLogRead(&row)

	if rl.Actor == nil {
		t.Fatal("expected Actor to be resolved when identity_type=api_key, even with actor_type=customer")
	}
	if rl.Actor.ObjectType != constants.ObjectTypeAPIKey {
		t.Errorf("expected ObjectType %q, got %q", constants.ObjectTypeAPIKey, rl.Actor.ObjectType)
	}
}

func TestMapRowToRequestLogRead_AccountInfo(t *testing.T) {
	row := baseRow()
	row.TargetAccountID = nullStr("acct_12345")
	row.AccountName = nullStr("My Company")

	rl := mapRowToRequestLogRead(&row)

	if rl.AccountID == nil || *rl.AccountID != "acct_12345" {
		t.Errorf("expected AccountID 'acct_12345', got %v", rl.AccountID)
	}
	if rl.AccountName == nil || *rl.AccountName != "My Company" {
		t.Errorf("expected AccountName 'My Company', got %v", rl.AccountName)
	}
}

func TestMapRowToRequestLogRead_QueryAndBodyJSON(t *testing.T) {
	row := baseRow()
	row.QueryJson = db.NullableRawMessage(`{"page":"2"}`)
	row.RequestBodyJson = db.NullableRawMessage(`{"name":"test"}`)

	rl := mapRowToRequestLogRead(&row)

	if rl.QueryJSON == nil || *rl.QueryJSON != `{"page":"2"}` {
		t.Errorf("expected QueryJSON, got %v", rl.QueryJSON)
	}
	if rl.BodyJSON == nil || *rl.BodyJSON != `{"name":"test"}` {
		t.Errorf("expected BodyJSON, got %v", rl.BodyJSON)
	}
}

func TestMapRowToRequestLogRead_NilQueryAndBodyJSON(t *testing.T) {
	row := baseRow()
	row.QueryJson = nil
	row.RequestBodyJson = nil

	rl := mapRowToRequestLogRead(&row)

	if rl.QueryJSON != nil {
		t.Errorf("expected QueryJSON nil, got %v", rl.QueryJSON)
	}
	if rl.BodyJSON != nil {
		t.Errorf("expected BodyJSON nil, got %v", rl.BodyJSON)
	}
}

func TestMapRowToRequestLogRead_IdempotencyKey(t *testing.T) {
	row := baseRow()
	row.IdempotencyKey = nullStr("idem-123-abc")

	rl := mapRowToRequestLogRead(&row)

	if rl.IdempotencyKey == nil || *rl.IdempotencyKey != "idem-123-abc" {
		t.Errorf("expected IdempotencyKey 'idem-123-abc', got %v", rl.IdempotencyKey)
	}
}

func TestMapRowToRequestLogRead_BasicFields(t *testing.T) {
	row := baseRow()
	row.ApiVersion = nullStr("2.0.0")
	row.IdentityType = nullStr("user")
	row.ClientIpString = nullStr("10.0.0.1")
	row.UserAgent = nullStr("Mozilla/5.0")
	row.Referrer = nullStr("https://example.com")
	row.ErrorCode = nullStr("not_found")
	row.ErrorMessage = nullStr("Resource not found")

	rl := mapRowToRequestLogRead(&row)

	if rl.ID != "rlog_test123" {
		t.Errorf("expected ID 'rlog_test123', got %q", rl.ID)
	}
	if rl.Method != "GET" {
		t.Errorf("expected Method 'GET', got %q", rl.Method)
	}
	if rl.StatusCode != 200 {
		t.Errorf("expected StatusCode 200, got %d", rl.StatusCode)
	}
	if rl.APIVersion == nil || *rl.APIVersion != "2.0.0" {
		t.Errorf("expected APIVersion '2.0.0', got %v", rl.APIVersion)
	}
	if rl.ClientIP == nil || *rl.ClientIP != "10.0.0.1" {
		t.Errorf("expected ClientIP '10.0.0.1', got %v", rl.ClientIP)
	}
	if rl.UserAgent == nil || *rl.UserAgent != "Mozilla/5.0" {
		t.Errorf("expected UserAgent 'Mozilla/5.0', got %v", rl.UserAgent)
	}
	if rl.ErrorCode == nil || *rl.ErrorCode != "not_found" {
		t.Errorf("expected ErrorCode 'not_found', got %v", rl.ErrorCode)
	}
}
