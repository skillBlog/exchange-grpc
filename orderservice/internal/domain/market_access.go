package domain

import "github.com/exchange-grpc/shared/roles"

// IsAccessibleByRoles проверяет RBAC для allowed_roles рынка.
// Пустой allowedRoles означает, что рынок открыт для всех.
func IsAccessibleByRoles(allowedRoles, userRoles []string) bool {
	return roles.Match(allowedRoles, userRoles)
}
