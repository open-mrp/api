package contracts

import (
	"context"
	"encoding/json"

	"github.com/augno/api/shared/appctx"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

const (
	// IdempotentReplayedHeader is the gRPC response metadata key that signals a
	// cached (replayed) response. Clients can inspect this to know whether the
	// server executed the request or returned a stored result.
	IdempotentReplayedHeader = "x-idempotent-replayed"
	// IdempotentReplayedHeaderValue is the value set on IdempotentReplayedHeader
	// when the response was served from cache.
	IdempotentReplayedHeaderValue = "true"
)

// SetIdempotentReplayed sets the idempotent replayed header on the gRPC response.
func SetIdempotentReplayed(ctx context.Context) {
	_ = grpc.SetHeader(ctx, metadata.Pairs(IdempotentReplayedHeader, IdempotentReplayedHeaderValue))
}

// IsIdempotentReplayed checks if the idempotent replayed header is set in the gRPC metadata.
func IsIdempotentReplayed(md metadata.MD) bool {
	values := md.Get(IdempotentReplayedHeader)
	return len(values) > 0 && values[0] == IdempotentReplayedHeaderValue
}

// WithIdempotencyTracking sets up idempotency response tracking for a gRPC handler.
// It returns the updated context and a finalize function that should be deferred.
// The finalize function will set the appropriate gRPC header if the response was replayed.
//
// Usage:
//
//	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
//	defer finalizeIdempotency()
func WithIdempotencyTracking(ctx context.Context) (context.Context, func()) {
	meta := &appctx.IdempotencyResponseMetadata{}
	ctx = appctx.WithIdempotencyResponseMetadata(ctx, meta)
	return ctx, func() {
		if meta.Replayed {
			SetIdempotentReplayed(ctx)
		}
	}
}

// IdempotencyChecker is the interface for request deduplication backends.
// Implementations check whether a request has already been processed and store
// responses for future replay.
type IdempotencyChecker interface {
	// CheckIdempotency looks up key and returns the cached result if one exists.
	CheckIdempotency(ctx context.Context, key string) (*IdempotencyCheckResult, error)
	// StoreResponse persists the handler's response so future requests with the
	// same key can be replayed.
	StoreResponse(ctx context.Context, key string, response proto.Message) error
}

// IdempotencyCheckResult is the outcome of an idempotency lookup.
type IdempotencyCheckResult struct {
	// Exists is true when a previous response was found for the key.
	Exists bool
	// ResponseCode is the gRPC status code of the cached response.
	ResponseCode int
	// ResponseData is the serialized response body from the original handler invocation.
	ResponseData []byte
}

type idempotencyInterceptorConfig struct {
	checker IdempotencyChecker
	methods map[string]bool
}

// IdempotencyInterceptorOption configures the idempotency server interceptor.
type IdempotencyInterceptorOption func(*idempotencyInterceptorConfig)

// WithIdempotencyChecker sets the IdempotencyChecker implementation used by the
// interceptor. If none is provided the interceptor is a no-op passthrough.
func WithIdempotencyChecker(checker IdempotencyChecker) IdempotencyInterceptorOption {
	return func(cfg *idempotencyInterceptorConfig) {
		cfg.checker = checker
	}
}

// WithIdempotentMethods restricts idempotency checking to the listed gRPC full method
// names (e.g. "/auth.AuthService/Login"). When empty, all methods are checked.
func WithIdempotentMethods(methods ...string) IdempotencyInterceptorOption {
	return func(cfg *idempotencyInterceptorConfig) {
		for _, method := range methods {
			cfg.methods[method] = true
		}
	}
}

// IdempotencyUnaryServerInterceptor returns a server interceptor that deduplicates
// requests using the idempotency key from incoming metadata. On a cache hit the stored
// response is returned immediately and the handler is skipped. On a cache miss the
// handler runs and its response is stored for future replay.
func IdempotencyUnaryServerInterceptor(opts ...IdempotencyInterceptorOption) grpc.UnaryServerInterceptor {
	cfg := &idempotencyInterceptorConfig{
		methods: make(map[string]bool),
	}

	for _, opt := range opts {
		opt(cfg)
	}

	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if cfg.checker == nil {
			return handler(ctx, req)
		}

		if len(cfg.methods) > 0 && !cfg.methods[info.FullMethod] {
			return handler(ctx, req)
		}

		idempotencyKey := GetIdempotencyKeyFromContext(ctx)
		if idempotencyKey == nil || *idempotencyKey == "" {
			return handler(ctx, req)
		}

		result, err := cfg.checker.CheckIdempotency(ctx, *idempotencyKey)
		if err != nil {
			return handler(ctx, req)
		}

		if result.Exists && result.ResponseData != nil {
			resp, err := unmarshalResponse(result.ResponseData)
			if err != nil {
				return handler(ctx, req)
			}

			SetIdempotentReplayed(ctx)
			return resp, nil
		}

		resp, handlerErr := handler(ctx, req)
		if handlerErr != nil {
			return resp, handlerErr
		}

		if protoResp, ok := resp.(proto.Message); ok {
			_ = cfg.checker.StoreResponse(ctx, *idempotencyKey, protoResp)
		}

		return resp, handlerErr
	}
}

