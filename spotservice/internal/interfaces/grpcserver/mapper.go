package grpcserver

import (
	"errors"

	commonv1 "github.com/exchange-grpc/proto/pb/common/v1"
	"github.com/exchange-grpc/spotservice/internal/domain"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func toGRPCError(err error) error {
	if err == nil {
		return nil
	}

	switch {
	case errors.Is(err, domain.ErrInvalidArgument):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, domain.ErrNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, domain.ErrMarketInactive):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, domain.ErrForbidden):
		return status.Error(codes.PermissionDenied, err.Error())
	default:
		return status.Error(codes.Internal, err.Error())
	}
}

func marketToProto(market domain.Market) *commonv1.Market {
	protoMarket := &commonv1.Market{
		Id:           market.ID,
		Name:         market.Name,
		BaseAsset:    market.BaseAsset,
		QuoteAsset:   market.QuoteAsset,
		Enabled:      market.Enabled,
		AllowedRoles: append([]string(nil), market.AllowedRoles...),
	}
	if market.DeletedAt != nil {
		protoMarket.DeletedAt = timestamppb.New(*market.DeletedAt)
	}
	return protoMarket
}

func marketsToProto(markets []domain.Market) []*commonv1.Market {
	result := make([]*commonv1.Market, 0, len(markets))
	for _, market := range markets {
		result = append(result, marketToProto(market))
	}
	return result
}
