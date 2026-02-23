package telegram

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/florist-agent/internal/domain"
)

// Handler — HTTP-обработчик вебхуков Telegram.
type Handler struct {
	api            *API
	extractor      domain.ContextExtractor
	repo           domain.Repository
	allowedChatIDs map[int64]bool
}

// NewHandler создаёт обработчик.
// allowedChatIDs — белый список chat ID, которым разрешено взаимодействовать с ботом.
// Если список пуст — принимаются сообщения от всех (небезопасно для прода).
func NewHandler(api *API, extractor domain.ContextExtractor, repo domain.Repository, allowedChatIDs []int64) *Handler {
	allowed := make(map[int64]bool, len(allowedChatIDs))
	for _, id := range allowedChatIDs {
		allowed[id] = true
	}
	return &Handler{
		api:            api,
		extractor:      extractor,
		repo:           repo,
		allowedChatIDs: allowed,
	}
}

// ServeHTTP обрабатывает POST /webhook от Telegram.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var update Update
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		log.Printf("[webhook] failed to decode update: %v", err)
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	// Мгновенный ответ Telegram — чтобы не было ретраев
	w.WriteHeader(http.StatusOK)

	// Фильтруем: нет сообщения или пустой текст — игнорируем
	if update.Message == nil || update.Message.Text == "" {
		return
	}

	// Фильтруем по белому списку чатов
	chatID := update.Message.Chat.ID
	if len(h.allowedChatIDs) > 0 && !h.allowedChatIDs[chatID] {
		log.Printf("[webhook] blocked message from chat_id=%d (not in whitelist)", chatID)
		return
	}

	// Запускаем обработку в горутине, чтобы не блокировать ответ
	go h.processMessage(chatID, update.Message.Text)
}

// processMessage — основная бизнес-логика: извлечение → сохранение → подтверждение.
func (h *Handler) processMessage(chatID int64, rawText string) {
	log.Printf("[process] chat_id=%d text=%q", chatID, rawText)

	// 1. Извлекаем контекст через LLM
	orderCtx, err := h.extractor.Extract(rawText)
	if err != nil {
		log.Printf("[process] extract error: %v", err)
		_ = h.api.SendMessage(chatID, "⚠️ Не удалось разобрать сообщение. Попробуйте ещё раз.")
		return
	}

	log.Printf("[process] extracted: intent=%s items=%q budget=%d", orderCtx.Intent, orderCtx.Items, orderCtx.Budget)

	// 2. Сохраняем в базу
	order := domain.Order{
		ChatID:    chatID,
		Context:   orderCtx,
		RawText:   rawText,
		CreatedAt: time.Now(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := h.repo.SaveOrder(ctx, order); err != nil {
		log.Printf("[process] save error: %v", err)
		_ = h.api.SendMessage(chatID, "⚠️ Ошибка сохранения заказа. Мы уже разбираемся!")
		return
	}

	// 3. Отправляем подтверждение клиенту
	reply := formatConfirmation(orderCtx)
	if err := h.api.SendMessage(chatID, reply); err != nil {
		log.Printf("[process] send reply error: %v", err)
	}
}

// formatConfirmation формирует красивое подтверждение для клиента.
func formatConfirmation(ctx domain.OrderContext) string {
	switch ctx.Intent {
	case domain.IntentNewOrder:
		msg := "✅ <b>Заказ принят!</b>\n\n"
		if ctx.Items != "" {
			msg += fmt.Sprintf("🌸 <b>Состав:</b> %s\n", ctx.Items)
		}
		if ctx.Budget > 0 {
			msg += fmt.Sprintf("💰 <b>Бюджет:</b> %d ₸\n", ctx.Budget)
		}
		if ctx.DeliveryDate != "" {
			msg += fmt.Sprintf("📅 <b>Доставка:</b> %s\n", ctx.DeliveryDate)
		}
		if ctx.Notes != "" {
			msg += fmt.Sprintf("📝 <b>Заметки:</b> %s\n", ctx.Notes)
		}
		msg += "\nФлорист скоро свяжется с вами! 💐"
		return msg
	case domain.IntentQuestion:
		return "📩 Ваш вопрос записан. Флорист ответит в ближайшее время!"
	default:
		return "👋 Спасибо за сообщение! Если хотите сделать заказ — просто напишите, что вам нужно."
	}
}
