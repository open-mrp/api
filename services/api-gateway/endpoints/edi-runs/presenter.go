package edirunep

import (
	"context"

	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	pb "github.com/augno/api/shared/proto/core"
)

func EDIRunPresenter(e *pb.EDIRunProto) apiresource.EDIRun {
	if e == nil {
		return apiresource.EDIRun{}
	}

	return apiresource.EDIRun{
		ID:           e.Id,
		Object:       constants.ObjectTypeEDIRun,
		CompletedAt:  grpcutil.TimestampToTime(e.CompletedAt),
		HasSucceeded: e.HasSucceeded,
		CreatedAt:    grpcutil.TimestampToTime(e.CreatedAt),
		UpdatedAt:    grpcutil.TimestampToTime(e.UpdatedAt),
	}
}

func EDIRunListPresenter(ctx context.Context, resp *pb.ListEDIRunsResponse) *apiresource.List[apiresource.EDIRun] {
	if resp == nil {
		return apiresource.NewList[apiresource.EDIRun](nil, apiresource.PageInfo{})
	}

	runs := make([]apiresource.EDIRun, len(resp.EdiRuns))
	for i, e := range resp.EdiRuns {
		runs[i] = EDIRunPresenter(e)
	}

	return apiresource.NewList(runs, grpcutil.MapProtoPageInfo(ctx, resp.PageInfo))
}
