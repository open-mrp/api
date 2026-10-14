package batchreleaseep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/field"
)

// Request to replace a department's release policy.
type UpdateReleasePolicyRequest struct {
	// ID of the department.
	DepartmentID string `path:"id" validate:"required"`
	// Who must sign a batch off before it leaves the department.
	Requirement constants.ReleaseRequirement `json:"requirement" validate:"required"`
	// What happens when a batch that hasn't been signed off is scanned onward. Defaults to `block`.
	Enforcement field.Optional[constants.ReleaseEnforcement] `json:"enforcement,omitzero"`
	// Whether the operator who scanned a batch may sign it off in this department. Defaults to `allowed`.
	OperatorSigning field.Optional[constants.ReleaseOperatorSigning] `json:"operator_signing,omitzero"`
	// Why the department needs, or doesn't need, a sign-off: its risk assessment. Maximum 2000 characters.
	Rationale field.Optional[string] `json:"rationale,omitzero" validate:"omitempty,max=2000"`
	// Yes/no questions the signer answers, in the order they are asked. At most 20, each at most 255 characters.
	Checklist []string `json:"checklist,omitzero"`
}

var sampleUpdateReleasePolicyRequest = &UpdateReleasePolicyRequest{
	DepartmentID:    apiresource.SampleDepartmentID,
	Requirement:     constants.ReleaseRequirementFinal,
	Enforcement:     field.Some(constants.ReleaseEnforcementBlock),
	OperatorSigning: field.Some(constants.ReleaseOperatorSigningForbidden),
	Rationale:       field.Some("Last check before the product ships."),
	Checklist:       []string{"Labeling and packaging correct?"},
}

func (*UpdateReleasePolicyRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateReleasePolicyRequest)
}

// Replaces the sign-off a department requires.
//
// The whole policy is replaced: fields you leave out go back to their defaults. Batches already in the department are held to the new policy from their next scan.
type UpdateReleasePolicyEndpoint struct{}

func (e *UpdateReleasePolicyEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateReleasePolicyRequest, *apiresource.ReleasePolicy] {
	return (&apiendpoint.APIEndpoint[*UpdateReleasePolicyRequest, *apiresource.ReleasePolicy]{
		Title:             "Update Release Policy",
		Method:            http.MethodPut,
		ContentType:       "application/json",
		Route:             "/v1/operations/departments/{id}/release-policy",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypeReleasePolicy,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainDepartments, Action: types.ActionUpdate},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateReleasePolicyRequest) (*apiresource.ReleasePolicy, *apierror.APIError) {
			return svc.(BatchReleaseSvc).UpdateReleasePolicy
		},
	})
}
