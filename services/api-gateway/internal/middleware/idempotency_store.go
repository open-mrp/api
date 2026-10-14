package middleware

import (
	"context"
	"encoding/json"
	"net/http"

	grpcclient "github.com/open-mrp/api/services/api-gateway/grpc-client"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/contracts"
	pb "github.com/open-mrp/api/shared/proto/platform"
	"github.com/open-mrp/apikit/appctx"
	"github.com/open-mrp/apikit/idempotency"
)

// IdempotencyStore keeps idempotency keys in platform-service, for apikit's idempotency middleware.
type IdempotencyStore struct {
	client *grpcclient.PlatformServiceClient
}

// NewIdempotencyStore returns a store over platform-service's idempotency RPCs.
func NewIdempotencyStore(client *grpcclient.PlatformServiceClient) *IdempotencyStore {
	return &IdempotencyStore{client: client}
}

// IdempotencyScope is the caller and the account it acts in, so one caller's key never replays another's response.
func IdempotencyScope(r *http.Request) (callerID, tenantID *string) {
	identity, _ := appctx.Identity[*types.Identity](r.Context())
	if identity != nil && identity.Actor != nil {
		callerID = &identity.Actor.ID
	}
	if identity.IsTargetAccountSet() {
		tenantID = &identity.Target.AccountID
	}
	return callerID, tenantID
}

// Begin records the key, or reports what an earlier request with it left behind.
func (s *IdempotencyStore) Begin(ctx context.Context, req idempotency.BeginRequest) (idempotency.BeginResult, error) {
	identityType := string(types.IdentityActorTypeUnauthenticated)
	if identity, _ := appctx.Identity[*types.Identity](ctx); identity != nil && identity.Actor != nil {
		identityType = string(identity.Type)
	}
	resp, err := s.client.Client.ProcessIdempotencyKey(ctx, &pb.ProcessIdempotencyKeyRequest{
		ActorId:         req.CallerID,
		IdentityType:    identityType,
		RequestId:       req.RequestID,
		RequestMethod:   req.Method,
		NormalizedRoute: req.Route,
		IdempotencyKey:  req.Key,
		ScopeHash:       req.ScopeHash,
		RequestBodyHash: req.RequestHash,
		TargetAccountId: req.TenantID,
	})
	if err != nil {
		return idempotency.BeginResult{}, contracts.ConvertGRPCError(ctx, err, "platform-service")
	}
	res := idempotency.BeginResult{KeyID: resp.IdempotencyKeyId}
	switch resp.Result {
	case pb.ProcessIdempotencyKeyResult_PROCESS_RESULT_HASH_MISMATCH:
		res.Outcome = idempotency.OutcomeMismatch
	case pb.ProcessIdempotencyKeyResult_PROCESS_RESULT_IN_PROGRESS:
		res.Outcome = idempotency.OutcomeInProgress
	case pb.ProcessIdempotencyKeyResult_PROCESS_RESULT_REPLAY:
		res.Outcome = idempotency.OutcomeReplay
		stored := &idempotency.StoredResponse{Body: resp.ResponseBody, Cookies: decodeCookies(resp.ResponseHeaders)}
		if resp.ResponseCode != nil {
			stored.StatusCode = int(*resp.ResponseCode)
		}
		res.Response = stored
	default:
		res.Outcome = idempotency.OutcomeNew
	}
	return res, nil
}

// Complete stores the response the request produced, for replays.
func (s *IdempotencyStore) Complete(ctx context.Context, keyID string, resp idempotency.StoredResponse) error {
	req := &pb.SetIdempotencyKeyResponseRequest{
		IdempotencyKeyId: keyID,
		ResponseCode:     int32(resp.StatusCode), // #nosec G115 - HTTP status code, always fits int32
		ResponseBody:     resp.Body,
	}
	if len(resp.Cookies) > 0 {
		req.ResponseHeaders = encodeCookies(resp.Cookies)
		ttl := int32(resp.CookieTTLSeconds) // #nosec G115 - a few minutes
		req.TtlSeconds = &ttl
	}
	if _, err := s.client.Client.SetIdempotencyKeyResponse(ctx, req); err != nil {
		return contracts.ConvertGRPCError(ctx, err, "platform-service")
	}
	return nil
}

// Release frees the key so a retry runs again, after a transient failure.
func (s *IdempotencyStore) Release(ctx context.Context, keyID string) error {
	if _, err := s.client.Client.ReleaseIdempotencyKey(ctx, &pb.ReleaseIdempotencyKeyRequest{IdempotencyKeyId: keyID}); err != nil {
		return contracts.ConvertGRPCError(ctx, err, "platform-service")
	}
	return nil
}

// storedCookie is how a Set-Cookie is kept with a stored response.
type storedCookie struct {
	Name     string `json:"name"`
	Value    string `json:"value"`
	Path     string `json:"path,omitempty"`
	Domain   string `json:"domain,omitempty"`
	MaxAge   int    `json:"max_age,omitempty"`
	Secure   bool   `json:"secure,omitempty"`
	HttpOnly bool   `json:"http_only,omitempty"`
	SameSite string `json:"same_site,omitempty"`
}

var sameSiteNames = map[http.SameSite]string{http.SameSiteLaxMode: "Lax", http.SameSiteStrictMode: "Strict", http.SameSiteNoneMode: "None"}

func encodeCookies(cookies []*http.Cookie) []byte {
	out := make([]storedCookie, 0, len(cookies))
	for _, c := range cookies {
		out = append(out, storedCookie{Name: c.Name, Value: c.Value, Path: c.Path, Domain: c.Domain, MaxAge: c.MaxAge, Secure: c.Secure, HttpOnly: c.HttpOnly, SameSite: sameSiteNames[c.SameSite]})
	}
	b, _ := json.Marshal(out)
	return b
}

func decodeCookies(raw []byte) []*http.Cookie {
	if len(raw) == 0 {
		return nil
	}
	var stored []storedCookie
	if err := json.Unmarshal(raw, &stored); err != nil {
		return nil
	}
	cookies := make([]*http.Cookie, 0, len(stored))
	for _, sc := range stored {
		c := &http.Cookie{Name: sc.Name, Value: sc.Value, Path: sc.Path, Domain: sc.Domain, MaxAge: sc.MaxAge, Secure: sc.Secure, HttpOnly: sc.HttpOnly} // #nosec G124 - replays the stored cookie's original attributes
		for mode, name := range sameSiteNames {
			if name == sc.SameSite {
				c.SameSite = mode
			}
		}
		cookies = append(cookies, c)
	}
	return cookies
}
