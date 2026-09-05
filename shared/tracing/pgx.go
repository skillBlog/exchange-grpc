package tracing

import (
	"github.com/exaring/otelpgx"
	"github.com/jackc/pgx/v5"
)

// NewPGXTracer возвращает tracer для pgxpool.Config.ConnConfig.Tracer.
func NewPGXTracer() pgx.QueryTracer {
	return otelpgx.NewTracer()
}
