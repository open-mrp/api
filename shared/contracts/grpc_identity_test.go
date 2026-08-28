package contracts

import (
	"context"
	"encoding/json"
	"net"
	"testing"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/appctx"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/test/bufconn"
)

func strPtr(s string) *string { return &s }

func TestGetIdentityFromMetadata_Errors(t *testing.T) {
	t.Parallel()
	duplicated := metadata.New(nil)
	duplicated.Append(IdentityHeader, `{"Type":"user"}`, `{"Type":"api_key"}`)

	tests := []struct {
		name string
		md   metadata.MD
	}{
		{
			name: "header absent",
			md:   metadata.New(nil),
		},
		{
			name: "nil metadata",
			md:   nil,
		},
		{
			name: "header duplicated",
			md:   duplicated,
		},
		{
			name: "malformed json",
			md:   metadata.Pairs(IdentityHeader, "{not-json"),
		},
		{
			name: "json of the wrong shape",
			md:   metadata.Pairs(IdentityHeader, `["user"]`),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			identity, apiErr := GetIdentityFromMetadata(tt.md)
			if identity != nil {
				t.Errorf("expected nil identity, got %+v", identity)
			}
			if apiErr == nil {
				t.Fatal("expected an APIError, got nil")
			}
			if apiErr.Code != apierror.ErrorCodeInternalError {
				t.Errorf("expected code %q, got %q", apierror.ErrorCodeInternalError, apiErr.Code)
			}
		})
	}
}

func TestSetIdentityInMetadata_RoundTrip(t *testing.T) {
	t.Parallel()
	relation := types.IdentityRelationTypeCustomer
	identity := &types.Identity{
		Type:               types.IdentityActorTypeUser,
		Target:             &types.IdentityTarget{AccountID: "acc_target", RelationType: &relation},
		AccountMode:        constants.AccountModeSandbox,
		SubscriptionStatus: strPtr("active"),
		Actor: &types.IdentityActor{
			RelationType: types.IdentityRelationTypeInternal,
			ID:           "usr_123",
			Name:         strPtr("Dana Smith"),
			AccountID:    strPtr("acc_actor"),
			RoleID:       strPtr("rol_1"),
			RoleType:     strPtr("admin"),
			RoleName:     strPtr("Administrator"),
			Permissions:  map[string]bool{"orders:create": true},
		},
	}

	md := metadata.New(nil)
	SetIdentityInMetadata(md, identity)

	if got := len(md.Get(IdentityHeader)); got != 1 {
		t.Fatalf("expected 1 identity header value, got %d", got)
	}

	decoded, apiErr := GetIdentityFromMetadata(md)
	if apiErr != nil {
		t.Fatalf("expected no error, got %v", apiErr)
	}
	if decoded.Type != identity.Type {
		t.Errorf("expected type %q, got %q", identity.Type, decoded.Type)
	}
	if decoded.AccountMode != identity.AccountMode {
		t.Errorf("expected account mode %q, got %q", identity.AccountMode, decoded.AccountMode)
	}
	if decoded.Target == nil || decoded.Target.AccountID != "acc_target" {
		t.Errorf("expected target account acc_target, got %+v", decoded.Target)
	}
	if decoded.Actor == nil {
		t.Fatal("expected actor, got nil")
	}
	if *decoded.Actor.Name != *identity.Actor.Name {
		t.Errorf("expected actor name %q, got %q", *identity.Actor.Name, *decoded.Actor.Name)
	}
	if *decoded.Actor.RoleName != *identity.Actor.RoleName {
		t.Errorf("expected role name %q, got %q", *identity.Actor.RoleName, *decoded.Actor.RoleName)
	}
	if !decoded.Actor.Permissions["orders:create"] {
		t.Errorf("expected permissions to survive, got %+v", decoded.Actor.Permissions)
	}
}

func TestSetIdentityInMetadata_NilIdentity(t *testing.T) {
	t.Parallel()
	md := metadata.New(nil)
	SetIdentityInMetadata(md, nil)

	if values := md.Get(IdentityHeader); len(values) != 1 || values[0] != "null" {
		t.Errorf("expected a single \"null\" header for a nil identity, got %v", values)
	}

	decoded, apiErr := GetIdentityFromMetadata(md)
	if apiErr != nil {
		t.Fatalf("expected no error decoding a null identity, got %v", apiErr)
	}
	if decoded == nil {
		t.Fatal("expected a non-nil zero identity")
	}
	if decoded.Type != "" || decoded.Actor != nil {
		t.Errorf("expected a zero identity, got %+v", decoded)
	}
}

