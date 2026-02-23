package llm

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/florist-agent/internal/domain"
)

// Extractor реализует domain.ContextExtractor через вызовы LLM.
type Extractor struct {
	client *Client
}

func NewExtractor(client *Client) *Extractor {
	return &Extractor{client: client}
}

const floristSystemPromptTemplate = `Ты — ИИ-ассистент флориста в Казахстане. 
Твоя задача: извлекать данные о заказах из переписки в Telegram.
Клиенты пишут на смеси русского и казахского, используют сленг.

Текущая дата и время: %s (часовой пояс: Алматы, UTC+5).

Правила:
1. Верни ТОЛЬКО валидный JSON. Никаких вступлений вроде "Вот ваш ответ".
2. Бюджет указывать только числом в тенге (например, 15000). Если нет — 0.
3. Дата должна быть в формате ISO 8601 (например, 2026-03-08T10:00:00+05:00). Если время не указано, ставь 12:00:00+05:00. Если дата относительная ("завтра", "послезавтра"), вычисли её относительно текущей даты.
4. Если клиент не указал имя, ставь пустую строку.

Формат JSON:
{
  "intent": "new_order" | "question" | "other",
  "client_name": "Имя или пустая строка",
  "items": "Что хотят купить (например: 15 пионов)",
  "budget": 0,
  "delivery_date": "дата или пустая строка",
  "notes": "предпочтения (например: без упаковки, оплата каспи)"
}`

// buildSystemPrompt подставляет текущую дату/время в шаблон промпта.
func buildSystemPrompt() string {
	now := time.Now().In(time.FixedZone("Asia/Almaty", 5*60*60))
	return fmt.Sprintf(floristSystemPromptTemplate, now.Format("2006-01-02 15:04:05"))
}

func (e *Extractor) Extract(rawText string) (domain.OrderContext, error) {
	// 1. Собираем промпт с текущей датой
	systemPrompt := buildSystemPrompt()

	// 2. Отправляем запрос в LLM
	rawResponse, err := e.client.Complete(systemPrompt, rawText, 500)
	if err != nil {
		return domain.OrderContext{}, fmt.Errorf("Extractor.Extract: %w", err)
	}

	// 3. Очищаем ответ от мусора (маркдаун, лишний текст)
	jsonStr, err := cleanJSONPayload(rawResponse)
	if err != nil {
		return domain.OrderContext{}, fmt.Errorf("Extractor.Extract: failed to find JSON: %w", err)
	}

	// 4. Десериализуем в доменную сущность
	var ctx domain.OrderContext
	if err := json.Unmarshal([]byte(jsonStr), &ctx); err != nil {
		return domain.OrderContext{}, fmt.Errorf("Extractor.Extract: unmarshal: %w (payload: %s)", err, jsonStr)
	}

	return ctx, nil
}

// cleanJSONPayload находит первый '{' и последний '}' в строке.
func cleanJSONPayload(raw string) (string, error) {
	start := strings.Index(raw, "{")
	end := strings.LastIndex(raw, "}")

	if start == -1 || end == -1 || start > end {
		return "", errors.New("no valid JSON object structure found in response")
	}

	return raw[start : end+1], nil
}
