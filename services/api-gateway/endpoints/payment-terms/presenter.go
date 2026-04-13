package paymenttermep

import (
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	pb "github.com/augno/api/shared/proto/core"
)

func PaymentTermPresenter(pt *pb.PaymentTermInfo, ownerAccount *apiresource.Account) apiresource.PaymentTerm {
	if pt == nil {
		return apiresource.PaymentTerm{}
	}

	return apiresource.PaymentTerm{
		ID:        pt.Id,
		Object:    constants.ObjectTypePaymentTerm,
		Name:      pt.Name,
		Status:    constants.PaymentTermStatus(pt.Status),
		Owner:     apiresource.NewOwnerWithAccount(pt.AccountId, ownerAccount),
		CreatedAt: grpcutil.TimestampToTime(pt.CreatedAt),
		UpdatedAt: grpcutil.TimestampToTime(pt.UpdatedAt),
	}
}

func PaymentTermListPresenter(resp *pb.ListPaymentTermsResponse, ownerAccount *apiresource.Account) *apiresource.List[apiresource.PaymentTerm] {
	if resp == nil {
		return apiresource.NewList[apiresource.PaymentTerm](nil, apiresource.PageInfo{})
	}

	paymentTerms := make([]apiresource.PaymentTerm, len(resp.PaymentTerms))
	for i, pt := range resp.PaymentTerms {
		paymentTerms[i] = PaymentTermPresenter(pt, ownerAccount)
	}

	return apiresource.NewList(paymentTerms, grpcutil.MapProtoPageInfo(resp.PageInfo))
}
