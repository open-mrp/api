package grpc

import (
	"context"
	"time"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/contracts"
	pb "github.com/open-mrp/api/shared/proto/core"

	"google.golang.org/protobuf/types/known/timestamppb"
)

type batchReleaseGRPCHandler struct {
	pb.UnimplementedCoreBatchReleaseServiceServer

	batchReleaseSvc domain.BatchReleaseSvc
}

func releasePolicyToProto(p *domain.ReleasePolicy) *pb.ReleasePolicyInfo {
	return &pb.ReleasePolicyInfo{
		DepartmentId:    p.DepartmentID,
		DepartmentName:  p.DepartmentName,
		Requirement:     string(p.Requirement),
		Enforcement:     string(p.Enforcement),
		Rationale:       p.Rationale,
		Checklist:       p.Checklist,
		UpdatedAt:       timestamppb.New(p.UpdatedAt),
		OperatorSigning: string(p.OperatorSigning),
	}
}

func batchSignatureToProto(s *domain.BatchSignature) *pb.BatchSignatureInfo {
	checklist := make([]*pb.ChecklistResponseInfo, len(s.Checklist))
	for i, response := range s.Checklist {
		checklist[i] = &pb.ChecklistResponseInfo{Prompt: response.Prompt, Answer: string(response.Answer)}
	}
	return &pb.BatchSignatureInfo{
		Id:                  s.ID,
		BatchId:             s.BatchID,
		ProductionStepId:    s.ProductionStepID,
		ProductionRunId:     s.ProductionRunID,
		Type:                string(s.Type),
		Decision:            string(s.Decision),
		Capacity:            string(s.Capacity),
		SignerAccountUserId: s.SignerAccountUserID,
		SignerName:          s.SignerName,
		SignerRoleName:      s.SignerRoleName,
		Checklist:           checklist,
		Note:                s.Note,
		ActorId:             s.ActorID,
		ActorType:           s.ActorType,
		SignedAt:            timestamppb.New(s.SignedAt),
		LotId:               s.LotID,
		LotNumber:           s.LotNumber,
	}
}

func operatorCheckInToProto(c *domain.OperatorCheckIn) *pb.OperatorCheckInInfo {
	info := &pb.OperatorCheckInInfo{
		Id:                c.ID,
		ScanningStationId: c.ScanningStationID,
		AccountUserId:     c.AccountUserID,
		OperatorName:      c.OperatorName,
		Status:            string(c.StatusAt(time.Now())),
		CheckedInAt:       timestamppb.New(c.CheckedInAt),
		ExpiresAt:         timestamppb.New(c.ExpiresAt),
	}
	if c.CheckedOutAt != nil {
		info.CheckedOutAt = timestamppb.New(*c.CheckedOutAt)
	}
	return info
}

func (h *batchReleaseGRPCHandler) GetReleasePolicy(ctx context.Context, req *pb.GetReleasePolicyRequest) (*pb.ReleasePolicyInfo, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	policy, apiErr := h.batchReleaseSvc.GetReleasePolicy(ctx, req.DepartmentId)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return releasePolicyToProto(policy), nil
}

func (h *batchReleaseGRPCHandler) UpdateReleasePolicy(ctx context.Context, req *pb.UpdateReleasePolicyRequest) (*pb.ReleasePolicyInfo, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	policy, apiErr := h.batchReleaseSvc.UpdateReleasePolicy(ctx, domain.UpdateReleasePolicyParams{
		DepartmentID:    req.DepartmentId,
		Requirement:     constants.ReleaseRequirement(req.Requirement),
		Enforcement:     constants.ReleaseEnforcement(req.Enforcement),
		Rationale:       req.Rationale,
		Checklist:       req.Checklist,
		OperatorSigning: constants.ReleaseOperatorSigning(req.OperatorSigning),
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return releasePolicyToProto(policy), nil
}

func (h *batchReleaseGRPCHandler) ListEligibleSigners(ctx context.Context, req *pb.ListEligibleSignersRequest) (*pb.ListEligibleSignersResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	signers, apiErr := h.batchReleaseSvc.ListEligibleSigners(ctx, req.BatchId)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	out := make([]*pb.EligibleSignerInfo, len(signers))
	for i, signer := range signers {
		capacities := make([]string, len(signer.Capacities))
		for j, capacity := range signer.Capacities {
			capacities[j] = string(capacity)
		}
		out[i] = &pb.EligibleSignerInfo{
			AccountUserId: signer.AccountUserID,
			Name:          signer.Name,
			RoleName:      signer.RoleName,
			Capacities:    capacities,
		}
	}
	return &pb.ListEligibleSignersResponse{Signers: out}, nil
}

func (h *batchReleaseGRPCHandler) ListBatchSignatures(ctx context.Context, req *pb.ListBatchSignaturesRequest) (*pb.ListBatchSignaturesResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	signatures, apiErr := h.batchReleaseSvc.ListBatchSignatures(ctx, req.BatchId)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	out := make([]*pb.BatchSignatureInfo, len(signatures))
	for i, signature := range signatures {
		out[i] = batchSignatureToProto(signature)
	}
	return &pb.ListBatchSignaturesResponse{Signatures: out}, nil
}

func (h *batchReleaseGRPCHandler) SignBatch(ctx context.Context, req *pb.SignBatchRequest) (*pb.BatchSignatureInfo, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	params := domain.SignBatchParams{
		BatchID:             req.BatchId,
		Decision:            constants.BatchSignatureDecision(req.Decision),
		SignerAccountUserID: req.SignerAccountUserId,
		PIN:                 req.Pin,
		Note:                req.Note,
	}
	if req.Type != nil {
		signatureType := constants.BatchSignatureType(*req.Type)
		params.Type = &signatureType
	}
	if req.Capacity != nil {
		capacity := constants.SignatureCapacity(*req.Capacity)
		params.Capacity = &capacity
	}
	for _, response := range req.Checklist {
		params.Checklist = append(params.Checklist, domain.ChecklistResponse{
			Prompt: response.Prompt,
			Answer: constants.ChecklistAnswer(response.Answer),
		})
	}

	signature, apiErr := h.batchReleaseSvc.SignBatch(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return batchSignatureToProto(signature), nil
}

func (h *batchReleaseGRPCHandler) CheckInOperator(ctx context.Context, req *pb.CheckInOperatorRequest) (*pb.OperatorCheckInInfo, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	checkIn, apiErr := h.batchReleaseSvc.CheckInOperator(ctx, domain.CheckInOperatorParams{
		ScanningStationID: req.ScanningStationId,
		AccountUserID:     req.AccountUserId,
		PIN:               req.Pin,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return operatorCheckInToProto(checkIn), nil
}

func (h *batchReleaseGRPCHandler) CheckOutOperator(ctx context.Context, req *pb.CheckOutOperatorRequest) (*pb.OperatorCheckInInfo, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	checkIn, apiErr := h.batchReleaseSvc.CheckOutOperator(ctx, req.ScanningStationId, req.Id)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return operatorCheckInToProto(checkIn), nil
}

func (h *batchReleaseGRPCHandler) GetActiveOperatorCheckIn(ctx context.Context, req *pb.GetActiveOperatorCheckInRequest) (*pb.OperatorCheckInInfo, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	checkIn, apiErr := h.batchReleaseSvc.GetActiveOperatorCheckIn(ctx, req.ScanningStationId)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return operatorCheckInToProto(checkIn), nil
}
