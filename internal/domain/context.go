package domain

// Intent определяет, что именно хочет клиент в сообщении.
type Intent string

const (
	IntentNewOrder Intent = "new_order"
	IntentQuestion Intent = "question"
	IntentOther    Intent = "other"
)

// OrderContext — структурированная выжимка из хаотичного сообщения.
type OrderContext struct {
	Intent       Intent `json:"intent"`
	ClientName   string `json:"client_name"`
	Items        string `json:"items"`
	Budget       int    `json:"budget"`        // В тенге
	DeliveryDate string `json:"delivery_date"` // ISO 8601
	Notes        string `json:"notes"`         // Особые пожелания
}

// ContextExtractor — порт для извлечения бизнес-сущностей из текста.
type ContextExtractor interface {
	Extract(rawText string) (OrderContext, error)
}