// unmarshalResponse decodes a JSON-encoded cached response into a generic map.
func unmarshalResponse(data []byte) (any, error) {
	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// LocalIdempotencyChecker is an in-memory IdempotencyChecker backed by a simple map.
// Intended for tests and single-instance development; not safe for concurrent use.
type LocalIdempotencyChecker struct {
	cache map[string]*IdempotencyCheckResult
}

// NewLocalIdempotencyChecker creates a LocalIdempotencyChecker with an empty cache.
func NewLocalIdempotencyChecker() *LocalIdempotencyChecker {
	return &LocalIdempotencyChecker{
		cache: make(map[string]*IdempotencyCheckResult),
	}
}

// CheckIdempotency looks up key in the local cache and returns the stored result if found.
func (c *LocalIdempotencyChecker) CheckIdempotency(ctx context.Context, key string) (*IdempotencyCheckResult, error) {
	if result, exists := c.cache[key]; exists {
		return result, nil
	}
	return &IdempotencyCheckResult{Exists: false}, nil
}

// StoreResponse serializes response with proto.Marshal and stores it in the local cache.
func (c *LocalIdempotencyChecker) StoreResponse(ctx context.Context, key string, response proto.Message) error {
	data, err := proto.Marshal(response)
	if err != nil {
		return err
	}
	c.cache[key] = &IdempotencyCheckResult{
		Exists:       true,
		ResponseCode: int(codes.OK),
		ResponseData: data,
	}
	return nil
}

// PlatformIdempotencyChecker delegates idempotency checks and response storage to the
// platform service via callback functions. Used in production for database-backed deduplication.
type PlatformIdempotencyChecker struct {
	processFunc     func(ctx context.Context, key string, requestParams []byte) (*ProcessIdempotencyResult, error)
	setResponseFunc func(ctx context.Context, keyID string, statusCode int, body []byte) error
	currentKeyID    string
}

// ProcessIdempotencyResult is the response from the platform service's idempotency check.
type ProcessIdempotencyResult struct {
	// IsNew is true when the key has never been seen before.
	IsNew bool
	// IsReplay is true when a completed response already exists for the key.
	IsReplay bool
	// IsInProgress is true when another request with the same key is currently executing.
	IsInProgress bool
	// KeyID is the platform-assigned identifier for this idempotency key.
	KeyID string
	// ResponseCode is the gRPC status code of the stored response (valid when IsReplay is true).
	ResponseCode int
	// ResponseBody is the serialized response body (valid when IsReplay is true).
	ResponseBody []byte
}

// NewPlatformIdempotencyChecker creates a PlatformIdempotencyChecker with the given
// callbacks for processing idempotency keys and storing responses.
func NewPlatformIdempotencyChecker(
	processFunc func(ctx context.Context, key string, requestParams []byte) (*ProcessIdempotencyResult, error),
	setResponseFunc func(ctx context.Context, keyID string, statusCode int, body []byte) error,
) *PlatformIdempotencyChecker {
	return &PlatformIdempotencyChecker{
		processFunc:     processFunc,
		setResponseFunc: setResponseFunc,
	}
}

// CheckIdempotency delegates the idempotency lookup to the platform service via processFunc.
func (c *PlatformIdempotencyChecker) CheckIdempotency(ctx context.Context, key string) (*IdempotencyCheckResult, error) {
	result, err := c.processFunc(ctx, key, nil)
	if err != nil {
		return nil, err
	}

	c.currentKeyID = result.KeyID

	if result.IsReplay {
		return &IdempotencyCheckResult{
			Exists:       true,
			ResponseCode: result.ResponseCode,
			ResponseData: result.ResponseBody,
		}, nil
	}

	if result.IsInProgress {
		return nil, status.Error(codes.Aborted, "request in progress")
	}

	return &IdempotencyCheckResult{Exists: false}, nil
}

// StoreResponse persists the handler's response via setResponseFunc so it can be replayed.
func (c *PlatformIdempotencyChecker) StoreResponse(ctx context.Context, key string, response proto.Message) error {
	if c.setResponseFunc == nil || c.currentKeyID == "" {
		return nil
	}

	data, err := json.Marshal(response)
	if err != nil {
		return err
	}

	return c.setResponseFunc(ctx, c.currentKeyID, int(codes.OK), data)
}
