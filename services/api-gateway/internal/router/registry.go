package router

import (
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
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
			handler := endpointer.GetHandler()
			switch endpointer.GetMethod() {
			case http.MethodGet:
				router.Get(endpointer.GetRoute(), handler)
			case http.MethodPost:
				router.Post(endpointer.GetRoute(), handler)
			case http.MethodPut:
				router.Put(endpointer.GetRoute(), handler)
			case http.MethodDelete:
				router.Delete(endpointer.GetRoute(), handler)
			case http.MethodPatch:
				router.Patch(endpointer.GetRoute(), handler)
			}
		}
	}
}
