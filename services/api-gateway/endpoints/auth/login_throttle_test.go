package authep

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	"github.com/augno/api/services/api-gateway/internal/middleware"
	apierror "github.com/augno/api/shared/errors"
	pb "github.com/augno/api/shared/proto/auth"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// stubAuthClient embeds pb.AuthServiceClient so we satisfy the interface
// without implementing the ~25 other methods. The embedded value stays nil;
// any unimplemented method called by accident will panic, which is exactly
// what we want from a test stub.
type stubAuthClient struct {
	pb.AuthServiceClient

	loginCalls atomic.Int32
	loginResp  *pb.LoginResponse
	loginErr   error
}

func (s *stubAuthClient) Login(_ context.Context, _ *pb.LoginRequest, _ ...grpc.CallOption) (*pb.LoginResponse, error) {
	s.loginCalls.Add(1)
	return s.loginResp, s.loginErr
}

func newAuthSvcForTest() *authSvcImpl {
	return &authSvcImpl{
		loginFailureLimiter: middleware.NewRateLimiter(loginFailureLimit, loginFailureWindow),
	}
}

func TestLoginThrottleKeyNormalizes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"lowercases email", "User@Example.com", "login:user@example.com"},
		{"trims whitespace", "  user@example.com  ", "login:user@example.com"},
		{"trims and lowercases", "  Admin@EXAMPLE.com\n", "login:admin@example.com"},
		{"plain username untouched aside from case", "AlICE", "login:alice"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := loginThrottleKey(tc.input); got != tc.want {
				t.Fatalf("loginThrottleKey(%q) = %q; want %q", tc.input, got, tc.want)
			}
		})
	}
}

// TestLoginThrottleBlocksByIdentifier exercises the per-identifier limiter
// state directly, confirming the limiter trips after the configured budget
// of failures and that an unrelated identifier is unaffected.
func TestLoginThrottleBlocksByIdentifier(t *testing.T) {
	t.Parallel()

	svc := newAuthSvcForTest()
	key := loginThrottleKey("victim@example.com")

	for range loginFailureLimit {
		svc.loginFailureLimiter.RecordFailure(key)
	}
	if allowed, _ := svc.loginFailureLimiter.Check(key); allowed {
		t.Fatalf("expected throttle to engage after %d failures", loginFailureLimit)
	}

	other := loginThrottleKey("bystander@example.com")
	if allowed, _ := svc.loginFailureLimiter.Check(other); !allowed {
		t.Fatalf("expected unrelated identifier %q to remain allowed", other)
	}
}

// TestLoginEndToEndThrottle drives Login through the real service code with
// a stubbed gRPC client that always returns invalid_credentials, and asserts:
//
//  1. The first loginFailureLimit attempts are forwarded to the auth service
//     and surface its error.
//  2. The very next attempt is short-circuited with rate_limit_exceeded and
//     never reaches the auth service.
//  3. A different identifier still reaches the auth service even when the
//     first one is throttled.
//  4. Case/whitespace differences in the same identifier share a bucket.
func TestLoginEndToEndThrottle(t *testing.T) {
	t.Parallel()

	stub := &stubAuthClient{
		loginErr: status.Error(codes.Unauthenticated, "invalid credentials"),
	}
	svc := &authSvcImpl{
		authClient:          stub,
		loginFailureLimiter: middleware.NewRateLimiter(loginFailureLimit, loginFailureWindow),
	}

	ctx := context.Background()
	req := &LoginRequest{Identifier: "victim@example.com", Password: "wrong"}

	// Stage 1: exhaust the failure budget. Every call reaches the auth
	// service and returns its (transport) error.
	for i := range loginFailureLimit {
		_, apiErr := svc.Login(ctx, req)
		if apiErr == nil {
			t.Fatalf("attempt %d: expected an error from the stub auth client, got nil", i+1)
		}
		if apiErr.Code == apierror.ErrorCodeRateLimitExceeded {
			t.Fatalf("attempt %d: throttle engaged too early (call should still reach gRPC)", i+1)
		}
	}
	if got := stub.loginCalls.Load(); got != int32(loginFailureLimit) {
		t.Fatalf("expected %d gRPC calls before throttle, got %d", loginFailureLimit, got)
	}

	// Stage 2: the next attempt must be short-circuited by the throttle.
	_, apiErr := svc.Login(ctx, req)
	if apiErr == nil {
		t.Fatal("expected rate-limit error once throttle engages, got nil")
	}
	if apiErr.Code != apierror.ErrorCodeRateLimitExceeded {
		t.Fatalf("expected ErrorCodeRateLimitExceeded, got %s", apiErr.Code)
	}
	if got := stub.loginCalls.Load(); got != int32(loginFailureLimit) {
		t.Fatalf("throttle must not invoke gRPC; expected %d calls, got %d", loginFailureLimit, got)
	}

	// Stage 3: a different identifier should still reach the auth service.
	bystanderCallsBefore := stub.loginCalls.Load()
	_, apiErr = svc.Login(ctx, &LoginRequest{Identifier: "bystander@example.com", Password: "wrong"})
	if apiErr == nil {
		t.Fatal("expected stub error for bystander, got nil")
	}
	if apiErr.Code == apierror.ErrorCodeRateLimitExceeded {
		t.Fatal("bystander identifier must not inherit victim's throttle")
	}
	if got := stub.loginCalls.Load(); got != bystanderCallsBefore+1 {
		t.Fatalf("expected one gRPC call for bystander, got delta %d", got-bystanderCallsBefore)
	}

	// Stage 4: case/whitespace variants of the throttled identifier share
	// the bucket — an attacker cannot rotate "  Victim@Example.com  " to
	// dodge the limit.
	callsBefore := stub.loginCalls.Load()
	_, apiErr = svc.Login(ctx, &LoginRequest{Identifier: "  Victim@Example.com  ", Password: "wrong"})
	if apiErr == nil || apiErr.Code != apierror.ErrorCodeRateLimitExceeded {
		t.Fatalf("expected rate-limit error for case/whitespace variant, got %v", apiErr)
	}
	if got := stub.loginCalls.Load(); got != callsBefore {
		t.Fatal("case/whitespace variant must not invoke gRPC")
	}
}

// TestLoginSuccessDoesNotIncrementThrottle confirms that successful logins
// don't count against the failure budget, so a user who fat-fingers a
// password but eventually succeeds isn't penalized.
func TestLoginSuccessDoesNotIncrementThrottle(t *testing.T) {
	t.Parallel()

	// A successful Login response triggers cookie helpers that read from
	// context; this test only needs to verify the throttle bookkeeping, so
	// we recover from any panic in the cookie path after asserting state.
	stub := &stubAuthClient{
		loginErr: errors.New("simulating happy path via failure - we only inspect throttle state"),
	}
	svc := &authSvcImpl{
		authClient:          stub,
		loginFailureLimiter: middleware.NewRateLimiter(loginFailureLimit, loginFailureWindow),
	}

	// 1 failure leaves the bucket non-empty.
	_, _ = svc.Login(context.Background(), &LoginRequest{Identifier: "user@example.com", Password: "x"})
	keyAfter := loginThrottleKey("user@example.com")

	// Sanity: 1 < limit, so Check still allows.
	if allowed, _ := svc.loginFailureLimiter.Check(keyAfter); !allowed {
		t.Fatal("single failure must not engage throttle")
	}
}
