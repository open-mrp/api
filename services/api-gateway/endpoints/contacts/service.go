package contactep

import (
	"context"
	"fmt"

	"github.com/open-mrp/api/services/api-gateway/internal/domain"
	grpcutil "github.com/open-mrp/api/services/api-gateway/internal/grpc"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/api-gateway/pkg/resourcekit"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	pb "github.com/open-mrp/api/shared/proto/core"
	"github.com/open-mrp/api/shared/tracing"
	"google.golang.org/grpc"
)

type ContactSvc interface {
	FindContactByEmail(ctx context.Context, req *FindContactByEmailRequest) (*apiresource.List[apiresource.ContactMatch], *apierror.APIError)
}

type ContactSvcConfig struct {
	// CoreClient (required) is the core-service gRPC client.
	CoreClient pb.CoreServiceClient
}

type contactSvcImpl struct {
	coreClient pb.CoreServiceClient
}

var contactSvcTracer = tracing.GetTracer("api-gateway.endpoints.contacts.service")

func (c *ContactSvcConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("contact endpoint service: core client is required")
	}
	return nil
}

func NewContactSvc(config *ContactSvcConfig) ContactSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &contactSvcImpl{
		coreClient: config.CoreClient,
	}
}

func (s *contactSvcImpl) FindContactByEmail(ctx context.Context, req *FindContactByEmailRequest) (*apiresource.List[apiresource.ContactMatch], *apierror.APIError) {
	resp, apiErr := grpcutil.CallRPC(ctx, contactSvcTracer, "service.contacts.find_by_email", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.FindContactsByEmailResponse, error) {
			return s.coreClient.FindContactsByEmail(ctx, &pb.FindContactsByEmailRequest{Email: req.Email}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	if resp == nil {
		return apiresource.NewList[apiresource.ContactMatch](nil, apiresource.PageInfo{}), nil
	}

	relationshipFilter := make(map[constants.ContactRelationship]bool, len(req.Relationships))
	for _, r := range req.Relationships {
		relationshipFilter[r] = true
	}

	meta := resourcekit.GetLoadMeta(ctx)
	items := make([]apiresource.ContactMatch, 0, len(resp.Matches))
	for _, m := range resp.Matches {
		relationship := constants.ContactRelationship(m.Relationship)
		if len(relationshipFilter) > 0 && !relationshipFilter[relationship] {
			continue
		}

		cm := apiresource.ContactMatch{
			ID:           m.AccountUserId,
			Object:       constants.ObjectTypeContactMatch,
			Email:        m.Email,
			Relationship: relationship,
		}

		// account_user lives in a related account the account-scoped batch-get can't reach, so build
		// it here from the inline data and stash it for ?include=account_user. Stash the FK ids too so
		// account_user.user/role/department hydrate through the AccountUser definition's global loaders.
		au := &apiresource.AccountUser{
			ID:         m.AccountUserId,
			Object:     constants.ObjectTypeAccountUser,
			Status:     constants.AccountUserStatus(m.StatusCode),
			LastUsedAt: grpcutil.TimestampToTimePtr(m.LastUsedAt),
			CreatedAt:  grpcutil.TimestampToTime(m.CreatedAt),
			UpdatedAt:  grpcutil.TimestampToTime(m.UpdatedAt),
		}
		meta.Set(constants.ObjectTypeContactMatch, cm.ID, "account_user", au)
		meta.Set(constants.ObjectTypeAccountUser, au.ID, "user_id", m.UserId)
		if m.RoleId != nil {
			meta.Set(constants.ObjectTypeAccountUser, au.ID, "role_id", *m.RoleId)
		}
		if m.DepartmentId != nil {
			meta.Set(constants.ObjectTypeAccountUser, au.ID, "department_id", *m.DepartmentId)
		}

		// Stash the contact's account id (generic across customer/supplier/partner/self) so
		// LoadAccounts hydrates the account on ?include=account.
		meta.Set(constants.ObjectTypeContactMatch, cm.ID, "account_id", m.AccountId)

		items = append(items, cm)
	}

	return apiresource.NewList(items, apiresource.PageInfo{}), nil
}
