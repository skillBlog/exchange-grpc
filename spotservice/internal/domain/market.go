package domain

import (
	"fmt"
	"strings"

	"github.com/exchange-grpc/shared/roles"
)

// Market — спотовая торговая пара, доступная на бирже.
type Market struct {
	ID                string
	Name              string
	BaseAsset         string
	QuoteAsset        string
	Enabled           bool
	AllowedRoles      []string
	MinOrderSize      string
	QuantityPrecision uint32
	MinNotional       string
}

const maxMarketIDLen = 64

// NewMarket создаёт Market с нормализацией ролей и базовой валидацией.
func NewMarket(id, name, baseAsset, quoteAsset string, enabled bool, allowedRoles []string) (Market, error) {
	id = strings.TrimSpace(id)
	name = strings.TrimSpace(name)
	if id == "" {
		return Market{}, fmt.Errorf("%w: market id is required", ErrInvalidArgument)
	}
	if len(id) > maxMarketIDLen {
		return Market{}, fmt.Errorf("%w: market id exceeds %d characters", ErrInvalidArgument, maxMarketIDLen)
	}
	if name == "" {
		return Market{}, fmt.Errorf("%w: market name is required", ErrInvalidArgument)
	}

	return Market{
		ID:           id,
		Name:         name,
		BaseAsset:    strings.TrimSpace(baseAsset),
		QuoteAsset:   strings.TrimSpace(quoteAsset),
		Enabled:      enabled,
		AllowedRoles: roles.NormalizeStrings(allowedRoles),
	}, nil
}

// IsActive сообщает, доступен ли рынок для торговли.
func (m Market) IsActive() bool {
	return m.Enabled
}

// IsAccessibleBy проверяет, есть ли у пользователя роль, разрешающая доступ к рынку.
// Пустой список AllowedRoles означает, что рынок открыт для всех.
func (m Market) IsAccessibleBy(userRoles []string) bool {
	return roles.Match(m.AllowedRoles, userRoles)
}
