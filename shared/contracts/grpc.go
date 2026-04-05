package contracts

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/appctx"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/version"
	"google.golang.org/grpc"
	grpccodes "google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	grpcstatus "google.golang.org/grpc/status"
)

const (
	// apiErrorMarker is used to mark gRPC messages that contain encoded API error information
	apiErrorMarker = "__API_ERROR__:"
	// IdentityHeader is the header name for the identity in the metadata
	IdentityHeader = "identity"
	// IdempotencyKeyHeader is the header name for the raw idempotency key string in the metadata
	IdempotencyKeyHeader = "idempotency-key"
	// IdempotencyKeyIDHeader is the header name for the idempotency key database ID in the metadata
	IdempotencyKeyIDHeader = "idempotency-key-id"
	// APIVersionHeader is the header name for the API version in the metadata
	APIVersionHeader = "api-version"
	// RequestIDHeader is the header name for the request ID in the metadata
	RequestIDHeader = "request-id"
	// ClientIPHeader is the header name for the originating HTTP client IP when
	// the API gateway forwards a user request to a backend service.
	ClientIPHeader = "client-ip"
)

// GetIdentityFromMetadata extracts the caller's identity from gRPC incoming metadata.
// Returns an APIError if the header is missing, duplicated, or malformed.
func GetIdentityFromMetadata(md metadata.MD) (*types.Identity, *apierror.APIError) {
	data := md.Get(IdentityHeader)
	if len(data) == 0 {
		return nil, apierror.NewInvariantViolationError("Identity not found in metadata.") // #nosec G101
	} else if len(data) > 1 {
		return nil, apierror.NewInvariantViolationError(fmt.Sprintf("Identity metadata malformed: %s", strings.Join(data, ", "))) // #nosec G101
	}
	var identity types.Identity
	err := json.Unmarshal([]byte(data[0]), &identity)
	if err != nil {
		return nil, apierror.NewInternalError(err, "Identity metadata malformed.")
	}
	return &identity, nil
}

// SetIdentityInMetadata serializes the identity as JSON and sets it on the outgoing gRPC metadata.
// Silently drops the value if marshalling fails.
func SetIdentityInMetadata(md metadata.MD, identity *types.Identity) {
	jsonData, err := json.Marshal(identity)
	if err != nil {
		return
	}
	md.Set(IdentityHeader, string(jsonData))
}

// IdentityUnaryServerInterceptor returns a server interceptor that extracts the caller's
// identity from incoming gRPC metadata and stores it in the context. If the header is
// absent or invalid the request proceeds without an identity.
func IdentityUnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		md, ok := metadata.FromIncomingContext(ctx)
		if ok {
			if identity, err := GetIdentityFromMetadata(md); err == nil {
				ctx = appctx.WithIdentity(ctx, identity)
			}
		}
		return handler(ctx, req)
	}
}

// IdempotencyKeyUnaryServerInterceptor returns a server interceptor that extracts the
// idempotency key and the handler's full method name from incoming metadata and stores
// them in the context.
func IdempotencyKeyUnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		ctx = appctx.WithHandler(ctx, info.FullMethod)
		md, ok := metadata.FromIncomingContext(ctx)
		if ok {
			values := md.Get(IdempotencyKeyHeader)
			if len(values) > 0 && values[0] != "" {
				ctx = appctx.WithIdempotencyKey(ctx, values[0])
			}
		}
		return handler(ctx, req)
	}
}

