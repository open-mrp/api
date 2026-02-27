package event

import (
	"testing"
	"time"

	loggingpb "github.com/augno/api/shared/proto/platform"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func strPtr(s string) *string { return &s }

func TestMapEventToDomain_MapsAccountID(t *testing.T) {
	accountID := "acct_home123"
	targetAccountID := "acct_target456"
	actorID := "usr_abc"
	actorType := "internal"
	identityType := "user"

	event := &loggingpb.RequestLog{
		Id:              "rlog_test1",
		Method:          "GET",
		Host:            "api.example.com",
		Path:            "/v1/test",
		NormalizedRoute: "/v1/test",
		StatusCode:      200,
		LatencyUs:       1234,
		AccountId:       &accountID,
		TargetAccountId: &targetAccountID,
		ActorId:         &actorID,
		ActorType:       &actorType,
		IdentityType:    &identityType,
		OccurredAt:      timestamppb.Now(),
		CreatedAt:       timestamppb.Now(),
		PublicEndpoint:  true,
	}

	rl := mapEventToDomain(event)

	if rl.AccountID == nil || *rl.AccountID != accountID {
		t.Errorf("expected AccountID %q, got %v", accountID, rl.AccountID)
	}
	if rl.TargetAccountID == nil || *rl.TargetAccountID != targetAccountID {
		t.Errorf("expected TargetAccountID %q, got %v", targetAccountID, rl.TargetAccountID)
	}
	if rl.ActorID == nil || *rl.ActorID != actorID {
		t.Errorf("expected ActorID %q, got %v", actorID, rl.ActorID)
	}
	if rl.ActorType == nil || *rl.ActorType != actorType {
		t.Errorf("expected ActorType %q, got %v", actorType, rl.ActorType)
	}
	if rl.IdentityType == nil || *rl.IdentityType != identityType {
		t.Errorf("expected IdentityType %q, got %v", identityType, rl.IdentityType)
	}
}

func TestMapEventToDomain_NilAccountID(t *testing.T) {
	event := &loggingpb.RequestLog{
		Id:             "rlog_test2",
		Method:         "GET",
		Host:           "api.example.com",
		Path:           "/v1/test",
		StatusCode:     200,
		OccurredAt:     timestamppb.Now(),
		CreatedAt:      timestamppb.Now(),
		PublicEndpoint: true,
	}

	rl := mapEventToDomain(event)

	if rl.AccountID != nil {
		t.Errorf("expected AccountID nil, got %v", rl.AccountID)
	}
}

func TestMapEventToDomain_AllFields(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)

	event := &loggingpb.RequestLog{
		Id:                   "rlog_full",
		Method:               "POST",
		Host:                 "api.example.com",
		Path:                 "/v1/things",
		NormalizedRoute:      "/v1/things",
		QueryJson:            strPtr(`{"page":"1"}`),
		StatusCode:           422,
		LatencyUs:            5000,
		AccountId:            strPtr("acct_home"),
		TargetAccountId:      strPtr("acct_target"),
		ActorId:              strPtr("apke_key1"),
		ActorType:            strPtr("customer"),
		IdentityType:         strPtr("api_key"),
		ClientIp:             []byte{192, 168, 1, 1},
		ClientIpString:       strPtr("192.168.1.1"),
		UserAgent:            strPtr("test-agent"),
		Referrer:             strPtr("https://example.com"),
		ErrorCode:            strPtr("validation_error"),
		ErrorMessage:         strPtr("invalid input"),
		OccurredAt:           timestamppb.New(now),
		CreatedAt:            timestamppb.New(now),
		IdempotencyKeyId:     strPtr("idem_key1"),
		InternalErrorMessage: strPtr("internal details"),
		StackTrace:           strPtr("goroutine 1 [running]:"),
		ApiVersion:           strPtr("1.0.0"),
		TraceId:              strPtr("trace123"),
		PublicEndpoint:       false,
		BodyJson:             strPtr(`{"name":"test"}`),
	}

	rl := mapEventToDomain(event)

	if rl.ID != "rlog_full" {
		t.Errorf("expected ID 'rlog_full', got %q", rl.ID)
	}
	if rl.Method != "POST" {
		t.Errorf("expected Method 'POST', got %q", rl.Method)
	}
	if rl.StatusCode != 422 {
		t.Errorf("expected StatusCode 422, got %d", rl.StatusCode)
	}
	if rl.LatencyUs != 5000 {
		t.Errorf("expected LatencyUs 5000, got %d", rl.LatencyUs)
	}
	if rl.PublicEndpoint {
		t.Error("expected PublicEndpoint false, got true")
	}

	checks := []struct {
		name     string
		got      *string
		expected string
	}{
		{"AccountID", rl.AccountID, "acct_home"},
		{"TargetAccountID", rl.TargetAccountID, "acct_target"},
		{"ActorID", rl.ActorID, "apke_key1"},
		{"ActorType", rl.ActorType, "customer"},
		{"IdentityType", rl.IdentityType, "api_key"},
		{"QueryJSON", rl.QueryJSON, `{"page":"1"}`},
		{"BodyJSON", rl.BodyJSON, `{"name":"test"}`},
		{"UserAgent", rl.UserAgent, "test-agent"},
		{"Referrer", rl.Referrer, "https://example.com"},
		{"ErrorCode", rl.ErrorCode, "validation_error"},
		{"ErrorMessage", rl.ErrorMessage, "invalid input"},
		{"APIVersion", rl.APIVersion, "1.0.0"},
		{"TraceID", rl.TraceID, "trace123"},
		{"IdempotencyKeyTypeID", rl.IdempotencyKeyTypeID, "idem_key1"},
		{"InternalErrorMessage", rl.InternalErrorMessage, "internal details"},
		{"StackTrace", rl.StackTrace, "goroutine 1 [running]:"},
		{"ClientIPString", rl.ClientIPString, "192.168.1.1"},
	}

	for _, check := range checks {
		if check.got == nil {
			t.Errorf("%s: expected %q, got nil", check.name, check.expected)
		} else if *check.got != check.expected {
			t.Errorf("%s: expected %q, got %q", check.name, check.expected, *check.got)
		}
	}

	if !rl.OccurredAt.Equal(now) {
		t.Errorf("expected OccurredAt %v, got %v", now, rl.OccurredAt)
	}
	if !rl.CreatedAt.Equal(now) {
		t.Errorf("expected CreatedAt %v, got %v", now, rl.CreatedAt)
	}
}

func TestMapEventToDomain_DefaultTimestamps(t *testing.T) {
	before := time.Now().UTC()

	event := &loggingpb.RequestLog{
		Id:         "rlog_notime",
		Method:     "GET",
		Host:       "api.example.com",
		Path:       "/v1/test",
		StatusCode: 200,
	}

	rl := mapEventToDomain(event)

	after := time.Now().UTC()

	if rl.OccurredAt.Before(before) || rl.OccurredAt.After(after) {
		t.Errorf("expected OccurredAt to default to ~now, got %v", rl.OccurredAt)
	}
	if rl.CreatedAt.Before(before) || rl.CreatedAt.After(after) {
		t.Errorf("expected CreatedAt to default to ~now, got %v", rl.CreatedAt)
	}
}
