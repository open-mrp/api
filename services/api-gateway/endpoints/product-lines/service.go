package productlineep

import (
	"context"
	"fmt"

	itemep "github.com/open-mrp/api/services/api-gateway/endpoints/items"

	jobep "github.com/open-mrp/api/services/api-gateway/endpoints/jobs"
	"github.com/open-mrp/api/services/api-gateway/internal/domain"
	grpcutil "github.com/open-mrp/api/services/api-gateway/internal/grpc"
	"github.com/open-mrp/api/services/api-gateway/internal/resourceloaders"
	apirequest "github.com/open-mrp/api/services/api-gateway/pkg/request"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/fieldpb"
	pb "github.com/open-mrp/api/shared/proto/core"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
	"github.com/open-mrp/apikit/tracing"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

type ProductLineSvc interface {
	ListProductLines(ctx context.Context, req *ListProductLinesRequest) (*apiresource.List[apiresource.ProductLine], *apierror.APIError)
	ExportProductLines(ctx context.Context, req *ExportProductLinesRequest) (*apiresource.AsyncJob, *apierror.APIError)
	GetProductLine(ctx context.Context, req *RetrieveProductLineRequest) (*apiresource.ProductLine, *apierror.APIError)
	CreateProductLine(ctx context.Context, req *CreateProductLineRequest) (*apiresource.ProductLine, *apierror.APIError)
	UpdateProductLine(ctx context.Context, req *UpdateProductLineRequest) (*apiresource.ProductLine, *apierror.APIError)
	DeleteProductLine(ctx context.Context, req *DeleteProductLineRequest) (*apiresource.EmptyResource, *apierror.APIError)
	BulkUpsertProductLines(ctx context.Context, req *BulkUpsertProductLinesRequest) (*apiresource.AsyncJob, *apierror.APIError)
}

type ProductLineSvcConfig struct {
	// CoreClient (required) is the core-service gRPC client.
	CoreClient pb.CoreServiceClient
}

type productLineSvcImpl struct {
	coreClient pb.CoreServiceClient
}

var productLineSvcTracer = tracing.GetTracer("api-gateway.endpoints.product-lines.service")

func (c *ProductLineSvcConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("product line endpoint service: core client is required")
	}
	return nil
}

func NewProductLineSvc(config *ProductLineSvcConfig) ProductLineSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &productLineSvcImpl{
		coreClient: config.CoreClient,
	}
}