func TestIdentityUnaryServerInterceptor_MalformedHeader(t *testing.T) {
	t.Parallel()
	duplicated := metadata.New(nil)
	duplicated.Append(IdentityHeader, `{"Type":"user"}`, `{"Type":"api_key"}`)

	tests := []struct {
		name          string
		md            metadata.MD
		wantIdentity  bool
		wantActorType types.IdentityActorType
	}{
		{
			name:          "well-formed header populates the context",
			md:            metadata.Pairs(IdentityHeader, `{"Type":"user","Actor":{"ID":"usr_1"}}`),
			wantIdentity:  true,
			wantActorType: types.IdentityActorTypeUser,
		},
		{
			name:         "malformed json is swallowed and the rpc proceeds",
			md:           metadata.Pairs(IdentityHeader, "{not-json"),
			wantIdentity: false,
		},
		{
			name:         "duplicated header is swallowed and the rpc proceeds",
			md:           duplicated,
			wantIdentity: false,
		},
		{
			name:         "absent header is swallowed and the rpc proceeds",
			md:           metadata.New(nil),
			wantIdentity: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			interceptor := IdentityUnaryServerInterceptor()
			ctx := metadata.NewIncomingContext(context.Background(), tt.md)

			called := false
			var handlerCtx context.Context
			handler := func(ctx context.Context, req any) (any, error) {
				called = true
				handlerCtx = ctx
				return "ok", nil
			}

			resp, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{FullMethod: "/svc/Method"}, handler)
			if !called {
				t.Fatal("expected the handler to be invoked")
			}
			if err != nil {
				t.Errorf("expected no error, got %v", err)
			}
			if resp != "ok" {
				t.Errorf("expected handler response to pass through, got %v", resp)
			}

			identity, ok := appctx.GetIdentityFromContext(handlerCtx)
			if ok != tt.wantIdentity {
				t.Fatalf("expected identity present=%v, got %v", tt.wantIdentity, ok)
			}
			if tt.wantIdentity && identity.Type != tt.wantActorType {
				t.Errorf("expected actor type %q, got %q", tt.wantActorType, identity.Type)
			}
		})
	}
}

func TestIdentityUnaryServerInterceptor_NoIncomingMetadata(t *testing.T) {
	t.Parallel()
	interceptor := IdentityUnaryServerInterceptor()

	var handlerCtx context.Context
	handler := func(ctx context.Context, req any) (any, error) {
		handlerCtx = ctx
		return nil, nil
	}

	if _, err := interceptor(context.Background(), nil, nil, handler); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if _, ok := appctx.GetIdentityFromContext(handlerCtx); ok {
		t.Error("expected no identity in context")
	}
}

// Accounts routinely carry accented, CJK and emoji names; the identity must
// survive the JSON encoding used to move it between services intact.
func TestSetIdentityInMetadata_NonASCIIRoundTrip(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		actor    string
		roleName string
	}{
		{
			name:     "accented latin",
			actor:    "José Álvarez",
			roleName: "Gestión de Inventario",
		},
		{
			name:     "cjk",
			actor:    "田中太郎",
			roleName: "管理者",
		},
		{
			name:     "emoji",
			actor:    "Ops 🚀",
			roleName: "Warehouse 📦",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			identity := &types.Identity{
				Type: types.IdentityActorTypeAPIKey,
				Actor: &types.IdentityActor{
					ID:       "apke_1",
					Name:     strPtr(tt.actor),
					RoleName: strPtr(tt.roleName),
				},
			}

			md := metadata.New(nil)
			SetIdentityInMetadata(md, identity)

			decoded, apiErr := GetIdentityFromMetadata(md)
			if apiErr != nil {
				t.Fatalf("expected no error, got %v", apiErr)
			}
			if *decoded.Actor.Name != tt.actor {
				t.Errorf("expected actor name %q, got %q", tt.actor, *decoded.Actor.Name)
			}
			if *decoded.Actor.RoleName != tt.roleName {
				t.Errorf("expected role name %q, got %q", tt.roleName, *decoded.Actor.RoleName)
			}
		})
	}
}

