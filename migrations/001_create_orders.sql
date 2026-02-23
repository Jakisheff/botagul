-- Таблица заказов для флористического бота.
-- OrderContext хранится в JSONB для гибкой аналитики.

CREATE TABLE IF NOT EXISTS orders (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    chat_id     BIGINT NOT NULL,
    raw_text    TEXT NOT NULL DEFAULT '',
    context     JSONB NOT NULL,
    created_at  TIMESTAMPTZ DEFAULT now()
);

-- Индексы для типовых запросов
CREATE INDEX IF NOT EXISTS idx_orders_chat_id ON orders(chat_id);
CREATE INDEX IF NOT EXISTS idx_orders_created_at ON orders(created_at);
