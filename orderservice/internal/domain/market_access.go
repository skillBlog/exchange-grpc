package domain

// IsAccessibleByRoles проверяет RBAC для allowed_roles рынка.
// Пустой allowedRoles означает, что рынок открыт для всех.
func IsAccessibleByRoles(allowedRoles, userRoles []string) bool {
	if len(allowedRoles) == 0 {
		return true
	}
	if len(userRoles) == 0 {
		return false
	}
	allowed := make(map[string]struct{}, len(allowedRoles))
	for _, role := range allowedRoles {
		allowed[role] = struct{}{}
	}
	for _, role := range userRoles {
		if _, ok := allowed[role]; ok {
			return true
		}
	}
	return false
}
