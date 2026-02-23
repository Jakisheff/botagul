package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/florist-agent/internal/config"
	"github.com/florist-agent/internal/llm"
	"github.com/florist-agent/internal/repository"
	"github.com/florist-agent/internal/telegram"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func main() {
	// Загружаем .env (если есть — используем, если нет — не падаем)
	if err := godotenv.Load(); err != nil {
		log.Println("[init] .env file not found, using system env vars")
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("[init] config error: %v", err)
	}

	// ---------- PostgreSQL ----------
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("[init] database connection error: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("[init] database ping failed: %v", err)
	}
	log.Println("[init] ✅ connected to PostgreSQL")

	// Авто-миграция
	repo := repository.NewPostgres(pool)
	if err := repo.Migrate(ctx); err != nil {
		log.Fatalf("[init] migration error: %v", err)
	}
	log.Println("[init] ✅ migration applied")

	// ---------- LLM ----------
	llmClient := llm.NewClient(cfg.GeminiAPIKey, cfg.GeminiModel)
	extractor := llm.NewExtractor(llmClient)
	log.Printf("[init] ✅ LLM configured (model: %s)", cfg.GeminiModel)

	// ---------- Telegram ----------
	tgAPI := telegram.NewAPI(cfg.TelegramToken)
	handler := telegram.NewHandler(tgAPI, extractor, repo, cfg.AllowedChatIDs)

	// ---------- HTTP Server ----------
	mux := http.NewServeMux()
	mux.Handle("/webhook", handler)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "ok")
	})

	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// ---------- Graceful Shutdown ----------
	go func() {
		log.Printf("[server] 🚀 botagul listening on :%s", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("[server] listen error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	log.Printf("[server] received signal %v, shutting down...", sig)

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("[server] shutdown error: %v", err)
	}

	log.Println("[server] 👋 botagul stopped gracefully")
}
