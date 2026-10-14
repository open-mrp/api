package batchreleaseep

import (
	"context"
	"fmt"

	"github.com/open-mrp/api/services/api-gateway/internal/domain"
	grpcutil "github.com/open-mrp/api/services/api-gateway/internal/grpc"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	pb "github.com/open-mrp/api/shared/proto/core"
	"github.com/open-mrp/api/shared/tracing"
	"google.golang.org/grpc"
)

const samplePin = "2580"

type BatchReleaseSvc interface {
	GetReleasePolicy(ctx context.Context, req *RetrieveReleasePolicyRequest) (*apiresource.ReleasePolicy, *apierror.APIError)
	UpdateReleasePolicy(ctx context.Context, req *UpdateReleasePolicyRequest) (*apiresource.ReleasePolicy, *apierror.APIError)
	ListEligibleSigners(ctx context.Context, req *ListEligibleSignersRequest) (*apiresource.List[apiresource.EligibleSigner], *apierror.APIError)
	ListBatchSignatures(ctx context.Context, req *ListBatchSignaturesRequest) (*apiresource.List[apiresource.BatchSignature], *apierror.APIError)
	SignBatch(ctx context.Context, req *SignBatchRequest) (*apiresource.BatchSignature, *apierror.APIError)
	CheckInOperator(ctx context.Context, req *CheckInOperatorRequest) (*apiresource.OperatorCheckIn, *apierror.APIError)
	CheckOutOperator(ctx context.Context, req *CheckOutOperatorRequest) (*apiresource.OperatorCheckIn, *apierror.APIError)
	GetActiveOperatorCheckIn(ctx context.Context, req *RetrieveActiveOperatorCheckInRequest) (*apiresource.OperatorCheckIn, *apierror.APIError)
}

type BatchReleaseSvcConfig struct {
	// CoreClient (required) is the core-service batch-release gRPC client.
	CoreClient pb.CoreBatchReleaseServiceClient
}

func (c *BatchReleaseSvcConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("batch release endpoint service: core client is required")
	}
	return nil
}

type batchReleaseSvcImpl struct {
	coreClient pb.CoreBatchReleaseServiceClient
}

var batchReleaseEpSvcTracer = tracing.GetTracer("api-gateway.endpoints.batch-releases.service")

func NewBatchReleaseSvc(config *BatchReleaseSvcConfig) BatchReleaseSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}
	return &batchReleaseSvcImpl{coreClient: config.CoreClient}
}

