package mapper

import (
	"strings"

	commonv1 "github.com/exchange-grpc/proto/pb/common/v1"
)

// MoneyFromString создаёт Money из строки (валюта по умолчанию USD).
func MoneyFromString(amount string) *commonv1.Money {
	amount = strings.TrimSpace(amount)
	if amount == "" {
		return nil
	}
	return &commonv1.Money{Amount: amount, Currency: "USD"}
}

// DecimalFromString создаёт Decimal из строки.
func DecimalFromString(value string) *commonv1.Decimal {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &commonv1.Decimal{Value: value}
}

// UuidToString извлекает строковое значение из Uuid.
func UuidToString(id *commonv1.Uuid) string {
	if id == nil {
		return ""
	}
	return strings.TrimSpace(id.GetValue())
}
