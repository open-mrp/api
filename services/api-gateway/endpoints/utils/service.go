package utilsep

import (
	"context"
	"fmt"

	"github.com/augno/api/services/api-gateway/internal/domain"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	pb "github.com/augno/api/shared/proto/core"
	"github.com/augno/api/shared/tracing"
	"google.golang.org/grpc"
)

type UtilsSvc interface {
	CheckDuplicate(ctx context.Context, req *CheckDuplicateRequest) (*apiresource.CheckDuplicateResult, *apierror.APIError)
	EmailRecord(ctx context.Context, req *EmailRecordRequest) (*apiresource.EmptyResource, *apierror.APIError)
	RequestDemo(ctx context.Context, req *RequestDemoRequest) (*apiresource.MessageResource, *apierror.APIError)
	SubmitFeedback(ctx context.Context, req *SubmitFeedbackRequest) (*apiresource.MessageResource, *apierror.APIError)
}

type UtilsSvcConfig struct {
	// CoreClient (required) is the core-service gRPC client.
	CoreClient pb.CoreServiceClient
}

type utilsSvcImpl struct {
	coreClient pb.CoreServiceClient
}

var utilsSvcTracer = tracing.GetTracer("api-gateway.endpoints.utils.service")

func (c *UtilsSvcConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("utils endpoint service: core client is required")
	}
	return nil
}

func NewUtilsSvc(config *UtilsSvcConfig) UtilsSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &utilsSvcImpl{
		coreClient: config.CoreClient,
	}
}

func (m *utilsSvcImpl) CheckDuplicate(ctx context.Context, req *CheckDuplicateRequest) (*apiresource.CheckDuplicateResult, *apierror.APIError) {
	var pbType pb.DuplicateCheckType
	switch req.Type {
	case "invoice_number":
		pbType = pb.DuplicateCheckType_DUPLICATE_CHECK_TYPE_INVOICE_NUMBER
	case "order_number":
		pbType = pb.DuplicateCheckType_DUPLICATE_CHECK_TYPE_ORDER_NUMBER
	case "customer_po_number":
		pbType = pb.DuplicateCheckType_DUPLICATE_CHECK_TYPE_CUSTOMER_PO_NUMBER
	default:
		return nil, apierror.NewValidationErrorWithParam("Invalid duplicate check type. Must be one of: invoice_number, order_number, customer_po_number.", "type")
	}

	pbReq := &pb.CheckDuplicateRequest{
		Type:         pbType,
		RecordNumber: req.RecordNumber,
		CustomerId:   req.CustomerID.Ptr(),
	}

	resp, apiErr := grpcutil.CallRPC(ctx, utilsSvcTracer, "service.utils.check_duplicate", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.CheckDuplicateResponse, error) {
			return m.coreClient.CheckDuplicate(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := &apiresource.CheckDuplicateResult{
		Object:      constants.ObjectTypeCheckDuplicateResult,
		IsDuplicate: resp.IsDuplicate,
	}
	if resp.Message != nil {
		result.Message = resp.Message
	}

	return result, nil
}

func (m *utilsSvcImpl) EmailRecord(ctx context.Context, req *EmailRecordRequest) (*apiresource.EmptyResource, *apierror.APIError) {
	var pbType pb.EmailRecordType
	switch req.Type {
	case constants.EmailRecordTypeInvoice:
		pbType = pb.EmailRecordType_EMAIL_RECORD_TYPE_INVOICE
	case constants.EmailRecordTypeSalesOrder:
		pbType = pb.EmailRecordType_EMAIL_RECORD_TYPE_SALES_ORDER
	case constants.EmailRecordTypePurchaseOrder:
		pbType = pb.EmailRecordType_EMAIL_RECORD_TYPE_PURCHASE_ORDER
	default:
		return nil, apierror.NewValidationErrorWithParam("Invalid email record type. Must be one of: invoice, sales_order, purchase_order.", "type")
	}

	pbReq := &pb.EmailRecordRequest{
		Id:   req.ID,
		Type: pbType,
	}

	_, apiErr := grpcutil.CallRPC(ctx, utilsSvcTracer, "service.utils.email_record", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.EmailRecordResponse, error) {
			return m.coreClient.EmailRecord(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return &apiresource.EmptyResource{}, nil
}

func (m *utilsSvcImpl) RequestDemo(ctx context.Context, req *RequestDemoRequest) (*apiresource.MessageResource, *apierror.APIError) {
	pbReq := &pb.RequestDemoRequest{
		Name:        req.Name,
		Email:       req.Email,
		Company:     req.Company,
		PhoneNumber: req.PhoneNumber.Ptr(),
		Message:     req.Message.Ptr(),
	}

	resp, apiErr := grpcutil.CallRPC(ctx, utilsSvcTracer, "service.utils.request_demo", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.RequestDemoResponse, error) {
			return m.coreClient.RequestDemo(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return &apiresource.MessageResource{
		Object:  constants.ObjectTypeMessage,
		Message: resp.Message,
	}, nil
}

func (m *utilsSvcImpl) SubmitFeedback(ctx context.Context, req *SubmitFeedbackRequest) (*apiresource.MessageResource, *apierror.APIError) {
	pbReq := &pb.SubmitFeedbackRequest{
		Question: req.Question,
		Answer:   req.Answer,
		PageUrl:  req.PageURL.Ptr(),
	}

	resp, apiErr := grpcutil.CallRPC(ctx, utilsSvcTracer, "service.utils.submit_feedback", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.SubmitFeedbackResponse, error) {
			return m.coreClient.SubmitFeedback(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return &apiresource.MessageResource{
		Object:  constants.ObjectTypeMessage,
		Message: resp.Message,
	}, nil
}
