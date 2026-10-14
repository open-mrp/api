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

// A signer's answer to one of the step's checklist prompts.
type ChecklistAnswerInput struct {
	// The prompt exactly as the release policy lists it.
	Prompt string `json:"prompt" validate:"required"`
	// The answer.
	Answer constants.ChecklistAnswer `json:"answer" validate:"required"`
}

// Request to sign a batch at the step it is at.
type SignBatchRequest struct {
	// ID of the batch.
	BatchID string `path:"id" validate:"required"`
	// Send `correction` to approve correcting the batch's scan, such as undoing it. Omit it to sign the release the step requires.
	Type field.Optional[constants.BatchSignatureType] `json:"type,omitzero"`
	// Whether the signer approves or rejects.
	Decision constants.BatchSignatureDecision `json:"decision" validate:"required"`
	// Which of the two signatures a `first_article` step needs this one is. Other steps sign in the capacity their requirement sets, so it can be omitted.
	Capacity field.Optional[constants.SignatureCapacity] `json:"capacity,omitzero"`
	// ID of the account user who is signing.
	SignerAccountUserID string `json:"signer_account_user_id" validate:"required"`
	// The signer's signing PIN.
	Pin string `json:"pin" validate:"required" sensitive:"true"`
	// An answer to every prompt on the step's checklist. Omit for a correction.
	Checklist []ChecklistAnswerInput `json:"checklist,omitzero"`
	// Required when rejecting and for a correction: say why. Maximum 2000 characters.
	Note field.Optional[string] `json:"note,omitzero" validate:"omitempty,max=2000"`
}

var sampleSignBatchRequest = &SignBatchRequest{
	BatchID:             apiresource.SampleBatchID,
	Decision:            constants.BatchSignatureDecisionApproved,
	SignerAccountUserID: apiresource.SampleAccountUserID,
	Pin:                 samplePin,
	Checklist: []ChecklistAnswerInput{{
		Prompt: "Labeling and packaging correct?",
		Answer: constants.ChecklistAnswerYes,
	}},
}

func (*SignBatchRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleSignBatchRequest)
}

// Signs a batch off at the production step it is at, or approves a correction to its scan.
//
// The signer is identified by their signing PIN, so a supervisor can sign on the operator's station without signing it out. Who may sign depends on the step's release requirement: quality and final releases need the `quality_releases` permission; supervisor releases need `batch_releases`, and a supervisor assigned to a department signs only for that department.
//
// Rejecting a batch puts it on hold until a later signature approves it, and withdraws an earlier release. An approving final release is refused while any batch earlier in its production still needs a sign-off.
type SignBatchEndpoint struct{}

func (e *SignBatchEndpoint) Materialize() *apiendpoint.APIEndpoint[*SignBatchRequest, *apiresource.BatchSignature] {
	return (&apiendpoint.APIEndpoint[*SignBatchRequest, *apiresource.BatchSignature]{
		Title:             "Sign Batch",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/operations/batches/{id}/signatures",
		SuccessStatusCode: http.StatusCreated,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypeBatchSignature,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainBatches, Action: types.ActionCreate},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *SignBatchRequest) (*apiresource.BatchSignature, *apierror.APIError) {
			return svc.(BatchReleaseSvc).SignBatch
		},
	})
}
