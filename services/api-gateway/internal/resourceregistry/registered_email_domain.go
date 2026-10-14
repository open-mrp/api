package resourceregistry

import (
	"context"

	"github.com/open-mrp/api/services/api-gateway/internal/resourceloaders"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/apikit/include"
)

func init() {
	// Email domains have no expandable sub-resources of their own; they are only
	// registered so they can be loaded as an email inbox's ?include=email_domain.
	include.Register(&include.Definition{
		ObjectType: constants.ObjectTypeEmailDomain,
		Load:       resourceloaders.LoadEmailDomains,
	})
	include.Register(&include.Definition{
		ObjectType: constants.ObjectTypeEmailSender,
		Load:       resourceloaders.LoadEmailSenders,
		Subs: []include.SubField{
			{
				Key:         "email_domain",
				Target:      constants.ObjectTypeEmailDomain,
				Cardinality: include.CardinalityOnePtr,
				ExtractIDs:  extractEmailDomainIDFromSender,
				Populate:    populateEmailDomainOnSender,
			},
		},
	})
}

func extractEmailDomainIDFromSender(ctx context.Context, parent any) []string {
	s := parent.(*apiresource.EmailSender)
	id, _ := include.GetLoadMeta(ctx).GetString(constants.ObjectTypeEmailSender, s.ID, "email_domain_id")
	if id == "" {
		return nil
	}
	return []string{id}
}

func populateEmailDomainOnSender(ctx context.Context, parent any, loaded map[string]any) {
	s := parent.(*apiresource.EmailSender)
	id, _ := include.GetLoadMeta(ctx).GetString(constants.ObjectTypeEmailSender, s.ID, "email_domain_id")
	if id == "" {
		return
	}
	if v, ok := loaded[id]; ok {
		s.EmailDomain = v.(*apiresource.EmailDomain)
	}
}
