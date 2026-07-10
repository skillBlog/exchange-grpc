package roles

import (
	"fmt"
	"strings"
)

// Role — типизированная роль пользователя в системе.
type Role string

const (
	RoleUser   Role = "user"
	RoleTrader Role = "trader"
	RoleAdmin  Role = "admin"
)

var validRoles = map[Role]struct{}{
	RoleUser:   {},
	RoleTrader: {},
	RoleAdmin:  {},
}

// Normalize приводит строку роли к каноническому lower-case виду.
func Normalize(raw string) Role {
	return Role(strings.ToLower(strings.TrimSpace(raw)))
}

// Parse нормализует и проверяет, что роль известна системе.
func Parse(raw string) (Role, bool) {
	role := Normalize(raw)
	if role == "" {
		return "", false
	}
	_, ok := validRoles[role]
	return role, ok
}

// Validate возвращает ошибку, если роль неизвестна.
func Validate(role Role) error {
	if _, ok := validRoles[role]; !ok {
		return fmt.Errorf("invalid role: %q", role)
	}
	return nil
}

// NormalizeStrings нормализует список ролей, отбрасывая пустые и дубликаты.
func NormalizeStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}

	seen := make(map[Role]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		role := Normalize(value)
		if role == "" {
			continue
		}
		if _, ok := seen[role]; ok {
			continue
		}
		seen[role] = struct{}{}
		result = append(result, string(role))
	}
	return result
}

// ContainsAny проверяет пересечение ролей пользователя с разрешённым набором.
func ContainsAny(userRoles []string, allowed ...Role) bool {
	if len(allowed) == 0 {
		return true
	}
	if len(userRoles) == 0 {
		return false
	}

	allowedSet := make(map[Role]struct{}, len(allowed))
	for _, role := range allowed {
		allowedSet[role] = struct{}{}
	}

	for _, raw := range userRoles {
		role, ok := Parse(raw)
		if !ok {
			continue
		}
		if _, ok := allowedSet[role]; ok {
			return true
		}
	}
	return false
}

// Match проверяет, есть ли у пользователя роль из allowedRoles.
// Пустой allowedRoles означает доступ для всех.
func Match(allowedRoles, userRoles []string) bool {
	if len(allowedRoles) == 0 {
		return true
	}
	if len(userRoles) == 0 {
		return false
	}

	allowed := make(map[Role]struct{}, len(allowedRoles))
	for _, raw := range allowedRoles {
		if role, ok := Parse(raw); ok {
			allowed[role] = struct{}{}
		}
	}

	for _, raw := range userRoles {
		if role, ok := Parse(raw); ok {
			if _, ok := allowed[role]; ok {
				return true
			}
		}
	}
	return false
}
