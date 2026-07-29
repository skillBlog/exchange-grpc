package spotclient

import (
	"context"
	"fmt"
	"time"

	commonv1 "github.com/exchange-grpc/proto/pb/common/v1"
	spotv1 "github.com/exchange-grpc/proto/pb/spot/v1"
	"github.com/exchange-grpc/orderservice/internal/application"
	"github.com/exchange-grpc/orderservice/internal/domain"
	"github.com/exchange-grpc/shared/grpc"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	googlegrpc "google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

const defaultGRPCTimeout = 5 * time.Second

// Client проверяет доступность рынка через SpotService.
type Client struct {
	api     spotv1.SpotServiceClient
	timeout time.Duration
}

// New создаёт обёртку gRPC-клиента Spot.
func New(conn googlegrpc.ClientConnInterface, timeout time.Duration) *Client {
	if timeout <= 0 {
		timeout = defaultGRPCTimeout
	}
	return &Client{
		api:     spotv1.NewSpotServiceClient(conn),
		timeout: timeout,
	}
}

// Dial открывает gRPC-соединение с клиентскими interceptors.
func Dial(ctx context.Context, target string, opts ...googlegrpc.DialOption) (*googlegrpc.ClientConn, error) {
	dialOpts := []googlegrpc.DialOption{
		googlegrpc.WithTransportCredentials(insecure.NewCredentials()),
		googlegrpc.WithStatsHandler(otelgrpc.NewClientHandler()),
		googlegrpc.WithUnaryInterceptor(grpc.ChainUnaryClient(
			grpc.UnaryClientRequestID,
			grpc.UnaryClientForwardAuthorization,
		)),
	}
	dialOpts = append(dialOpts, opts...)

	return googlegrpc.NewClient(target, dialOpts...)
}

// EnsureMarketAvailable загружает рынок и проверяет, что он доступен для торговли и разрешён пользователю.
func (c *Client) EnsureMarketAvailable(ctx context.Context, marketID string, userRoles []string) error {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	resp, err := c.api.GetMarket(ctx, &spotv1.GetMarketRequest{MarketId: marketID})
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return fmt.Errorf("%w: market %q", domain.ErrNotFound, marketID)
		}
		return fmt.Errorf("get market: %w", err)
	}

	market := resp.GetMarket()
	if market == nil {
		return fmt.Errorf("%w: market %q", domain.ErrNotFound, marketID)
	}
	if !market.GetEnabled() {
		return fmt.Errorf("%w: market %q", domain.ErrMarketInactive, marketID)
	}

	if !domain.IsAccessibleByRoles(protoRolesToStrings(market.GetAllowedRoles()), userRoles) {
		return fmt.Errorf("%w: market %q", domain.ErrForbidden, marketID)
	}

	return nil
}

func protoRolesToStrings(values []commonv1.Role) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		switch value {
		case commonv1.Role_ROLE_USER:
			result = append(result, "user")
		case commonv1.Role_ROLE_TRADER:
			result = append(result, "trader")
		case commonv1.Role_ROLE_ADMIN:
			result = append(result, "admin")
		}
	}
	return result
}

var _ application.MarketChecker = (*Client)(nil)
