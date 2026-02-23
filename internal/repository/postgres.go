package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/florist-agent/internal/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Postgres реализует domain.Repository через PostgreSQL + pgx.
type Postgres struct {
	pool *pgxpool.Pool
}

// NewPostgres создаёт репозиторий, принимая уже готовый пул соединений.
func NewPostgres(pool *pgxpool.Pool) *Postgres {
	return &Postgres{pool: pool}
}

// Migrate создаёт таблицу orders, если она не существует.
func (p *Postgres) Migrate(ctx context.Context) error {
	query := `
	CREATE TABLE IF NOT EXISTS orders (
		id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		chat_id     BIGINT NOT NULL,
		raw_text    TEXT NOT NULL DEFAULT '',
		context     JSONB NOT NULL,
		created_at  TIMESTAMPTZ DEFAULT now()
	);
	CREATE INDEX IF NOT EXISTS idx_orders_chat_id ON orders(chat_id);
	CREATE INDEX IF NOT EXISTS idx_orders_created_at ON orders(created_at);
	`
	_, err := p.pool.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("repository.Migrate: %w", err)
	}
	return nil
}

// SaveOrder вставляет заказ в таблицу orders.
// OrderContext сериализуется в JSONB.
func (p *Postgres) SaveOrder(ctx context.Context, order domain.Order) error {
	contextJSON, err := json.Marshal(order.Context)
	if err != nil {
		return fmt.Errorf("repository.SaveOrder: marshal context: %w", err)
	}

	query := `
	INSERT INTO orders (chat_id, raw_text, context, created_at)
	VALUES ($1, $2, $3, $4)
	`

	_, err = p.pool.Exec(ctx, query, order.ChatID, order.RawText, contextJSON, order.CreatedAt)
	if err != nil {
		return fmt.Errorf("repository.SaveOrder: insert: %w", err)
	}

	return nil
}
