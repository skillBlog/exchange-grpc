package grpcserver

import (
	"errors"

	commonv1 "github.com/exchange-grpc/proto/pb/common/v1"
	"github.com/exchange-grpc/shared/grpc"
	"github.com/exchange-grpc/spotservice/internal/domain"
)

func toGRPCError(err error) error {
	return grpc.ToStatusError(err, "too many requests")
}

func marketToProto(market domain.Market) *commonv1.Market {
	return &commonv1.Market{
		Id:           market.ID,
		Name:         market.Name,
		BaseAsset:    market.BaseAsset,
		QuoteAsset:   market.QuoteAsset,
		Enabled:      market.Enabled,
		AllowedRoles: append([]string(nil), market.AllowedRoles...),
	}
}

func marketsToProto(markets []domain.Market) []*commonv1.Market {
	result := make([]*commonv1.Market, 0, len(markets))
	for _, market := range markets {
		result = append(result, marketToProto(market))
	}
	return result
}

// IsDomainError сообщает, распознана ли ошибка как domain sentinel.
func IsDomainError(err error) bool {
	return errors.Is(err, domain.ErrInvalidArgument) ||
		errors.Is(err, domain.ErrNotFound) ||
		errors.Is(err, domain.ErrMarketInactive) ||
		errors.Is(err, domain.ErrForbidden)
}
