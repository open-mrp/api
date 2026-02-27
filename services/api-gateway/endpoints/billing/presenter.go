package billingep

import (
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	pb "github.com/augno/api/shared/proto/billing"
)

func PricingPlanPresenter(p *pb.PricingPlan) apiresource.PricingPlan {
	if p == nil {
		return apiresource.PricingPlan{}
	}

	limits := make([]apiresource.PlanLimit, len(p.Limits))
	for i, l := range p.Limits {
		var value *int
		if l.Value != nil {
			v := int(*l.Value)
			value = &v
		}
		limits[i] = apiresource.PlanLimit{
			Key:   l.Key,
			Value: value,
		}
	}

	var pricePerMonth *float64
	if p.PricePerMonth != nil {
		pricePerMonth = p.PricePerMonth
	}

	var seatMinimum *int
	if p.SeatMinimum != nil {
		v := int(*p.SeatMinimum)
		seatMinimum = &v
	}

	var includesPreviousPlan *string
	if p.IncludesPreviousPlan != nil {
		includesPreviousPlan = p.IncludesPreviousPlan
	}

	return apiresource.PricingPlan{
		ID:                   p.Id,
		Object:               constants.ObjectTypePricingPlan,
		Name:                 p.Name,
		PlanTypeCode:         constants.PublicPlanCode(p.PlanTypeCode),
		PricePerSeat:         p.PricePerSeat,
		PricePerMonth:        pricePerMonth,
		SeatMinimum:          seatMinimum,
		Limits:               limits,
		DisplayFeatures:      p.DisplayFeatures,
		DisplayOrder:         int(p.DisplayOrder),
		IsHighlighted:        p.IsHighlighted,
		ButtonText:           p.ButtonText,
		IncludesPreviousPlan: includesPreviousPlan,
	}
}

func PricingPlansListPresenter(resp *pb.ListPricingPlansResponse) *apiresource.List[apiresource.PricingPlan] {
	if resp == nil {
		return apiresource.NewList([]apiresource.PricingPlan{}, apiresource.PageInfo{})
	}

	plans := make([]apiresource.PricingPlan, len(resp.Plans))
	for i, p := range resp.Plans {
		plans[i] = PricingPlanPresenter(p)
	}

	return apiresource.NewList(plans, mapProtoPageInfo(resp.PageInfo))
}

func mapProtoPageInfo(pi *pb.PageInfo) apiresource.PageInfo {
	if pi == nil {
		return apiresource.PageInfo{}
	}
	return apiresource.PageInfo{
		NextCursor:  pi.NextCursor,
		PrevCursor:  pi.PrevCursor,
		HasNextPage: pi.HasNextPage,
		HasPrevPage: pi.HasPrevPage,
	}
}

func usageItemPresenter(item *pb.UsageItem) apiresource.UsageItem {
	if item == nil {
		return apiresource.UsageItem{}
	}
	ui := apiresource.UsageItem{
		Current: int(item.Current),
	}
	if item.Limit != nil {
		v := int(*item.Limit)
		ui.Limit = &v
	}
	return ui
}

func ProrationPreviewPresenter(resp *pb.GetProrationPreviewResponse) *apiresource.PlanChangeProration {
	if resp == nil || resp.Preview == nil {
		return &apiresource.PlanChangeProration{}
	}

	p := resp.Preview
	lineItems := make([]apiresource.PlanChangeLineItem, len(p.LineItems))
	for i, li := range p.LineItems {
		lineItems[i] = apiresource.PlanChangeLineItem{
			Description: li.Description,
			Amount:      li.Amount,
			IsProration: li.IsProration,
		}
	}

	return &apiresource.PlanChangeProration{
		CreditAmount:                p.CreditAmount,
		ChargeAmount:                p.ChargeAmount,
		NetAmount:                   p.NetAmount,
		FormattedNetAmount:          p.FormattedNetAmount,
		IsCredit:                    p.IsCredit,
		TotalInvoiceAmount:          p.TotalInvoiceAmount,
		FormattedTotalInvoiceAmount: p.FormattedTotalInvoiceAmount,
		MonthlyBillAmount:           p.MonthlyBillAmount,
		FormattedMonthlyBillAmount:  p.FormattedMonthlyBillAmount,
		LineItems:                   lineItems,
	}
}

func SwitchPlanPresenter(resp *pb.SwitchPlanResponse) *apiresource.SwitchPlanResponse {
	if resp == nil {
		return &apiresource.SwitchPlanResponse{}
	}

	return &apiresource.SwitchPlanResponse{
		Success:         resp.Success,
		RequiresPayment: resp.RequiresPayment,
		CheckoutURL:     resp.CheckoutUrl,
	}
}

func ConfirmPlanSwitchPresenter(resp *pb.ConfirmPlanSwitchResponse) *apiresource.ConfirmPlanSwitchResponse {
	if resp == nil {
		return &apiresource.ConfirmPlanSwitchResponse{}
	}

	return &apiresource.ConfirmPlanSwitchResponse{
		Success: resp.Success,
	}
}

func AccountUsagePresenter(resp *pb.GetAccountUsageResponse) *apiresource.AccountUsageResponse {
	if resp == nil {
		return &apiresource.AccountUsageResponse{}
	}

	result := &apiresource.AccountUsageResponse{
		Seats:     usageItemPresenter(resp.Seats),
		Invoices:  usageItemPresenter(resp.Invoices),
		Batches:   usageItemPresenter(resp.Batches),
		Sandboxes: usageItemPresenter(resp.Sandboxes),
	}

	if resp.Subscription != nil {
		result.Subscription = &apiresource.SubscriptionInfo{
			Status:            resp.Subscription.Status,
			CurrentPeriodEnd:  grpcutil.TimestampToTimePtr(resp.Subscription.CurrentPeriodEnd),
			TrialEnd:          grpcutil.TimestampToTimePtr(resp.Subscription.TrialEnd),
			CancelAtPeriodEnd: resp.Subscription.CancelAtPeriodEnd,
			CancelAt:          grpcutil.TimestampToTimePtr(resp.Subscription.CancelAt),
		}
	}

	return result
}
