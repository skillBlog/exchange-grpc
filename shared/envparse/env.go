package envparse

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Loader читает env и останавливается на первой ошибке парсинга.
type Loader struct {
	err error
}

// Err возвращает первую ошибку парсинга duration/int, если она была.
func (l *Loader) Err() error {
	return l.err
}

// String возвращает значение переменной или fallback, если её нет.
func (l *Loader) String(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return fallback
}

// Duration читает time.Duration. Нет переменной или пустое значение — fallback.
// Заданное, но непарсабельное значение — ошибка с именем переменной.
func (l *Loader) Duration(key string, fallback time.Duration) time.Duration {
	if l.err != nil {
		return fallback
	}
	raw, ok := lookupNonEmpty(key)
	if !ok {
		return fallback
	}
	parsed, err := time.ParseDuration(raw)
	if err != nil {
		l.err = fmt.Errorf("parse %s %q: %w", key, raw, err)
		return fallback
	}
	return parsed
}

// Int читает int. Нет переменной или пустое значение — fallback.
// Заданное, но непарсабельное значение — ошибка с именем переменной.
func (l *Loader) Int(key string, fallback int) int {
	if l.err != nil {
		return fallback
	}
	raw, ok := lookupNonEmpty(key)
	if !ok {
		return fallback
	}
	parsed, err := strconv.Atoi(raw)
	if err != nil {
		l.err = fmt.Errorf("parse %s %q: %w", key, raw, err)
		return fallback
	}
	return parsed
}

func lookupNonEmpty(key string) (string, bool) {
	value, ok := os.LookupEnv(key)
	if !ok {
		return "", false
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return "", false
	}
	return value, true
}