func (s *batchReleaseSvcImpl) GetReleasePolicy(ctx context.Context, req *RetrieveReleasePolicyRequest) (*apiresource.ReleasePolicy, *apierror.APIError) {
	resp, apiErr := grpcutil.CallRPC(ctx, batchReleaseEpSvcTracer, "service.batch_releases.get_release_policy", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ReleasePolicyInfo, error) {
			return s.coreClient.GetReleasePolicy(ctx, &pb.GetReleasePolicyRequest{DepartmentId: req.DepartmentID}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	return releasePolicyPresenter(resp), nil
}

func (s *batchReleaseSvcImpl) UpdateReleasePolicy(ctx context.Context, req *UpdateReleasePolicyRequest) (*apiresource.ReleasePolicy, *apierror.APIError) {
	enforcement := constants.ReleaseEnforcementBlock
	if v, ok := req.Enforcement.Value(); ok {
		enforcement = v
	}
	operatorSigning := constants.ReleaseOperatorSigningAllowed
	if v, ok := req.OperatorSigning.Value(); ok {
		operatorSigning = v
	}
	resp, apiErr := grpcutil.CallRPC(ctx, batchReleaseEpSvcTracer, "service.batch_releases.update_release_policy", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ReleasePolicyInfo, error) {
			return s.coreClient.UpdateReleasePolicy(ctx, &pb.UpdateReleasePolicyRequest{
				DepartmentId:    req.DepartmentID,
				Requirement:     string(req.Requirement),
				Enforcement:     string(enforcement),
				Rationale:       req.Rationale.Ptr(),
				Checklist:       req.Checklist,
				OperatorSigning: string(operatorSigning),
			}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	return releasePolicyPresenter(resp), nil
}

func (s *batchReleaseSvcImpl) ListEligibleSigners(ctx context.Context, req *ListEligibleSignersRequest) (*apiresource.List[apiresource.EligibleSigner], *apierror.APIError) {
	resp, apiErr := grpcutil.CallRPC(ctx, batchReleaseEpSvcTracer, "service.batch_releases.list_eligible_signers", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListEligibleSignersResponse, error) {
			return s.coreClient.ListEligibleSigners(ctx, &pb.ListEligibleSignersRequest{BatchId: req.BatchID}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	signers := make([]apiresource.EligibleSigner, len(resp.Signers))
	for i, signer := range resp.Signers {
		capacities := make([]constants.SignatureCapacity, len(signer.Capacities))
		for j, capacity := range signer.Capacities {
			capacities[j] = constants.SignatureCapacity(capacity)
		}
		signers[i] = apiresource.EligibleSigner{
			Object:      constants.ObjectTypeEligibleSigner,
			AccountUser: apiresource.NewEntity(signer.AccountUserId, constants.ObjectTypeAccountUser, &signer.Name, nil),
			Role:        signer.RoleName,
			Capacities:  capacities,
		}
	}
	return apiresource.NewList(signers, apiresource.PageInfo{}), nil
}

func (s *batchReleaseSvcImpl) ListBatchSignatures(ctx context.Context, req *ListBatchSignaturesRequest) (*apiresource.List[apiresource.BatchSignature], *apierror.APIError) {
	resp, apiErr := grpcutil.CallRPC(ctx, batchReleaseEpSvcTracer, "service.batch_releases.list_batch_signatures", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListBatchSignaturesResponse, error) {
			return s.coreClient.ListBatchSignatures(ctx, &pb.ListBatchSignaturesRequest{BatchId: req.BatchID}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	signatures := make([]apiresource.BatchSignature, len(resp.Signatures))
	for i, signature := range resp.Signatures {
		signatures[i] = *batchSignaturePresenter(signature)
	}
	return apiresource.NewList(signatures, apiresource.PageInfo{}), nil
}

func (s *batchReleaseSvcImpl) SignBatch(ctx context.Context, req *SignBatchRequest) (*apiresource.BatchSignature, *apierror.APIError) {
	in := &pb.SignBatchRequest{
		BatchId:             req.BatchID,
		Decision:            string(req.Decision),
		SignerAccountUserId: req.SignerAccountUserID,
		Pin:                 req.Pin,
		Note:                req.Note.Ptr(),
	}
	if v, ok := req.Type.Value(); ok {
		in.Type = new(string(v))
	}
	if v, ok := req.Capacity.Value(); ok {
		in.Capacity = new(string(v))
	}
	for _, answer := range req.Checklist {
		in.Checklist = append(in.Checklist, &pb.ChecklistResponseInfo{Prompt: answer.Prompt, Answer: string(answer.Answer)})
	}
	resp, apiErr := grpcutil.CallRPC(ctx, batchReleaseEpSvcTracer, "service.batch_releases.sign_batch", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.BatchSignatureInfo, error) {
			return s.coreClient.SignBatch(ctx, in, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	return batchSignaturePresenter(resp), nil
}

func (s *batchReleaseSvcImpl) CheckInOperator(ctx context.Context, req *CheckInOperatorRequest) (*apiresource.OperatorCheckIn, *apierror.APIError) {
	resp, apiErr := grpcutil.CallRPC(ctx, batchReleaseEpSvcTracer, "service.batch_releases.check_in_operator", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.OperatorCheckInInfo, error) {
			return s.coreClient.CheckInOperator(ctx, &pb.CheckInOperatorRequest{
				ScanningStationId: req.ScanningStationID,
				AccountUserId:     req.AccountUserID,
				Pin:               req.Pin,
			}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	return operatorCheckInPresenter(resp), nil
}

func (s *batchReleaseSvcImpl) CheckOutOperator(ctx context.Context, req *CheckOutOperatorRequest) (*apiresource.OperatorCheckIn, *apierror.APIError) {
	resp, apiErr := grpcutil.CallRPC(ctx, batchReleaseEpSvcTracer, "service.batch_releases.check_out_operator", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.OperatorCheckInInfo, error) {
			return s.coreClient.CheckOutOperator(ctx, &pb.CheckOutOperatorRequest{
				ScanningStationId: req.ScanningStationID,
				Id:                req.CheckInID,
			}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	return operatorCheckInPresenter(resp), nil
}

func (s *batchReleaseSvcImpl) GetActiveOperatorCheckIn(ctx context.Context, req *RetrieveActiveOperatorCheckInRequest) (*apiresource.OperatorCheckIn, *apierror.APIError) {
	resp, apiErr := grpcutil.CallRPC(ctx, batchReleaseEpSvcTracer, "service.batch_releases.get_active_operator_check_in", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.OperatorCheckInInfo, error) {
			return s.coreClient.GetActiveOperatorCheckIn(ctx, &pb.GetActiveOperatorCheckInRequest{ScanningStationId: req.ScanningStationID}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	return operatorCheckInPresenter(resp), nil
}

func releasePolicyPresenter(p *pb.ReleasePolicyInfo) *apiresource.ReleasePolicy {
	checklist := p.Checklist
	if checklist == nil {
		checklist = []string{}
	}
	return &apiresource.ReleasePolicy{
		Object:          constants.ObjectTypeReleasePolicy,
		Department:      apiresource.NewEntity(p.DepartmentId, constants.ObjectTypeDepartment, &p.DepartmentName, nil),
		Requirement:     constants.ReleaseRequirement(p.Requirement),
		Enforcement:     constants.ReleaseEnforcement(p.Enforcement),
		OperatorSigning: constants.ReleaseOperatorSigning(p.OperatorSigning),
		Rationale:       p.Rationale,
		Checklist:       apiresource.NewList(checklist, apiresource.PageInfo{}),
		UpdatedAt:       grpcutil.TimestampToTime(p.UpdatedAt),
	}
}

func batchSignaturePresenter(s *pb.BatchSignatureInfo) *apiresource.BatchSignature {
	checklist := make([]apiresource.ChecklistResponse, len(s.Checklist))
	for i, response := range s.Checklist {
		checklist[i] = apiresource.ChecklistResponse{
			Object: constants.ObjectTypeChecklistResponse,
			Prompt: response.Prompt,
			Answer: constants.ChecklistAnswer(response.Answer),
		}
	}
	signature := &apiresource.BatchSignature{
		ID:             s.Id,
		Object:         constants.ObjectTypeBatchSignature,
		Batch:          &apiresource.BatchReference{ID: s.BatchId, Object: constants.ObjectTypeBatch},
		ProductionStep: apiresource.NewEntity(s.ProductionStepId, constants.ObjectTypeProductionStep, nil, nil),
		Type:           constants.BatchSignatureType(s.Type),
		Decision:       constants.BatchSignatureDecision(s.Decision),
		Capacity:       constants.SignatureCapacity(s.Capacity),
		Signer:         apiresource.NewEntity(s.SignerAccountUserId, constants.ObjectTypeAccountUser, &s.SignerName, nil),
		SignerRole:     s.SignerRoleName,
		Checklist:      apiresource.NewList(checklist, apiresource.PageInfo{}),
		Note:           s.Note,
		Actor:          apiresource.NewActor(s.ActorId, constants.ActorType(s.ActorType), nil, nil),
		SignedAt:       grpcutil.TimestampToTime(s.SignedAt),
	}
	if s.LotId != nil {
		signature.Lot = apiresource.NewEntity(*s.LotId, constants.ObjectTypeLot, s.LotNumber, nil)
	}
	if s.ProductionRunId != nil {
		signature.ProductionRun = apiresource.NewEntity(*s.ProductionRunId, constants.ObjectTypeProductionRun, nil, nil)
	}
	return signature
}

func operatorCheckInPresenter(c *pb.OperatorCheckInInfo) *apiresource.OperatorCheckIn {
	return &apiresource.OperatorCheckIn{
		ID:              c.Id,
		Object:          constants.ObjectTypeOperatorCheckIn,
		ScanningStation: apiresource.NewEntity(c.ScanningStationId, constants.ObjectTypeScanningStation, nil, nil),
		Operator:        apiresource.NewEntity(c.AccountUserId, constants.ObjectTypeAccountUser, &c.OperatorName, nil),
		Status:          constants.OperatorCheckInStatus(c.Status),
		CheckedInAt:     grpcutil.TimestampToTime(c.CheckedInAt),
		ExpiresAt:       grpcutil.TimestampToTime(c.ExpiresAt),
		CheckedOutAt:    grpcutil.TimestampToTimePtr(c.CheckedOutAt),
	}
}
