package registrationflowep

import (
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	pb "github.com/augno/api/shared/proto/core"
)

func RegistrationFlowOptionPresenter(opt *pb.RegistrationFlowOptionInfo) apiresource.RegistrationFlowOption {
	if opt == nil {
		return apiresource.RegistrationFlowOption{}
	}

	return apiresource.RegistrationFlowOption{
		ID:     opt.Id,
		Object: constants.ObjectTypeRegistrationFlowOption,
		Name:   opt.Name,
	}
}

func RegistrationFlowPresenter(rf *pb.RegistrationFlowInfo) apiresource.RegistrationFlow {
	if rf == nil {
		return apiresource.RegistrationFlow{}
	}

	customerGroupOptions := make([]apiresource.RegistrationFlowOption, len(rf.CustomerGroupOptions))
	for i, opt := range rf.CustomerGroupOptions {
		customerGroupOptions[i] = RegistrationFlowOptionPresenter(opt)
	}

	paymentTermOptions := make([]apiresource.RegistrationFlowOption, len(rf.PaymentTermOptions))
	for i, opt := range rf.PaymentTermOptions {
		paymentTermOptions[i] = RegistrationFlowOptionPresenter(opt)
	}

	shippingTermOptions := make([]apiresource.RegistrationFlowOption, len(rf.ShippingTermOptions))
	for i, opt := range rf.ShippingTermOptions {
		shippingTermOptions[i] = RegistrationFlowOptionPresenter(opt)
	}

	return apiresource.RegistrationFlow{
		ID:                   rf.Id,
		Object:               constants.ObjectTypeRegistrationFlow,
		Name:                 rf.Name,
		CustomerGroupOptions: customerGroupOptions,
		PaymentTermOptions:   paymentTermOptions,
		ShippingTermOptions:  shippingTermOptions,
		CreatedAt:            grpcutil.TimestampToTime(rf.CreatedAt),
		UpdatedAt:            grpcutil.TimestampToTime(rf.UpdatedAt),
	}
}

func RegistrationFlowListPresenter(resp *pb.ListRegistrationFlowsResponse) *apiresource.List[apiresource.RegistrationFlow] {
	if resp == nil {
		return apiresource.NewList[apiresource.RegistrationFlow](nil, apiresource.PageInfo{})
	}

	registrationFlows := make([]apiresource.RegistrationFlow, len(resp.RegistrationFlows))
	for i, rf := range resp.RegistrationFlows {
		registrationFlows[i] = RegistrationFlowPresenter(rf)
	}

	return apiresource.NewList(registrationFlows, grpcutil.MapProtoPageInfo(resp.PageInfo))
}
