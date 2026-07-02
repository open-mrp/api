package contactep

import (
	"context"
	"fmt"

	"github.com/augno/api/services/api-gateway/internal/domain"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	"github.com/augno/api/services/api-gateway/internal/resourceloaders"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/api-gateway/pkg/resourcekit"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	pb "github.com/augno/api/shared/proto/notification"
	"github.com/augno/api/shared/tracing"
	"google.golang.org/grpc"
)

// ContactSvc backs the messaging directory endpoint via the notification-service ChatService gRPC
// client.
type ContactSvc interface {
	ListContacts(ctx context.Context, req *ListContactsRequest) (*apiresource.List[apiresource.Actor], *apierror.APIError)
}

type ContactSvcConfig struct {
	// ChatClient (required) is the notification-service ChatService gRPC client.
	ChatClient pb.ChatServiceClient
}

type contactSvcImpl struct {
	chatClient pb.ChatServiceClient
}

var contactSvcTracer = tracing.GetTracer("api-gateway.endpoints.messaging-contacts.service")

func (c *ContactSvcConfig) validate() error {
	if c.ChatClient == nil {
		return fmt.Errorf("messaging contact endpoint service: chat client is required")
	}
	return nil
}

func NewContactSvc(config *ContactSvcConfig) ContactSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}
	return &contactSvcImpl{chatClient: config.ChatClient}
}

func (s *contactSvcImpl) ListContacts(ctx context.Context, req *ListContactsRequest) (*apiresource.List[apiresource.Actor], *apierror.APIError) {
	var query string
	if req.Query != nil {
		query = *req.Query
	}
	resp, rpcErr := grpcutil.CallRPC(ctx, contactSvcTracer, "service.messaging_contacts.list", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListContactsResponse, error) {
			return s.chatClient.ListContacts(ctx, &pb.ListContactsRequest{Query: query}, opts...)
		})
	if rpcErr != nil {
		return nil, rpcErr
	}
	items := make([]apiresource.Actor, 0, len(resp.Contacts))
	for _, c := range resp.Contacts {
		if a := contactActorFromProto(c); a != nil {
			items = append(items, *a)
		}
	}
	s.hydrateContacts(ctx, items)
	return apiresource.NewList(items, apiresource.PageInfo{}), nil
}

// hydrateContacts fills the profile photo on user-actor contacts and stashes each contact's role FK
// in the request-scoped LoadMeta so `?include=role` resolves. Both are sourced from a single batch
// resolve of the contacts' account_user ids (the chat directory returns ids + names only).
// Best-effort: on failure the contacts keep their nil avatars and roles stay unresolvable rather
// than failing the request.
func (s *contactSvcImpl) hydrateContacts(ctx context.Context, contacts []apiresource.Actor) {
	userIDs := make([]string, 0, len(contacts))
	for i := range contacts {
		if contacts[i].Type == constants.ActorTypeUser && contacts[i].ID != "" {
			userIDs = append(userIDs, contacts[i].ID)
		}
	}
	if len(userIDs) == 0 {
		return
	}
	names, apiErr := resourceloaders.LoadAccountUserNames(ctx, userIDs)
	if apiErr != nil {
		return
	}
	meta := resourcekit.GetLoadMeta(ctx)
	for i := range contacts {
		n, ok := names[contacts[i].ID]
		if !ok {
			continue
		}
		if n.ImageURL != nil {
			contacts[i].AvatarURL = n.ImageURL
		}
		if n.RoleID != nil && *n.RoleID != "" {
			meta.Set(constants.ObjectTypeActor, contacts[i].ID, "role_id", *n.RoleID)
		}
	}
}

// contactActorFromProto maps a directory contact to an Actor. Internal contacts are `user` actors
// keyed by their account_user id; the customer-facing support contact is a shared `group` actor
// ("Customer Service") that deliberately exposes no individual staff member.
func contactActorFromProto(c *pb.ContactInfo) *apiresource.Actor {
	if c == nil {
		return nil
	}
	name := c.Name
	if c.AccountUserId != nil && *c.AccountUserId != "" {
		return apiresource.NewActor(*c.AccountUserId, constants.ActorTypeUser, &name, nil)
	}
	return apiresource.NewActor(supportContactActorID, constants.ActorTypeGroup, &name, nil)
}

// supportContactActorID is the stable identifier for the customer-facing support contact, a shared
// group identity rather than an individual account_user.
const supportContactActorID = "support"