// serveIdentityEcho starts a gRPC server on an in-memory listener whose interceptor records the identity it decodes from incoming metadata, and returns a client connection to it.
func serveIdentityEcho(t *testing.T, got chan<- *types.Identity) *grpc.ClientConn {
	t.Helper()
	lis := bufconn.Listen(1 << 20)
	srv := grpc.NewServer(grpc.UnaryInterceptor(
		func(ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
			md, _ := metadata.FromIncomingContext(ctx)
			identity, _ := GetIdentityFromMetadata(md)
			got <- identity
			return handler(ctx, req)
		}))
	healthpb.RegisterHealthServer(srv, health.NewServer())
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)

	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) { return lis.DialContext(ctx) }),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return conn
}

// A non-ASCII name in the identity must not fail the RPC: gRPC rejects a non-printable-ASCII value on a non "-bin" metadata key, which would otherwise take down every RPC for the whole account.
func TestSetIdentityInMetadata_NonASCIINameSurvivesRPC(t *testing.T) {
	t.Parallel()
	names := []string{"José Ünïcode", "山田太郎", "Ada 🚀 Lovelace", "plain ascii"}
	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got := make(chan *types.Identity, 1)
			conn := serveIdentityEcho(t, got)

			md := metadata.New(nil)
			SetIdentityInMetadata(md, &types.Identity{
				Type:  types.IdentityActorTypeUser,
				Actor: &types.IdentityActor{ID: "usr_1", Name: strPtr(name), RoleName: strPtr(name)},
			})
			ctx := metadata.NewOutgoingContext(t.Context(), md)

			if _, err := healthpb.NewHealthClient(conn).Check(ctx, &healthpb.HealthCheckRequest{}); err != nil {
				t.Fatalf("RPC failed carrying name %q: %v", name, err)
			}
			received := <-got
			if received == nil || received.Actor == nil || received.Actor.Name == nil {
				t.Fatalf("identity did not survive the wire: %+v", received)
			}
			if *received.Actor.Name != name {
				t.Errorf("name = %q, want %q", *received.Actor.Name, name)
			}
			if *received.Actor.RoleName != name {
				t.Errorf("role name = %q, want %q", *received.Actor.RoleName, name)
			}
		})
	}
}

// Pins the reason the escaping exists: the raw JSON gRPC would have carried before this fix is rejected by the transport.
func TestSetIdentityInMetadata_RawNonASCIIWouldFailRPC(t *testing.T) {
	t.Parallel()
	got := make(chan *types.Identity, 1)
	conn := serveIdentityEcho(t, got)

	raw, err := json.Marshal(&types.Identity{
		Type:  types.IdentityActorTypeUser,
		Actor: &types.IdentityActor{ID: "usr_1", Name: strPtr("José")},
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	md := metadata.New(nil)
	md.Set(IdentityHeader, string(raw))
	ctx := metadata.NewOutgoingContext(t.Context(), md)

	if _, err := healthpb.NewHealthClient(conn).Check(ctx, &healthpb.HealthCheckRequest{}); err == nil {
		t.Fatal("expected the unescaped non-ASCII header to be rejected by gRPC; if this now passes, asciiEscapeJSON may no longer be needed")
	}
}

// The escaped form must be pure printable ASCII, which is what gRPC requires of a non "-bin" metadata value.
func TestAsciiEscapeJSON_OutputIsPrintableASCII(t *testing.T) {
	t.Parallel()
	for _, in := range []string{"José", "山田", "🚀", "tab\there", "del\x7fbyte", "plain"} {
		b, err := json.Marshal(map[string]string{"n": in})
		if err != nil {
			t.Fatalf("marshal %q: %v", in, err)
		}
		out := asciiEscapeJSON(b)
		for i := 0; i < len(out); i++ {
			if out[i] < 0x20 || out[i] > 0x7e {
				t.Errorf("%q: byte %d = %#x is outside printable ASCII", in, i, out[i])
			}
		}
		var back map[string]string
		if err := json.Unmarshal([]byte(out), &back); err != nil {
			t.Fatalf("%q: escaped form is not valid JSON: %v", in, err)
		}
		if back["n"] != in {
			t.Errorf("round-trip of %q gave %q", in, back["n"])
		}
	}
}
