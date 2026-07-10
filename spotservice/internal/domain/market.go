package domain

import "github.com/exchange-grpc/shared/roles"

// Market — спотовая торговая пара, доступная на бирже.
type Market struct {
	ID           string
	Name         string
	BaseAsset    string
	QuoteAsset   string
	Enabled      bool
	AllowedRoles []string
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
