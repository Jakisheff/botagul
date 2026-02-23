package domain

import (
	"context"
	"time"
)

// Order — заказ, обогащённый метаданными (кто, когда, откуда).
type Order struct {
	ID        string       `json:"id"`
	ChatID    int64        `json:"chat_id"`
	Context   OrderContext `json:"context"`
	RawText   string       `json:"raw_text"`
	CreatedAt time.Time    `json:"created_at"`
}

// Repository — порт для персистентного хранения заказов.
type Repository interface {
	SaveOrder(ctx context.Context, order Order) error
}