func (m *productLineSvcImpl) ExportProductLines(ctx context.Context, req *ExportProductLinesRequest) (*apiresource.AsyncJob, *apierror.APIError) {
	resp, apiErr := grpcutil.CallRPC(ctx, productLineSvcTracer, "service.product_lines.export", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ExportProductLinesResponse, error) {
			return m.coreClient.ExportProductLines(ctx, &pb.ExportProductLinesRequest{Query: req.Query}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	return jobep.JobFromProto(ctx, resp.GetJob()), nil
}

func (m *productLineSvcImpl) ListProductLines(ctx context.Context, req *ListProductLinesRequest) (*apiresource.List[apiresource.ProductLine], *apierror.APIError) {
	pbReq := &pb.ListProductLinesRequest{
		Cursor: req.Cursor,
		Limit:  req.Limit,
		Query:  req.Query,
	}
	for _, status := range req.Statuses {
		pbReq.Statuses = append(pbReq.Statuses, string(status))
	}

	resp, apiErr := grpcutil.CallRPC(ctx, productLineSvcTracer, "service.product-lines.list", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListProductLinesResponse, error) {
			return m.coreClient.ListProductLines(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	ids := make([]string, len(resp.ProductLines))
	for i, pl := range resp.ProductLines {
		ids[i] = pl.Id
	}
	loaded, apiErr := resourceloaders.LoadProductLines(ctx, ids)
	if apiErr != nil {
		return nil, apiErr
	}
	items := make([]apiresource.ProductLine, 0, len(ids))
	for _, id := range ids {
		if v, ok := loaded[id]; ok {
			items = append(items, *(v.(*apiresource.ProductLine)))
		}
	}
	return apiresource.NewList(items, grpcutil.MapProtoPageInfo(ctx, resp.PageInfo)), nil
}

func (m *productLineSvcImpl) GetProductLine(ctx context.Context, req *RetrieveProductLineRequest) (*apiresource.ProductLine, *apierror.APIError) {
	return loadProductLineByID(ctx, req.ProductLineID)
}

func (m *productLineSvcImpl) CreateProductLine(ctx context.Context, req *CreateProductLineRequest) (*apiresource.ProductLine, *apierror.APIError) {
	if apiErr := itemep.RejectPreview9Field(ctx, req.UnitGroupID.IsSet(), "unit_group_id", itemep.UnitGroupsRetired); apiErr != nil {
		return nil, apiErr
	}
	pbReq := &pb.CreateProductLineRequest{
		Name:             req.Name,
		Description:      req.Description.Ptr(),
		UnitGroupId:      req.UnitGroupID.Ptr(),
		CommissionPolicy: string(req.CommissionPolicy),
		FreightPolicy:    string(req.FreightPolicy),
		Supply:           req.Supply.Ptr().StringPtr(),
		Status:           req.Status.Ptr().StringPtr(),
	}

	resp, apiErr := grpcutil.CallRPC(ctx, productLineSvcTracer, "service.product-lines.create", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.CreateProductLineResponse, error) {
			return m.coreClient.CreateProductLine(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return loadProductLineByID(ctx, resp.ProductLine.Id)
}

func (m *productLineSvcImpl) UpdateProductLine(ctx context.Context, req *UpdateProductLineRequest) (*apiresource.ProductLine, *apierror.APIError) {
	if apiErr := itemep.RejectPreview9Field(ctx, req.UnitGroupID.IsSet(), "unit_group_id", itemep.UnitGroupsRetired); apiErr != nil {
		return nil, apiErr
	}
	pbReq := &pb.UpdateProductLineRequest{
		Id:               req.ProductLineID,
		Name:             req.Name.Ptr(),
		Description:      fieldpb.StringClearableToProto(req.Description),
		UnitGroupId:      req.UnitGroupID.Ptr(),
		CommissionPolicy: req.CommissionPolicy.Ptr().StringPtr(),
		FreightPolicy:    req.FreightPolicy.Ptr().StringPtr(),
		Supply:           fieldpb.EnumClearableToProto(req.Supply),
		Status:           req.Status.Ptr().StringPtr(),
	}

	resp, apiErr := grpcutil.CallRPC(ctx, productLineSvcTracer, "service.product-lines.update", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.UpdateProductLineResponse, error) {
			return m.coreClient.UpdateProductLine(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return loadProductLineByID(ctx, resp.ProductLine.Id)
}

func (m *productLineSvcImpl) DeleteProductLine(ctx context.Context, req *DeleteProductLineRequest) (*apiresource.EmptyResource, *apierror.APIError) {
	pbReq := &pb.DeleteProductLineRequest{
		Id: req.ProductLineID,
	}

	_, apiErr := grpcutil.CallRPC(ctx, productLineSvcTracer, "service.product-lines.delete", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*emptypb.Empty, error) {
			return m.coreClient.DeleteProductLine(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return &apiresource.EmptyResource{}, nil
}

func (m *productLineSvcImpl) BulkUpsertProductLines(ctx context.Context, req *BulkUpsertProductLinesRequest) (*apiresource.AsyncJob, *apierror.APIError) {
	inputs := make([]*pb.BulkUpsertProductLineInput, len(req.ProductLines))
	for i, pl := range req.ProductLines {
		inputs[i] = &pb.BulkUpsertProductLineInput{
			Name:             pl.Name,
			CommissionPolicy: string(pl.CommissionPolicy),
			FreightPolicy:    string(pl.FreightPolicy),
		}
		if ug, ok := pl.UnitGroup.Value(); ok {
			if apiErr := itemep.RejectPreview9Field(ctx, true, fmt.Sprintf("product_lines[%d].unit_group", i), itemep.UnitGroupsRetired); apiErr != nil {
				return nil, apiErr
			}
			inputs[i].UnitGroup = apirequest.ObjectIdentifierToProto(ug)
		}
	}

	pbReq := &pb.BulkUpsertProductLinesRequest{
		ProductLines: inputs,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, productLineSvcTracer, "service.product-lines.bulk-upsert", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.BulkUpsertProductLinesResponse, error) {
			return m.coreClient.BulkUpsertProductLines(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return jobep.JobFromProto(ctx, resp.GetJob()), nil
}

// wraps the single-ID load pattern used after every mutation and for the retrieve endpoint.
func loadProductLineByID(ctx context.Context, id string) (*apiresource.ProductLine, *apierror.APIError) {
	loaded, apiErr := resourceloaders.LoadProductLines(ctx, []string{id})
	if apiErr != nil {
		return nil, apiErr
	}
	v, ok := loaded[id]
	if !ok {
		return nil, apierror.NewNotFoundError("Product line not found.")
	}
	return v.(*apiresource.ProductLine), nil
}

const (
	productLinesRoute           = "/v1/catalog/product-lines"
	productLineRoute            = "/v1/catalog/product-lines/{id}"
	productLinesBulkUpsertRoute = "/v1/catalog/product-lines/actions/bulk-upsert"
)

func productLineIncludes() *apiendpoint.IncludeConfig {
	return apiendpoint.IncludesFor(apiendpoint.IncludesParams{
		ObjectType: constants.ObjectTypeProductLine,
		Fields:     []string{"owner", "owner.account", "unit_group"},
	})
}
