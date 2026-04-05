package accountep

import (
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	pb "github.com/augno/api/shared/proto/core"
)

func AccountPresenter(a *pb.AccountInfo) apiresource.Account {
	if a == nil {
		return apiresource.Account{}
	}

	account := apiresource.Account{
		ID:        a.Id,
		Object:    constants.ObjectTypeAccount,
		Name:      a.Name,
		CreatedAt: grpcutil.TimestampToTime(a.CreatedAt),
		UpdatedAt: grpcutil.TimestampToTime(a.UpdatedAt),
	}

	if a.DefaultBillingAddressId != nil {
		account.DefaultBillingAddress = &apiresource.Address{
			ID:     *a.DefaultBillingAddressId,
			Object: constants.ObjectTypeAddress,
		}
	}

	if a.DefaultShippingAddressId != nil {
		account.DefaultShippingAddress = &apiresource.Address{
			ID:     *a.DefaultShippingAddressId,
			Object: constants.ObjectTypeAddress,
		}
	}

	if a.Branding != nil {
		account.Branding = &apiresource.AccountBranding{
			ID:              a.Branding.Id,
			Object:          constants.ObjectTypeAccountBranding,
			SupportEmail:    a.Branding.SupportEmail,
			PhoneNumber:     a.Branding.PhoneNumber,
			LogoURL:         a.Branding.LogoUrl,
			FacebookHandle:  a.Branding.FacebookHandle,
			InstagramHandle: a.Branding.InstagramHandle,
			LinkedInHandle:  a.Branding.LinkedinHandle,
			TwitterHandle:   a.Branding.TwitterHandle,
			WebsiteURL:      a.Branding.WebsiteUrl,
			CreatedAt:       grpcutil.TimestampToTime(a.Branding.CreatedAt),
			UpdatedAt:       grpcutil.TimestampToTime(a.Branding.UpdatedAt),
		}
	}

	if a.Portal != nil {
		account.Portal = &apiresource.AccountPortal{
			ID:        a.Portal.Id,
			Object:    constants.ObjectTypeAccountPortal,
			Slug:      a.Portal.Slug,
			CreatedAt: grpcutil.TimestampToTime(a.Portal.CreatedAt),
			UpdatedAt: grpcutil.TimestampToTime(a.Portal.UpdatedAt),
		}
	}

	return account
}

func PublicAccountPresenter(a *pb.PublicAccountInfo) apiresource.PublicAccount {
	if a == nil {
		return apiresource.PublicAccount{}
	}

	pub := apiresource.PublicAccount{
		ID:           a.Id,
		Object:       constants.ObjectTypePublicAccount,
		Name:         a.Name,
		Slug:         a.Slug,
		SupportEmail: a.SupportEmail,
		LogoURL:      a.LogoUrl,
	}

	if a.DefaultBillingAddressId != nil {
		pub.DefaultBillingAddress = &apiresource.Address{
			ID:     *a.DefaultBillingAddressId,
			Object: constants.ObjectTypeAddress,
		}
	}

	return pub
}
