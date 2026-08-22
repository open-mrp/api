package resourceloaders

import (
	"context"

	"github.com/open-mrp/api/services/api-gateway/internal/domain"
	grpcutil "github.com/open-mrp/api/services/api-gateway/internal/grpc"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	pb "github.com/open-mrp/api/shared/proto/core"
	"github.com/open-mrp/api/shared/tracing"
	"google.golang.org/grpc"
)

var portalDomainLoaderTracer = tracing.GetTracer("api-gateway.resourceloaders.portal_domain")

// LoadPortalDomains fetches portal domains by ID for expansion as a sub-resource.
func LoadPortalDomains(ctx context.Context, ids []string) (map[string]any, *apierror.APIError) {
	if len(ids) == 0 {
		return nil, nil
	}

	resp, apiErr := grpcutil.CallRPC(ctx, portalDomainLoaderTracer, "loader.portal_domains.batch_get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.BatchGetPortalDomainsByIDsResponse, error) {
			return portalDomainClient.BatchGetPortalDomainsByIDs(ctx, &pb.BatchGetPortalDomainsByIDsRequest{Ids: ids}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	out := make(map[string]any, len(resp.PortalDomains))
	for _, d := range resp.PortalDomains {
		out[d.Id] = portalDomainBaseFromProto(d)
	}
	return out, nil
}

// portalDomainBaseFromProto maps a portal domain proto to its API resource. It mirrors portalDomainFromProto in the portal-domains endpoint package, duplicated here because that package would otherwise form an import cycle with resourceloaders.
func portalDomainBaseFromProto(d *pb.PortalDomainInfo) *apiresource.PortalDomain {
	records := make([]apiresource.DNSRecord, 0, len(d.DnsRecords))
	for _, r := range d.DnsRecords {
		records = append(records, apiresource.DNSRecord{
			Object: constants.ObjectTypeDNSRecord,
			Type:   constants.DNSRecordType(r.Type),
			Name:   r.Name,
			Value:  r.Value,
			Reason: constants.DNSRecordReason(r.Reason),
		})
	}

	return &apiresource.PortalDomain{
		ID:         d.Id,
		Object:     constants.ObjectTypePortalDomain,
		Domain:     d.Domain,
		Status:     constants.PortalDomainStatus(d.Status),
		DNSRecords: apiresource.NewList(records, apiresource.PageInfo{}),
		VerifiedAt: grpcutil.TimestampToTimePtr(d.VerifiedAt),
		CreatedAt:  grpcutil.TimestampToTime(d.CreatedAt),
		UpdatedAt:  grpcutil.TimestampToTime(d.UpdatedAt),
	}
}
