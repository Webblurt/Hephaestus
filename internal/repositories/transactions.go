package repositories

import (
	"context"

	"github.com/jackc/pgx/v5"
)

func (r *Repository) BeginTx(ctx context.Context) (pgx.Tx, error) {
	return r.DB.Begin(ctx)
}
