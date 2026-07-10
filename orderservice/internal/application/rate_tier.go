package application

import "github.com/exchange-grpc/shared/roles"

// CreateOrderRateTier определяет лимит CreateOrder по ролям пользователя.
type CreateOrderRateTier string

const (
	CreateOrderRateTierBasic   CreateOrderRateTier = "basic"
	CreateOrderRateTierPremium CreateOrderRateTier = "premium"
	CreateOrderRateTierAdmin   CreateOrderRateTier = "admin"
)

// CreateOrderRateTierFromRoles выбирает tier по JWT-ролям (с нормализацией регистра).
func CreateOrderRateTierFromRoles(userRoles []string) CreateOrderRateTier {
	normalized := roles.NormalizeStrings(userRoles)
	if roles.ContainsAny(normalized, roles.RoleAdmin) {
		return CreateOrderRateTierAdmin
	}
	if roles.ContainsAny(normalized, roles.RoleTrader) {
		return CreateOrderRateTierPremium
	}
	return CreateOrderRateTierBasic
}
