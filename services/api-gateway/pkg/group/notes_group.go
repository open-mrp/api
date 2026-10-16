package httpgroup

import (
	"fmt"

	noteep "github.com/open-mrp/api/services/api-gateway/endpoints/notes"
	grpcclient "github.com/open-mrp/api/services/api-gateway/grpc-client"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
)

type NotesEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type NotesEndpointGroupConfig struct {
	// CoreClient (required) is the core-service gRPC client.
	CoreClient *grpcclient.CoreServiceClient
}

func (c *NotesEndpointGroupConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("notes endpoint group: core client is required")
	}
	return nil
}

func (*NotesEndpointGroup) Materialize(config *NotesEndpointGroupConfig) *NotesEndpointGroup {
	if err := config.validate(); err != nil {
		panic(err)
	}

	svc := noteep.NewNoteSvc(&noteep.NoteSvcConfig{CoreClient: config.CoreClient.Note})

	inner := &apiendpoint.APIEndpointGroup{
		Title:        "Notes",
		Description:  "Notes on records and on your whole account, from people, API keys and agents. Agents read them before they work.",
		ResourceType: &apiresource.Note{},
	}

	inner.Endpoints = []apiendpoint.APIEndpointer{
		apiendpoint.From(&noteep.ListNotesEndpoint{}).WithService(inner, svc),
		apiendpoint.From(&noteep.RetrieveNoteEndpoint{}).WithService(inner, svc),
		apiendpoint.From(&noteep.CreateNoteEndpoint{}).WithService(inner, svc),
		apiendpoint.From(&noteep.UpdateNoteEndpoint{}).WithService(inner, svc),
		apiendpoint.From(&noteep.DeleteNoteEndpoint{}).WithService(inner, svc),
	}

	return &NotesEndpointGroup{inner}
}
