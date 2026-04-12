package requestlogep

import (
	"testing"

	"github.com/augno/api/services/api-gateway/pkg/resource/resourcetest"
	pb "github.com/augno/api/shared/proto/platform"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func newFullRequestLogInfo() *pb.RequestLogInfo {
	accountID := "ac_01abc"
	accountName := "Acme Inc."
	queryJSON := `{"limit":10}`
	apiVersion := "2026-01-01"
	identityType := "user"
	clientIP := "198.51.100.7"
	userAgent := "Mozilla/5.0"
	referrer := "https://example.com"
	idempotencyKey := "idem_123"
	bodyJSON := `{"foo":"bar"}`
	responseJSON := `{"ok":true}`

	roleID := "rl_01abc"
	roleName := "Admin"
	roleTypeCode := "admin"
	actorName := "John Doe"
	actorEmail := "jdoe@example.com"

	return &pb.RequestLogInfo{
		Id:              "rl_01abc",
		Method:          "GET",
		Path:            "/v1/core/sandboxes",
		Host:            "https://api.augno.com",
		NormalizedRoute: "/v1/core/sandboxes",
		QueryJson:       &queryJSON,
		StatusCode:      200,
		LatencyUs:       12345,
		ApiVersion:      &apiVersion,
		IdentityType:    &identityType,
		ClientIp:        &clientIP,
		UserAgent:       &userAgent,
		Referrer:        &referrer,
		OccurredAt:      timestamppb.Now(),
		CreatedAt:       timestamppb.Now(),
		AccountId:       &accountID,
		AccountName:     &accountName,
		IdempotencyKey:  &idempotencyKey,
		BodyJson:        &bodyJSON,
		ResponseJson:    &responseJSON,
		Actor: &pb.RequestLogActor{
			Id:           "us_01abc",
			ActorType:    "user",
			Name:         &actorName,
			Email:        &actorEmail,
			RoleId:       &roleID,
			RoleName:     &roleName,
			RoleTypeCode: &roleTypeCode,
		},
	}
}

var testPerms = map[string]bool{"customer:read": true}

func TestRequestLogPresenter(t *testing.T) {
	t.Parallel()
	result := RequestLogPresenter(newFullRequestLogInfo(), testPerms)
	resourcetest.ValidateResourceStruct(t, "RequestLog", result)
	require.NotNil(t, result.Actor)
	require.NotNil(t, result.Actor.Role)
	require.Equal(t, &[]string{"customer:read"}, result.Actor.Role.Permissions)
	require.Equal(t, "jdoe@example.com", *result.Actor.Handle)
}

func TestRequestLogPresenter_ActorRoleWithoutTypeCode(t *testing.T) {
	t.Parallel()
	roleID := "rl_01abc"
	roleName := "Admin"
	actorName := "John Doe"

	rl := &pb.RequestLogInfo{
		Id:              "rl_01abc",
		Method:          "GET",
		Path:            "/v1/test",
		Host:            "https://api.augno.com",
		NormalizedRoute: "/v1/test",
		StatusCode:      200,
		LatencyUs:       100,
		OccurredAt:      timestamppb.Now(),
		CreatedAt:       timestamppb.Now(),
		Actor: &pb.RequestLogActor{
			Id:           "us_01abc",
			ActorType:    "user",
			Name:         &actorName,
			RoleId:       &roleID,
			RoleName:     &roleName,
			RoleTypeCode: nil,
		},
	}

	result := RequestLogPresenter(rl, nil)
	resourcetest.ValidateResourceStruct(t, "RequestLog(ActorRoleWithoutTypeCode)", result)
	require.NotNil(t, result.Actor)
	require.Nil(t, result.Actor.Role)
	require.Nil(t, result.Actor.Handle)
}

func TestRequestLogPresenter_APIKeyActorUsesRedactedValueAsHandle(t *testing.T) {
	t.Parallel()
	actorName := "Service Key"
	redactedValue := "aug_sk_live_****1234"

	rl := &pb.RequestLogInfo{
		Id:              "rl_01abc",
		Method:          "GET",
		Path:            "/v1/test",
		Host:            "https://api.augno.com",
		NormalizedRoute: "/v1/test",
		StatusCode:      200,
		LatencyUs:       100,
		OccurredAt:      timestamppb.Now(),
		CreatedAt:       timestamppb.Now(),
		Actor: &pb.RequestLogActor{
			Id:            "apke_01abc",
			ActorType:     "api_key",
			Name:          &actorName,
			RedactedValue: &redactedValue,
		},
	}

	result := RequestLogPresenter(rl, nil)

	require.NotNil(t, result.Actor)
	require.Equal(t, "apke_01abc", result.Actor.ID)
	require.Equal(t, redactedValue, *result.Actor.Handle)
}

func TestRequestLogListPresenter_NullsJSONPayloadFields(t *testing.T) {
	t.Parallel()
	resp := &pb.ListRequestLogsResponse{
		RequestLogs: []*pb.RequestLogInfo{newFullRequestLogInfo()},
	}

	result := RequestLogListPresenter(resp, func(roleID *string) map[string]bool {
		return nil
	})

	require.Len(t, result.Data, 1)
	require.Nil(t, result.Data[0].QueryJSON)
	require.Nil(t, result.Data[0].RequestBodyJSON)
	require.Nil(t, result.Data[0].ResponseBodyJSON)
}
