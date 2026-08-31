package resourceloaders

import (
	"context"

	"github.com/open-mrp/api/services/api-gateway/internal/domain"
	grpcutil "github.com/open-mrp/api/services/api-gateway/internal/grpc"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	notifpb "github.com/open-mrp/api/shared/proto/notification"
	"github.com/open-mrp/api/shared/tracing"
	"google.golang.org/grpc"
)

var emailDomainLoaderTracer = tracing.GetTracer("api-gateway.resourceloaders.email_domain")

// LoadEmailDomains fetches email domains by ID for expansion as a sub-resource
// (an email inbox's ?include=email_domain). EmailBridgeService exposes no batch
// get, so domains are fetched one at a time; in practice each inbox carries a
// single domain id and the resolver dedups across roots before calling this loader.
func LoadEmailDomains(ctx context.Context, ids []string) (map[string]any, *apierror.APIError) {
	if len(ids) == 0 {
		return nil, nil
	}
	out := make(map[string]any, len(ids))
	for _, id := range ids {
		domainID := id
		resp, apiErr := grpcutil.CallRPC(ctx, emailDomainLoaderTracer, "loader.email_domains.get", domain.ServiceName,
			func(ctx context.Context, opts ...grpc.CallOption) (*notifpb.EmailDomainInfo, error) {
				return emailBridgeClient.GetEmailDomain(ctx, &notifpb.GetEmailDomainRequest{Id: domainID}, opts...)
			})
		if apiErr != nil {
			return nil, apiErr
		}
		if resp == nil {
			continue
		}
		out[domainID] = emailDomainBaseFromProto(resp)
	}
	return out, nil
}

// emailDomainBaseFromProto maps an email domain proto to its API resource. It
// mirrors emailDomainFromProto in the email-bridge endpoint package, duplicated
// here because that package would otherwise form an import cycle with resourceloaders.
func emailDomainBaseFromProto(d *notifpb.EmailDomainInfo) *apiresource.EmailDomain {
	return &apiresource.EmailDomain{
		ID:                d.Id,
		Object:            constants.ObjectTypeEmailDomain,
		Domain:            d.Domain,
		Status:            constants.EmailDomainStatus(d.Status),
		DkimTokens:        orEmptyStrSlice(d.DkimTokens),
		MailFromDomain:    d.MailFromDomain,
		MailFromMxRecord:  d.MailFromMxRecord,
		MailFromSpfRecord: d.MailFromSpfRecord,
		VerifiedAt:        grpcutil.TimestampToTimePtr(d.VerifiedAt),
		CreatedAt:         grpcutil.TimestampToTime(d.CreatedAt),
		UpdatedAt:         grpcutil.TimestampToTime(d.UpdatedAt),
	}
}

// LoadEmailInboxes exists only to satisfy the resourcekit registration for
// email inboxes, which are returned as endpoint roots and never fetched as an
// expandable sub-resource of another type. It must never be invoked.
func LoadEmailInboxes(_ context.Context, _ []string) (map[string]any, *apierror.APIError) {
	return nil, apierror.NewInvariantViolationError(
		"LoadEmailInboxes should not be called — email inboxes are not used as expandable sub-resources",
	)
}
