package domain

import "time"

// Market — спотовая торговая пара, доступная на бирже.
type Market struct {
	ID           string
	Name         string
	BaseAsset    string
	QuoteAsset   string
	Enabled      bool
	DeletedAt    *time.Time
	AllowedRoles []string
}

// IsActive сообщает, доступен ли рынок для торговли: включён и не помечен как удалённый.
func (m Market) IsActive() bool {
	return m.Enabled && m.DeletedAt == nil
}

// IsAccessibleBy проверяет, есть ли у пользователя роль, разрешающая доступ к рынку.
// Пустой список AllowedRoles означает, что рынок открыт для всех.
func (m Market) IsAccessibleBy(userRoles []string) bool {
	if len(m.AllowedRoles) == 0 {
		return true
	}
	if len(userRoles) == 0 {
		return false
	}
	allowed := make(map[string]struct{}, len(m.AllowedRoles))
	for _, role := range m.AllowedRoles {
		allowed[role] = struct{}{}
	}
	for _, role := range userRoles {
		if _, ok := allowed[role]; ok {
			return true
		}
	}
	return false
}
