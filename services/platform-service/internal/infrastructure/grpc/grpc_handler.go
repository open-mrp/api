package grpc

import (
	"context"
	"errors"

	"github.com/augno/api/services/platform-service/internal/domain"
	"github.com/augno/api/services/platform-service/pkg/idempotency"
	"github.com/augno/api/shared/contracts"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/id"
	pb "github.com/augno/api/shared/proto/platform"
	"github.com/augno/api/shared/tracing"

	"google.golang.org/grpc"
)

var idempotencyGRPCHandlerTracer = tracing.GetTracer("platform-service.grpc_handler")

type gRPCHandler struct {
	pb.UnimplementedIdempotencyServiceServer

	idempotencyRepo domain.IdempotencyKeyRepo
}

func NewGRPCHandler(server *grpc.Server, idempotencyRepo domain.IdempotencyKeyRepo) *gRPCHandler {
	handler := &gRPCHandler{
		idempotencyRepo: idempotencyRepo,
	}

	pb.RegisterIdempotencyServiceServer(server, handler)
	return handler
}

func (h *gRPCHandler) ProcessIdempotencyKey(ctx context.Context, req *pb.ProcessIdempotencyKeyRequest) (*pb.ProcessIdempotencyKeyResponse, error) {
	ctx, span := idempotencyGRPCHandlerTracer.Start(ctx, "grpc_handler.process_idempotency_key")
	defer span.End()

	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	typeID, err := id.GenID(id.IdempotencyKeyIDPrefix, nil)
	if err != nil {
		tracing.Trace(span, err)
		return nil, contracts.ConvertAPIErrorToGRPC(err)
	}

	key := &domain.IdempotencyKey{
		ID:              typeID,
		IdempotencyKey:  req.IdempotencyKey,
		ActorID:         req.ActorId,
		TargetAccountID: req.TargetAccountId,
		IdentityType:    req.IdentityType,
		RequestMethod:   req.RequestMethod,
		NormalizedRoute: req.NormalizedRoute,
		ScopeHash:       req.ScopeHash,
		RequestBodyHash: req.RequestBodyHash,
		RequestParams:   req.RequestParams,
		RecoveryPoint:   idempotency.RecoveryPointStarted.String(),
	}

	result, upsertErr := h.idempotencyRepo.UpsertAndLock(ctx, key)
	if upsertErr != nil {
		if errors.Is(upsertErr, domain.ErrHashMismatch) {
			return &pb.ProcessIdempotencyKeyResponse{
				Result: pb.ProcessIdempotencyKeyResult_PROCESS_RESULT_HASH_MISMATCH,
			}, nil
		}
		if errors.Is(upsertErr, domain.ErrKeyLocked) {
			return &pb.ProcessIdempotencyKeyResponse{
				Result: pb.ProcessIdempotencyKeyResult_PROCESS_RESULT_IN_PROGRESS,
			}, nil
		}
		apiErr := apierror.NewInternalError(upsertErr, "Failed to process idempotency key.")
		tracing.Trace(span, apiErr)
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	if result.Created {
		return &pb.ProcessIdempotencyKeyResponse{
			Result:           pb.ProcessIdempotencyKeyResult_PROCESS_RESULT_NEW,
			IdempotencyKeyId: result.Key.ID,
		}, nil
	}

	if result.Key.HasResponse() {
		var responseCode *int32
		if result.Key.ResponseCode != nil {
			code := int32(*result.Key.ResponseCode) // #nosec G115 - HTTP status code
			responseCode = &code
		}
		return &pb.ProcessIdempotencyKeyResponse{
			Result:           pb.ProcessIdempotencyKeyResult_PROCESS_RESULT_REPLAY,
			IdempotencyKeyId: result.Key.ID,
			ResponseCode:     responseCode,
			ResponseBody:     result.Key.ResponseBody,
			ResponseHeaders:  result.Key.ResponseHeaders,
		}, nil
	}

	if result.Key.IsLocked() && !result.Locked {
		return &pb.ProcessIdempotencyKeyResponse{
			Result:           pb.ProcessIdempotencyKeyResult_PROCESS_RESULT_IN_PROGRESS,
			IdempotencyKeyId: result.Key.ID,
		}, nil
	}

	return &pb.ProcessIdempotencyKeyResponse{
		Result:           pb.ProcessIdempotencyKeyResult_PROCESS_RESULT_NEW,
		IdempotencyKeyId: result.Key.ID,
	}, nil
}

func (h *gRPCHandler) SetIdempotencyKeyResponse(ctx context.Context, req *pb.SetIdempotencyKeyResponseRequest) (*pb.SetIdempotencyKeyResponseResponse, error) {
	ctx, span := idempotencyGRPCHandlerTracer.Start(ctx, "grpc_handler.set_idempotency_key_response")
	defer span.End()

	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	params := domain.SetResponseParams{
		ID:            req.IdempotencyKeyId,
		StatusCode:    int(req.ResponseCode),
		Body:          req.ResponseBody,
		Headers:       req.ResponseHeaders,
		TTLSeconds:    req.TtlSeconds,
		RecoveryPoint: idempotency.RecoveryPointFinished.String(),
	}
	err := h.idempotencyRepo.SetResponse(ctx, params)
	if err != nil {
		if errors.Is(err, domain.ErrKeyNotFound) {
			apiErr := apierror.NewResourceNotFoundError("Idempotency key not found.")
			tracing.Trace(span, apiErr)
			return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
		}
		apiErr := apierror.NewInternalError(err, "Failed to set idempotency key response.")
		tracing.Trace(span, apiErr)
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.SetIdempotencyKeyResponseResponse{
		Success: true,
	}, nil
}

func (h *gRPCHandler) ReleaseIdempotencyKey(ctx context.Context, req *pb.ReleaseIdempotencyKeyRequest) (*pb.ReleaseIdempotencyKeyResponse, error) {
	ctx, span := idempotencyGRPCHandlerTracer.Start(ctx, "grpc_handler.release_idempotency_key")
	defer span.End()

	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	err := h.idempotencyRepo.ReleaseLock(ctx, req.IdempotencyKeyId)
	if err != nil {
		apiErr := apierror.NewInternalError(err, "Failed to release idempotency key lock.")
		tracing.Trace(span, apiErr)
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.ReleaseIdempotencyKeyResponse{
		Success: true,
	}, nil
}

func (h *gRPCHandler) AdvanceRecoveryPoint(ctx context.Context, req *pb.AdvanceRecoveryPointRequest) (*pb.AdvanceRecoveryPointResponse, error) {
	ctx, span := idempotencyGRPCHandlerTracer.Start(ctx, "grpc_handler.advance_recovery_point")
	defer span.End()

	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	err := h.idempotencyRepo.AdvanceRecoveryPoint(ctx, domain.AdvanceRecoveryPointParams{
		ID:            req.IdempotencyKeyId,
		RecoveryPoint: req.RecoveryPoint,
		StepData:      req.StepData,
	})
	if err != nil {
		if errors.Is(err, domain.ErrKeyNotFound) {
			apiErr := apierror.NewResourceNotFoundError("Idempotency key not found.")
			tracing.Trace(span, apiErr)
			return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
		}
		apiErr := apierror.NewInternalError(err, "Failed to advance recovery point.")
		tracing.Trace(span, apiErr)
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.AdvanceRecoveryPointResponse{
		Success: true,
	}, nil
}

func (h *gRPCHandler) GetRecoveryPoint(ctx context.Context, req *pb.GetRecoveryPointRequest) (*pb.GetRecoveryPointResponse, error) {
	ctx, span := idempotencyGRPCHandlerTracer.Start(ctx, "grpc_handler.get_recovery_point")
	defer span.End()

	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	result, err := h.idempotencyRepo.GetRecoveryPoint(ctx, req.IdempotencyKeyId)
	if err != nil {
		if errors.Is(err, domain.ErrKeyNotFound) {
			apiErr := apierror.NewResourceNotFoundError("Idempotency key not found.")
			tracing.Trace(span, apiErr)
			return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
		}
		apiErr := apierror.NewInternalError(err, "Failed to get recovery point.")
		tracing.Trace(span, apiErr)
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.GetRecoveryPointResponse{
		RecoveryPoint: result.RecoveryPoint,
		StepData:      result.StepData,
	}, nil
}
