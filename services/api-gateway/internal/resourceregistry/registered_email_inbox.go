package resourceregistry

import (
	"context"

	"github.com/open-mrp/api/services/api-gateway/internal/resourceloaders"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/api-gateway/pkg/resourcekit"
	"github.com/open-mrp/api/shared/constants"
)

func init() {
	// The base mapper (emailInboxFromProto) leaves email_domain and agent_config nil and stashes
	// their FK ids in LoadMeta. When the caller requests the include, the resolver reads that id,
	// batch-loads the full resource via its own loader, and attaches it in place.
	resourcekit.Register(&resourcekit.Definition{
		ObjectType: constants.ObjectTypeEmailInbox,
		Load:       resourceloaders.LoadEmailInboxes,
		Subs: []resourcekit.SubField{
			{
				Key:         "email_domain",
				Target:      constants.ObjectTypeEmailDomain,
				Cardinality: resourcekit.CardinalityOnePtr,
				ExtractIDs:  extractEmailDomainIDFromInbox,
				Populate:    populateEmailDomainOnInbox,
			},
			{
				Key:         "agent_config",
				Target:      constants.ObjectTypeAgentDefinition,
				Cardinality: resourcekit.CardinalityOnePtr,
				ExtractIDs:  extractAgentConfigIDFromInbox,
				Populate:    populateAgentConfigOnInbox,
			},
		},
	})
}

func extractEmailDomainIDFromInbox(ctx context.Context, parent any) []string {
	i := parent.(*apiresource.EmailInbox)
	id, _ := resourcekit.GetLoadMeta(ctx).GetString(constants.ObjectTypeEmailInbox, i.ID, "email_domain_id")
	if id == "" {
		return nil
	}
	return []string{id}
}

func populateEmailDomainOnInbox(ctx context.Context, parent any, loaded map[string]any) {
	i := parent.(*apiresource.EmailInbox)
	id, _ := resourcekit.GetLoadMeta(ctx).GetString(constants.ObjectTypeEmailInbox, i.ID, "email_domain_id")
	if id == "" {
		return
	}
	if v, ok := loaded[id]; ok {
		i.EmailDomain = v.(*apiresource.EmailDomain)
	}
}

func extractAgentConfigIDFromInbox(ctx context.Context, parent any) []string {
	i := parent.(*apiresource.EmailInbox)
	id, _ := resourcekit.GetLoadMeta(ctx).GetString(constants.ObjectTypeEmailInbox, i.ID, "agent_config_id")
	if id == "" {
		return nil
	}
	return []string{id}
}

func populateAgentConfigOnInbox(ctx context.Context, parent any, loaded map[string]any) {
	i := parent.(*apiresource.EmailInbox)
	id, _ := resourcekit.GetLoadMeta(ctx).GetString(constants.ObjectTypeEmailInbox, i.ID, "agent_config_id")
	if id == "" {
		return
	}
	if v, ok := loaded[id]; ok {
		i.AgentConfig = v.(*apiresource.AgentDefinition)
	}
}
