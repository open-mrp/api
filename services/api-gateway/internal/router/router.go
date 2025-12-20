package router

import (
	"context"
	"fmt"
	"net/http"

	apicontext "github.com/augno/api/services/api-gateway/internal/context"
	httptransport "github.com/augno/api/services/api-gateway/internal/http"
	"github.com/augno/api/shared/contracts"
)

type router struct {
	*http.ServeMux
	routes      []Route
	handlers    map[string]map[string]http.HandlerFunc
	middlewares []func(http.HandlerFunc) http.HandlerFunc
}

func NewRouter() *router {
	router := &router{
		ServeMux:    http.NewServeMux(),
		routes:      make([]Route, 0),
		handlers:    make(map[string]map[string]http.HandlerFunc),
		middlewares: make([]func(http.HandlerFunc) http.HandlerFunc, 0),
	}

	router.ServeMux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/" {
			if rootHandlers, exists := router.handlers["/"]; exists {
				if methodHandler, methodExists := rootHandlers[req.Method]; methodExists {
					finalHandler := methodHandler
					for i := len(router.middlewares) - 1; i >= 0; i-- {
						finalHandler = router.middlewares[i](finalHandler)
					}
					finalHandler(w, req)
					return
				} else {
					httptransport.RespondWithAPIError(req.Context(), w, contracts.NewMethodNotAllowedError(fmt.Sprintf("Method %s not allowed for path %s.", req.Method, req.URL.Path)))
					return
				}
			}
		}

		for _, route := range router.routes {
			pathMatches := false
			var params map[string]string

			if route.PathPattern != nil {
				params = extractPathParams(route.PathPattern, route.PathParams, req.URL.Path)
				pathMatches = params != nil
			} else {
				pathMatches = route.Path == req.URL.Path
			}

			if pathMatches {
				if route.Method == req.Method || req.Method == http.MethodOptions {
					if params != nil {
						ctx := context.WithValue(req.Context(), apicontext.PathParamsKey, params)
						ctx = apicontext.WithRoutePattern(ctx, route.Path)
						req = req.WithContext(ctx)
					} else {
						ctx := apicontext.WithRoutePattern(req.Context(), route.Path)
						req = req.WithContext(ctx)
					}

					finalHandler := route.Handler
					for i := len(router.middlewares) - 1; i >= 0; i-- {
						finalHandler = router.middlewares[i](finalHandler)
					}
					finalHandler(w, req)
					return
				}
			}
		}

		httptransport.RespondWithAPIError(req.Context(), w, contracts.NewResourceNotFoundError(fmt.Sprintf("The requested endpoint %s %s was not found.", req.Method, req.URL.Path)))
	})

	return router
}

func (r *router) AddMiddleware(middleware func(http.HandlerFunc) http.HandlerFunc) {
	r.middlewares = append(r.middlewares, middleware)
}

func (r *router) handle(method, path string, handler http.HandlerFunc) {
	pattern, paramNames := compilePathPattern(path)

	route := Route{
		Method:      method,
		Path:        path,
		Handler:     handler,
		PathPattern: pattern,
		PathParams:  paramNames,
	}

	r.routes = append(r.routes, route)

	if pattern == nil {
		if r.handlers[path] == nil {
			r.handlers[path] = make(map[string]http.HandlerFunc)

			r.ServeMux.HandleFunc(path, func(w http.ResponseWriter, req *http.Request) {
				ctx := apicontext.WithRoutePattern(req.Context(), path)
				req = req.WithContext(ctx)

				methodHandlers := r.handlers[path]
				handler, exists := methodHandlers[req.Method]
				if !exists && req.Method == http.MethodOptions {
					for _, h := range methodHandlers {
						handler = h
						exists = true
						break
					}
				}

				if exists {
					finalHandler := handler
					for i := len(r.middlewares) - 1; i >= 0; i-- {
						finalHandler = r.middlewares[i](finalHandler)
					}
					finalHandler(w, req)
				} else {
					httptransport.RespondWithAPIError(req.Context(), w, contracts.NewMethodNotAllowedError(fmt.Sprintf("Method %s not allowed for path %s.", req.Method, req.URL.Path)))
				}
			})
		}

		r.handlers[path][method] = handler
	}
}

func (r *router) Get(path string, handler http.HandlerFunc) {
	r.handle("GET", path, handler)
}

func (r *router) Post(path string, handler http.HandlerFunc) {
	r.handle("POST", path, handler)
}

func (r *router) Put(path string, handler http.HandlerFunc) {
	r.handle("PUT", path, handler)
}

func (r *router) Patch(path string, handler http.HandlerFunc) {
	r.handle("PATCH", path, handler)
}

func (r *router) Delete(path string, handler http.HandlerFunc) {
	r.handle("DELETE", path, handler)
}

func (r *router) RegisterWithMiddleware(method, path string, handler http.HandlerFunc, middlewares ...func(http.HandlerFunc) http.HandlerFunc) {
	finalHandler := handler
	for i := len(middlewares) - 1; i >= 0; i-- {
		finalHandler = middlewares[i](finalHandler)
	}

	for i := len(r.middlewares) - 1; i >= 0; i-- {
		finalHandler = r.middlewares[i](finalHandler)
	}

	r.handle(method, path, finalHandler)
}

func (r *router) GetRoutes() []any {
	routes := make([]any, len(r.routes))
	for i, route := range r.routes {
		routes[i] = map[string]any{
			"Method":      route.Method,
			"Path":        route.Path,
			"PathPattern": route.PathPattern,
		}
	}
	return routes
}
