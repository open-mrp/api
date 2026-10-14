package lifecycle

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
)

type job struct{ ID string }

func handler(any) apiendpoint.ServiceHandler[*Request, *job] {
	return func(context.Context, *Request) (*job, *apierror.APIError) { return &job{}, nil }
}

func TestAction(t *testing.T) {
	t.Parallel()
	ep := Action("Release Job", "/v1/operations/jobs/{id}/actions/release", constants.ObjectTypeJob, nil, handler)
	assert.Equal(t, http.MethodPost, ep.Method)
	assert.Equal(t, http.StatusOK, ep.SuccessStatusCode)
	assert.True(t, ep.Public)
	assert.True(t, ep.AgentTool)
	assert.False(t, ep.Preview)
	assert.Equal(t, constants.ObjectTypeJob, ep.ObjectType)

	Action("Create Jobs", "/v1/sales/sales-orders/{id}/actions/create-jobs", constants.ObjectTypeJob, nil, handler)
}

func TestAction_RejectsOtherRoutes(t *testing.T) {
	t.Parallel()
	for _, route := range []string{
		"/v1/operations/jobs/{id}/release",
		"/v1/operations/jobs/actions/release",
		"/v1/operations/jobs/{id}/actions/Release",
		"/operations/jobs/{id}/actions/release",
	} {
		assert.Panics(t, func() { Action("x", route, constants.ObjectTypeJob, nil, handler) }, route)
	}
	assert.Panics(t, func() { Action("x", "/v1/operations/jobs/{id}/actions/release", "", nil, handler) })
}
