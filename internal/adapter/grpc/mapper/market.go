package mapper

import (
	"github.com/exchange-grpc/api/proto/exchange/v1"
	"github.com/exchange-grpc/internal/domain"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// MarketToProto преобразует доменный рынок в protobuf-представление.
func MarketToProto(market domain.Market) *exchangev1.Market {
	protoMarket := &exchangev1.Market{
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

// MarketsToProto преобразует срез доменных рынков в protobuf-сообщения.
func MarketsToProto(markets []domain.Market) []*exchangev1.Market {
	result := make([]*exchangev1.Market, 0, len(markets))
	for _, market := range markets {
		result = append(result, MarketToProto(market))
	}
	return result
}
