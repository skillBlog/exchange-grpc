package grpcserver

import (
	"strings"

	commonv1 "github.com/exchange-grpc/proto/pb/common/v1"
	"github.com/exchange-grpc/shared/grpc"
	"github.com/exchange-grpc/shared/roles"
	"github.com/exchange-grpc/spotservice/internal/domain"
)

func toGRPCError(err error) error {
	return grpc.ToStatusError(err, "too many requests")
}

func marketToProto(market domain.Market) *commonv1.Market {
	msg := &commonv1.Market{
		Id:                market.ID,
		Name:              market.Name,
		BaseAsset:         market.BaseAsset,
		QuoteAsset:        market.QuoteAsset,
		Enabled:           market.Enabled,
		AllowedRoles:      rolesToProto(market.AllowedRoles),
		QuantityPrecision: market.QuantityPrecision,
	}
	if market.MinOrderSize != "" {
		msg.MinOrderSize = &commonv1.Decimal{Value: market.MinOrderSize}
	}
	if market.MinNotional != "" {
		msg.MinNotional = &commonv1.Decimal{Value: market.MinNotional}
	}
	return msg
}

func marketsToProto(markets []domain.Market) []*commonv1.Market {
	result := make([]*commonv1.Market, 0, len(markets))
	for _, market := range markets {
		result = append(result, marketToProto(market))
	}
	return result
}

func rolesToProto(values []string) []commonv1.Role {
	result := make([]commonv1.Role, 0, len(values))
	for _, value := range values {
		if strings.TrimSpace(value) == "" {
			continue
		}
		role, ok := roles.Parse(value)
		if !ok {
			result = append(result, commonv1.Role_ROLE_UNSPECIFIED)
			continue
		}
		switch role {
		case roles.RoleUser:
			result = append(result, commonv1.Role_ROLE_USER)
		case roles.RoleTrader:
			result = append(result, commonv1.Role_ROLE_TRADER)
		case roles.RoleAdmin:
			result = append(result, commonv1.Role_ROLE_ADMIN)
		default:
			result = append(result, commonv1.Role_ROLE_UNSPECIFIED)
		}
	}
	return result
}
