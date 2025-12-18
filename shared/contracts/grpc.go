package contracts

import (
	"context"
	"errors"
	"fmt"
	"strings"

	grpccodes "google.golang.org/grpc/codes"
	grpcstatus "google.golang.org/grpc/status"
)

const (
	// apiErrorMarker is used to mark gRPC messages that contain encoded API error information
	apiErrorMarker = "__API_ERROR__:"
)

// ConvertGRPCError converts a gRPC error to an APIError.
func ConvertGRPCError(ctx context.Context, err error, serviceName string) *APIError {
	if err == nil {
		return nil
	}

	st, ok := grpcstatus.FromError(err)
	if !ok {
		if isDeadlineError(ctx, err) {
			return NewRequestTimeoutError(serviceName + " request timed out.")
		}
		return NewInternalError(err, serviceName+" RPC failed")
	}

	message := st.Message()

	// Check if message contains encoded API error information
	if jsonData, ok := strings.CutPrefix(message, apiErrorMarker); ok {
		if apiErr, err := apiErrorFromJSON([]byte(jsonData)); err == nil {
			return apiErr
		}
	}

	// If there is not an API error, this is an uncaught gRPC error
	switch st.Code() {
	case grpccodes.DeadlineExceeded:
		return NewRequestTimeoutError(fmt.Sprintf("Service %s encountered the error: %s", serviceName, err.Error()))
	default:
		return NewInternalError(err, fmt.Sprintf("Service %s encountered the error: %s", serviceName, err.Error()))
	}
}

func NewMissingGRPCRequestDataError() error {
	apiErr := NewInternalError(errors.New("missing gRPC request data"), "Missing gRPC request data")
	return ConvertAPIErrorToGRPC(apiErr)
}

func NewMissingIdentityMetadataError() error {
	apiErr := NewInternalError(errors.New("missing identity metadata"), "Missing identity metadata")
	return ConvertAPIErrorToGRPC(apiErr)
}

// ConvertAPIErrorToGRPC converts an APIError to a gRPC status error.
func ConvertAPIErrorToGRPC(apiErr *APIError) error {
	if apiErr == nil {
		return nil
	}

	jsonData, err := apiErr.toJSON()
	if err != nil {
		return grpcstatus.Error(grpccodes.Internal, fmt.Sprintf("Failed to encode API error: %s", apiErr.Error()))
	}

	// Prepend marker to indicate this is an encoded API error
	encodedMessage := apiErrorMarker + string(jsonData)

	// Map to appropriate gRPC code based on error code
	var grpcCode grpccodes.Code
	switch apiErr.Code {
	// Authentication errors
	case ErrorCodeInvalidCredentials, ErrorCodeExpiredToken, ErrorCodeExpiredAPIKey:
		grpcCode = grpccodes.Unauthenticated

	// Authorization errors
	case ErrorCodeInsufficientPerms:
		grpcCode = grpccodes.PermissionDenied

	// Validation errors
	case ErrorCodeValidationFailed, ErrorCodeMissingField, ErrorCodeInvalidFormat,
		ErrorCodeParameterMissing, ErrorCodeParameterInvalid, ErrorCodeParameterUnknown,
		ErrorCodeParametersExclusive:
		grpcCode = grpccodes.InvalidArgument

	// Resource errors
	case ErrorCodeResourceNotFound:
		grpcCode = grpccodes.NotFound
	case ErrorCodeResourceExists:
		grpcCode = grpccodes.AlreadyExists
	case ErrorCodeResourceConflict:
		grpcCode = grpccodes.Aborted

	// Business logic errors
	case ErrorCodeRateLimitExceeded:
		grpcCode = grpccodes.ResourceExhausted

	// Server errors
	case ErrorCodeSvcUnavailable:
		grpcCode = grpccodes.Unavailable
	case ErrorCodeRequestTimeout, ErrorCodeTimeout:
		grpcCode = grpccodes.DeadlineExceeded
	case ErrorCodeMethodNotAllowed:
		grpcCode = grpccodes.Unimplemented

	// Default to Internal for all other errors
	case ErrorCodeInternalError, ErrorCodeExternalSvcError, ErrorCodeConnectionError,
		ErrorCodeClientClosedRequest:
		fallthrough
	default:
		grpcCode = grpccodes.Internal
	}

	return grpcstatus.Error(grpcCode, encodedMessage)
}

func isDeadlineError(ctx context.Context, err error) bool {
	return errors.Is(err, context.DeadlineExceeded) || ctx.Err() == context.DeadlineExceeded
}