// ConvertGRPCError translates a gRPC status error into an APIError. If the status message
// contains an encoded APIError (prefixed with apiErrorMarker) the original error is
// reconstructed losslessly. Otherwise the gRPC code is mapped to a generic APIError.
//
// ctx should be the original HTTP request context so that client disconnections
// (context.Canceled on the request) can be distinguished from server-side timeouts.
func ConvertGRPCError(ctx context.Context, err error, serviceName string) *apierror.APIError {
	if err == nil {
		return nil
	}

	// If the error is not a gRPC status error, check for context errors.
	st, ok := grpcstatus.FromError(err)
	if !ok {
		if isClientCancellation(ctx, err) {
			return apierror.NewClientClosedRequestError("Client closed the connection.")
		}
		if isDeadlineError(ctx, err) {
			return apierror.NewRequestTimeoutError(serviceName + " request timed out.")
		}
		return apierror.NewInternalError(err, serviceName+" RPC failed.")
	}

	message := st.Message()

	// Check if message contains encoded API error information
	if jsonData, ok := strings.CutPrefix(message, apiErrorMarker); ok {
		if apiErr, err := apierror.APIErrorFromJSON([]byte(jsonData)); err == nil {
			return apiErr
		}
	}

	// If there is not an API error, this is an uncaught gRPC error
	switch st.Code() {
	case grpccodes.NotFound:
		return apierror.NewResourceNotFoundError(message)
	case grpccodes.Canceled:
		if isClientCancellation(ctx, err) {
			return apierror.NewClientClosedRequestError("Client closed the connection.")
		}
		return apierror.NewInternalError(err, fmt.Sprintf("Service %s encountered the error: %s.", serviceName, err.Error()))
	case grpccodes.DeadlineExceeded:
		if isClientCancellation(ctx, err) {
			return apierror.NewClientClosedRequestError("Client closed the connection.")
		}
		return apierror.NewRequestTimeoutError(fmt.Sprintf("Service %s encountered the error: %s.", serviceName, err.Error()))
	default:
		return apierror.NewInternalError(err, fmt.Sprintf("Service %s encountered the error: %s.", serviceName, err.Error()))
	}
}

// NewMissingGRPCRequestDataError returns a gRPC-encoded invariant violation for a nil request payload.
func NewMissingGRPCRequestDataError() error {
	apiErr := apierror.NewInvariantViolationError("Missing gRPC request data.")
	return ConvertAPIErrorToGRPC(apiErr)
}

// NewMissingIdentityMetadataError returns a gRPC-encoded invariant violation for a missing identity header.
func NewMissingIdentityMetadataError() error {
	apiErr := apierror.NewInvariantViolationError("Missing identity metadata.")
	return ConvertAPIErrorToGRPC(apiErr)
}

// ConvertAPIErrorToGRPC encodes an APIError as a gRPC status error. The full APIError JSON
// is embedded in the status message (prefixed with apiErrorMarker) so that ConvertGRPCError
// can reconstruct it losslessly. The gRPC status code is chosen to match the APIError category.
func ConvertAPIErrorToGRPC(apiErr *apierror.APIError) error {
	if apiErr == nil {
		return nil
	}

	// Encode the API error to JSON
	jsonData, err := apiErr.ToJSON()
	if err != nil {
		return grpcstatus.Error(grpccodes.Internal, fmt.Sprintf("Failed to encode API error: %s", apiErr.Error()))
	}

	// Prepend marker to indicate this is an encoded API error
	encodedMessage := apiErrorMarker + string(jsonData)

	// Map to appropriate gRPC code based on error code
	var grpcCode grpccodes.Code
	switch apiErr.Code {
	// Authentication errors
	case apierror.ErrorCodeInvalidCredentials, apierror.ErrorCodeExpiredToken, apierror.ErrorCodeExpiredAPIKey:
		grpcCode = grpccodes.Unauthenticated

	// Authorization errors
	case apierror.ErrorCodeInsufficientPerms:
		grpcCode = grpccodes.PermissionDenied

	// Validation errors
	case apierror.ErrorCodeValidationFailed, apierror.ErrorCodeMissingField, apierror.ErrorCodeInvalidFormat,
		apierror.ErrorCodeParameterMissing, apierror.ErrorCodeParameterInvalid, apierror.ErrorCodeParameterUnknown,
		apierror.ErrorCodeParametersExclusive:
		grpcCode = grpccodes.InvalidArgument

	// Resource errors
	case apierror.ErrorCodeResourceNotFound:
		grpcCode = grpccodes.NotFound
	case apierror.ErrorCodeResourceGone:
		grpcCode = grpccodes.FailedPrecondition
	case apierror.ErrorCodeResourceExists:
		grpcCode = grpccodes.AlreadyExists
	case apierror.ErrorCodeResourceConflict:
		grpcCode = grpccodes.Aborted

	// Business logic errors
	case apierror.ErrorCodeRateLimitExceeded, apierror.ErrorCodeLimitExceeded:
		grpcCode = grpccodes.ResourceExhausted

	// Server errors
	case apierror.ErrorCodeSvcUnavailable:
		grpcCode = grpccodes.Unavailable
	case apierror.ErrorCodeRequestTimeout, apierror.ErrorCodeTimeout:
		grpcCode = grpccodes.DeadlineExceeded
	case apierror.ErrorCodeMethodNotAllowed:
		grpcCode = grpccodes.Unimplemented

	// Default to Internal for all other errors
	case apierror.ErrorCodeInternalError, apierror.ErrorCodeExternalSvcError, apierror.ErrorCodeConnectionError,
		apierror.ErrorCodeClientClosedRequest:
		fallthrough
	default:
		grpcCode = grpccodes.Internal
	}

	return grpcstatus.Error(grpcCode, encodedMessage)
}

