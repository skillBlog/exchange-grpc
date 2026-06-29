package spotinstrument

import (
	"context"
	"fmt"

	exchangev1 "github.com/exchange-grpc/api/proto/exchange/v1"
	"github.com/exchange-grpc/internal/domain"
	"github.com/exchange-grpc/internal/platform/interceptor"
	"github.com/exchange-grpc/internal/usecase/order"
	"github.com/exchange-grpc/internal/platform/tracing"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

// Client проверяет доступность рынка через SpotInstrumentService.
type Client struct {
	api exchangev1.SpotInstrumentServiceClient
}

// New создаёт обёртку gRPC-клиента SpotInstrument.
func New(conn grpc.ClientConnInterface) *Client {
	return &Client{api: exchangev1.NewSpotInstrumentServiceClient(conn)}
}

// Dial открывает gRPC-соединение с клиентским интерсептором x-request-id.
func Dial(ctx context.Context, target string, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	dialOpts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(interceptor.UnaryClientRequestID),
	}
	dialOpts = append(dialOpts, tracing.ClientDialOptions()...)
	dialOpts = append(dialOpts, opts...)

	return grpc.NewClient(target, dialOpts...)
}

// EnsureMarketAvailable загружает рынок и проверяет, что он доступен для торговли и разрешён пользователю.
func (c *Client) EnsureMarketAvailable(ctx context.Context, marketID string, userRoles []string) error {
	resp, err := c.api.GetMarket(ctx, &exchangev1.GetMarketRequest{MarketId: marketID})
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
	if !market.GetEnabled() || market.GetDeletedAt() != nil {
		return fmt.Errorf("%w: market %q", domain.ErrMarketInactive, marketID)
	}

	access := domain.Market{AllowedRoles: append([]string(nil), market.GetAllowedRoles()...)}
	if !access.IsAccessibleBy(userRoles) {
		return fmt.Errorf("%w: market %q", domain.ErrForbidden, marketID)
	}

	return nil
}

var _ order.MarketChecker = (*Client)(nil)
