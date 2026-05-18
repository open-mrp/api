package productionrunep

import (
	"context"

	batchep "github.com/augno/api/services/api-gateway/endpoints/batches"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	pb "github.com/augno/api/shared/proto/core"
)

func ProductionRunSummaryPresenter(info *pb.ProductionRunSummaryInfo) apiresource.ProductionRunSummary {
	s := apiresource.ProductionRunSummary{
		ID:         info.Id,
		Object:     constants.ObjectTypeProductionRun,
		Number:     info.Number,
		BatchCount: info.BatchCount,
		CreatedAt:  grpcutil.TimestampToTime(info.CreatedAt),
		UpdatedAt:  grpcutil.TimestampToTime(info.UpdatedAt),
	}

	if info.ResponsibleUserId != "" {
		s.ResponsibleUser = &apiresource.AccountUser{
			ID:        info.ResponsibleUserId,
			Object:    constants.ObjectTypeAccountUser,
			Name:      info.ResponsibleUserName,
			Status:    constants.AccountUserStatus(info.GetResponsibleUserStatusCode()),
			CreatedAt: grpcutil.TimestampToTime(info.ResponsibleUserCreatedAt),
			UpdatedAt: grpcutil.TimestampToTime(info.ResponsibleUserUpdatedAt),
		}
	}

	s.StartedAt = grpcutil.TimestampToTimePtr(info.StartedAt)
	s.CompletedAt = grpcutil.TimestampToTimePtr(info.CompletedAt)

	return s
}

func ProductionRunDetailPresenter(info *pb.ProductionRunInfo) apiresource.ProductionRunDetail {
	d := apiresource.ProductionRunDetail{
		ID:         info.Id,
		Object:     constants.ObjectTypeProductionRun,
		Number:     info.Number,
		BatchCount: info.BatchCount,
		CreatedAt:  grpcutil.TimestampToTime(info.CreatedAt),
		UpdatedAt:  grpcutil.TimestampToTime(info.UpdatedAt),
	}

	if info.ResponsibleUserId != "" {
		d.ResponsibleUser = &apiresource.AccountUser{
			ID:        info.ResponsibleUserId,
			Object:    constants.ObjectTypeAccountUser,
			Name:      info.ResponsibleUserName,
			Status:    constants.AccountUserStatus(info.GetResponsibleUserStatusCode()),
			CreatedAt: grpcutil.TimestampToTime(info.ResponsibleUserCreatedAt),
			UpdatedAt: grpcutil.TimestampToTime(info.ResponsibleUserUpdatedAt),
		}
	}

	d.StartedAt = grpcutil.TimestampToTimePtr(info.StartedAt)
	d.CompletedAt = grpcutil.TimestampToTimePtr(info.CompletedAt)

	return d
}

func ProductionRunListPresenter(ctx context.Context, resp *pb.ListProductionRunsResponse) *apiresource.List[apiresource.ProductionRunSummary] {
	runs := make([]apiresource.ProductionRunSummary, len(resp.ProductionRuns))
	for i, pr := range resp.ProductionRuns {
		runs[i] = ProductionRunSummaryPresenter(pr)
	}

	return apiresource.NewList(runs, grpcutil.MapProtoPageInfo(ctx, resp.PageInfo))
}

func AddBatchesPresenter(resp *pb.AddBatchesToProductionRunResponse) *apiresource.List[apiresource.Batch] {
	batches := make([]apiresource.Batch, len(resp.Batches))
	for i, b := range resp.Batches {
		batches[i] = batchep.BaseBatchPresenter(b)
	}

	return apiresource.NewList(batches, apiresource.PageInfo{})
}

func ListBatchesPresenter(ctx context.Context, resp *pb.ListBatchesByProductionRunResponse) *apiresource.List[apiresource.Batch] {
	batches := make([]apiresource.Batch, len(resp.Batches))
	for i, b := range resp.Batches {
		batches[i] = batchep.BatchPresenter(b)
	}

	var pi apiresource.PageInfo
	if resp.PageInfo != nil {
		pi = grpcutil.MapProtoPageInfo(ctx, resp.PageInfo)
	}

	return apiresource.NewList(batches, pi)
}