// isClientCancellation reports whether the original request context was canceled,
// indicating the client disconnected before the server finished processing.
// This checks the parent context (typically the HTTP request context), not the
// RPC child context, so a canceled parent means the client went away.
func isClientCancellation(ctx context.Context, err error) bool {
	return ctx.Err() == context.Canceled ||
		(errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded))
}

// isDeadlineError reports whether err or the context indicate a deadline/timeout.
func isDeadlineError(ctx context.Context, err error) bool {
	return errors.Is(err, context.DeadlineExceeded) || ctx.Err() == context.DeadlineExceeded
}

// GetIdempotencyKeyFromContext extracts the idempotency key from gRPC metadata.
func GetIdempotencyKeyFromContext(ctx context.Context) *string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil
	}
	values := md.Get(IdempotencyKeyHeader)
	if len(values) == 0 {
		return nil
	}
	return &values[0]
}

// SetAPIVersionInMetadata sets the API version in the metadata.
func SetAPIVersionInMetadata(md metadata.MD, version string) {
	md.Set(APIVersionHeader, version)
}

// GetAPIVersionFromMetadata extracts the API version from the metadata.
func GetAPIVersionFromMetadata(md metadata.MD) string {
	values := md.Get(APIVersionHeader)
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

// SetRequestIDInMetadata sets the request ID in the metadata.
func SetRequestIDInMetadata(md metadata.MD, requestID string) {
	md.Set(RequestIDHeader, requestID)
}

// GetRequestIDFromMetadata extracts the request ID from the metadata.
func GetRequestIDFromMetadata(md metadata.MD) string {
	values := md.Get(RequestIDHeader)
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

// APIVersionUnaryServerInterceptor extracts the API version from gRPC metadata and adds it to the context.
func APIVersionUnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		md, ok := metadata.FromIncomingContext(ctx)
		if ok {
			if versionStr := GetAPIVersionFromMetadata(md); versionStr != "" {
				if v, err := version.Parse(versionStr); err == nil {
					ctx = appctx.WithAPIVersion(ctx, v)
				}
			}
		}
		return handler(ctx, req)
	}
}

// RequestIDUnaryServerInterceptor extracts the request ID from gRPC metadata and adds it to the context.
func RequestIDUnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		md, ok := metadata.FromIncomingContext(ctx)
		if ok {
			if requestID := GetRequestIDFromMetadata(md); requestID != "" {
				ctx = appctx.WithRequestID(ctx, requestID)
			}
		}
		return handler(ctx, req)
	}
}

// GetClientIPFromMetadata extracts the client IP from gRPC metadata.
func GetClientIPFromMetadata(md metadata.MD) string {
	values := md.Get(ClientIPHeader)
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

// ClientIPUnaryServerInterceptor extracts the client IP from gRPC metadata and
// adds it to the context for downstream use (e.g. audit events).
func ClientIPUnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		md, ok := metadata.FromIncomingContext(ctx)
		if ok {
			if ip := GetClientIPFromMetadata(md); ip != "" {
				ctx = appctx.WithPropagatedClientIP(ctx, ip)
			}
		}
		return handler(ctx, req)
	}
}
