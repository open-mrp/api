package grpc

import (
	"context"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/shared/contracts"
	pb "github.com/augno/api/shared/proto/core"
)

func catalogProductLineToProto(pl *domain.CatalogProductLine) *pb.CatalogProductLineProto {
	if pl == nil {
		return nil
	}
	return &pb.CatalogProductLineProto{
		Id:   pl.ID,
		Name: pl.Name,
	}
}

func catalogCategoryToProto(cat *domain.CatalogCategory) *pb.CatalogCategoryProto {
	if cat == nil {
		return nil
	}

	products := make([]*pb.CatalogProductProto, len(cat.Products))
	for i, p := range cat.Products {
		attrs := make([]*pb.CatalogAttributeProto, len(p.Attributes))
		for j, a := range p.Attributes {
			attrs[j] = &pb.CatalogAttributeProto{
				Id:           a.ID,
				Name:         a.Name,
				PropertyId:   a.PropertyID,
				PropertyName: a.PropertyName,
			}
		}
		products[i] = &pb.CatalogProductProto{
			ItemId:      p.ItemID,
			Sku:         p.SKU,
			Description: p.Description,
			Attributes:  attrs,
		}
	}

	properties := make([]*pb.CatalogPropertyProto, len(cat.Properties))
	for i, pr := range cat.Properties {
		properties[i] = &pb.CatalogPropertyProto{
			Id:   pr.ID,
			Name: pr.Name,
		}
	}

	return &pb.CatalogCategoryProto{
		Id:         cat.ID,
		Name:       cat.Name,
		Products:   products,
		Properties: properties,
	}
}

func (h *gRPCHandler) ListCatalogProductLines(ctx context.Context, req *pb.ListCatalogProductLinesRequest) (*pb.ListCatalogProductLinesResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	result, apiErr := h.catalogSvc.ListCatalogProductLines(ctx, domain.ListCatalogProductLinesParams{
		Cursor: req.Cursor,
		Limit:  req.Limit,
		Query:  req.Query,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	pbProductLines := make([]*pb.CatalogProductLineProto, len(result.ProductLines))
	for i, pl := range result.ProductLines {
		pbProductLines[i] = catalogProductLineToProto(pl)
	}

	return &pb.ListCatalogProductLinesResponse{
		ProductLines: pbProductLines,
		PageInfo: &pb.PageInfo{
			NextCursor:  result.PageInfo.NextCursor,
			PrevCursor:  result.PageInfo.PrevCursor,
			HasNextPage: result.PageInfo.HasNextPage,
			HasPrevPage: result.PageInfo.HasPrevPage,
		},
	}, nil
}

func (h *gRPCHandler) ListCatalogProducts(ctx context.Context, req *pb.ListCatalogProductsRequest) (*pb.ListCatalogProductsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	result, apiErr := h.catalogSvc.ListCatalogProducts(ctx, domain.ListCatalogProductsParams{
		ProductLineID: req.ProductLineId,
		Cursor:        req.Cursor,
		Limit:         req.Limit,
		Query:         req.Query,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	pbCategories := make([]*pb.CatalogCategoryProto, len(result.Categories))
	for i, cat := range result.Categories {
		pbCategories[i] = catalogCategoryToProto(cat)
	}

	return &pb.ListCatalogProductsResponse{
		Categories: pbCategories,
		PageInfo: &pb.PageInfo{
			NextCursor:  result.PageInfo.NextCursor,
			PrevCursor:  result.PageInfo.PrevCursor,
			HasNextPage: result.PageInfo.HasNextPage,
			HasPrevPage: result.PageInfo.HasPrevPage,
		},
	}, nil
}
