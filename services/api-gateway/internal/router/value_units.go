package router

import (
	"context"
	"reflect"
	"sync"

	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
	"github.com/open-mrp/apikit/include"
	"github.com/open-mrp/apikit/object"
	kitrouter "github.com/open-mrp/apikit/router"

	"github.com/open-mrp/api/services/api-gateway/internal/resourceloaders"
)

// groupRegistry registers endpoint groups so that every value shape in a response reaches the client with its
// units loaded (resourceloaders.FillUnits): after the service handler for what it built, and after each include
// loader for what includes bring in.
type groupRegistry struct {
	*kitrouter.Registry
}

func newGroupRegistry() *groupRegistry {
	fillIncludeLoaderUnits()
	return &groupRegistry{Registry: kitrouter.NewRegistry()}
}

func (r *groupRegistry) RegisterGroup(group *apiendpoint.APIEndpointGroup) {
	for _, ep := range group.Endpoints {
		fillHandlerUnits(ep)
	}
	r.Registry.RegisterGroup(group)
}

var (
	contextType  = reflect.TypeFor[context.Context]()
	apiErrorType = reflect.TypeFor[*apierror.APIError]()
)

// fillHandlerUnits wraps an endpoint's service handler so its response leaves with every unit stub loaded. Endpoints
// whose response type cannot hold a unit are left alone.
func fillHandlerUnits(ep apiendpoint.APIEndpointer) {
	if !resourceloaders.MayHoldUnit(ep.GetResponseType()) {
		return
	}
	v := reflect.ValueOf(ep)
	if v.Kind() != reflect.Pointer || v.Elem().Kind() != reflect.Struct {
		return
	}
	field := v.Elem().FieldByName("ServiceHandler")
	if !field.IsValid() || field.Kind() != reflect.Func || field.IsNil() || !field.CanSet() {
		return
	}
	bind := reflect.ValueOf(field.Interface())
	handlerType := field.Type().Out(0)
	if handlerType.NumIn() != 2 || handlerType.In(0) != contextType || handlerType.NumOut() != 2 || handlerType.Out(1) != apiErrorType {
		return
	}
	field.Set(reflect.MakeFunc(field.Type(), func(args []reflect.Value) []reflect.Value {
		handler := bind.Call(args)[0]
		if handler.IsNil() {
			return []reflect.Value{handler}
		}
		return []reflect.Value{reflect.MakeFunc(handlerType, func(in []reflect.Value) []reflect.Value {
			out := handler.Call(in)
			if !out[1].IsNil() {
				return out
			}
			ctx := in[0].Interface().(context.Context)
			if apiErr := resourceloaders.FillUnits(ctx, out[0].Interface()); apiErr != nil {
				out[1] = reflect.ValueOf(apiErr)
			}
			return out
		})}
	}))
}

var fillIncludeLoaderUnitsOnce sync.Once

// fillIncludeLoaderUnits wraps every registered include loader, and every include field's Populate, so what an
// include brings in carries its units too: a loader's records, and the values a presenter stashed for a field.
func fillIncludeLoaderUnits() {
	fillIncludeLoaderUnitsOnce.Do(func() {
		for _, name := range object.Type("").EnumValues() {
			def := include.Lookup(object.Type(name))
			if def == nil {
				continue
			}
			load := def.Load
			def.Load = func(ctx context.Context, ids []string) (map[string]any, *apierror.APIError) {
				loaded, apiErr := load(ctx, ids)
				if apiErr != nil || len(loaded) == 0 {
					return loaded, apiErr
				}
				if apiErr := resourceloaders.FillUnits(ctx, loaded); apiErr != nil {
					return nil, apiErr
				}
				return loaded, nil
			}
			for i := range def.Subs {
				populate := def.Subs[i].Populate
				if populate == nil {
					continue
				}
				def.Subs[i].Populate = func(ctx context.Context, parent any, loaded map[string]any) {
					populate(ctx, parent, loaded)
					// Populate cannot fail, so a unit that does not load stays a stub.
					_ = resourceloaders.FillUnits(ctx, parent)
				}
			}
		}
	})
}
