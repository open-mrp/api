package router

import (
	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
)

type Registry struct {
	groups []apiendpoint.APIEndpointGroup
}

func NewRegistry() *Registry {
	return &Registry{
		groups: make([]apiendpoint.APIEndpointGroup, 0),
	}
}

func (r *Registry) RegisterGroup(group *apiendpoint.APIEndpointGroup) {
	r.groups = append(r.groups, *group)
}

func (r *Registry) RegisterEndpoints(router *router) {
	for _, group := range r.groups {
		for _, endpointer := range group.Endpoints {
			router.HandleEndpoint(endpointer.GetMethod(), endpointer.GetRoute(), endpointer.GetHandler(), endpointer.IsPublic())
		}
	}
}
