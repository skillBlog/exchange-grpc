package integration

import (
	"context"
	"testing"

	orderv1 "github.com/exchange-grpc/proto/pb/order/v1"
	spotv1 "github.com/exchange-grpc/proto/pb/spot/v1"
	ordertestserver "github.com/exchange-grpc/orderservice/pkg/testserver"
	spottestserver "github.com/exchange-grpc/spotservice/pkg/testserver"
	"github.com/exchange-grpc/shared/grpc"
	"google.golang.org/grpc/health/grpc_health_v1"
)

var testTokenService = spottestserver.TestTokenService()

// TestUserID — стабильный UUID пользователя для интеграционных тестов.
const TestUserID = "11111111-1111-1111-1111-111111111111"

// Suite запускает Spot и Order gRPC-сервисы in-process для интеграционных тестов.
type Suite struct {
	SpotClient    spotv1.SpotServiceClient
	HealthClient  grpc_health_v1.HealthClient
	OrderClient   orderv1.OrderServiceClient
	OrderServices ordertestserver.Services
}

// NewSuite подключает оба сервиса с интерсепторами, как в production.
func NewSuite(t *testing.T) *Suite {
	t.Helper()

	spot := spottestserver.NewSpot(t, testTokenService)
	order := ordertestserver.NewOrder(t, spot.Conn, testTokenService)

	return &Suite{
		SpotClient:    spot.Client,
		HealthClient:  spot.HealthClient,
		OrderClient:   order.Client,
		OrderServices: order.Services,
	}
}

// AuthContext добавляет Bearer JWT в исходящий gRPC metadata.
func AuthContext(ctx context.Context, userID string, roles ...string) context.Context {
	authenticated, err := grpc.OutgoingContextWithAuth(ctx, testTokenService, userID, roles)
	if err != nil {
		panic(err)
	}
	return authenticated
}
