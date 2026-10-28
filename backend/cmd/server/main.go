package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	// 1. Настройка логирования (slog)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	// 2. Загрузка конфигурации
	port := os.Getenv("PORT")
	if port == "" {
		port = "5000"
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		slog.Error("DATABASE_URL is required but not set")
		os.Exit(1)
	}

	// 3. Подключение к базе данных (с повторными попытками)
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		slog.Error("failed to create database pool", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	// Ждем базу данных (до 10 секунд), чтобы избежать падения при старте Docker
	var pingErr error
	for i := 0; i < 10; i++ {
		pingErr = pool.Ping(ctx)
		if pingErr == nil {
			break
		}
		slog.Warn("database not ready, retrying...", "attempt", i+1, "error", pingErr)
		time.Sleep(1 * time.Second)
	}
	if pingErr != nil {
		slog.Error("database ping failed after retries", "error", pingErr)
		os.Exit(1)
	}
	slog.Info("Successfully connected to database")

	// 4. Настройка HTTP-роутера
	mux := http.NewServeMux()

	// Health-check эндпоинт (полезен для Docker/Kubernetes)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		if err := pool.Ping(r.Context()); err != nil {
			http.Error(w, "database unhealthy", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Пример базового эндпоинта (заглушка)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"message": "Backend is running!"}`))
	})

	// 5. Настройка и запуск HTTP-сервера
	srv := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	// Запускаем сервер в отдельной горутине, чтобы он не блокировал основной поток
	go func() {
		slog.Info("starting server", "port", port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server failed to start", "error", err)
			os.Exit(1)
		}
	}()

	// 6. Graceful Shutdown (ожидание сигналов завершения)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit // Блокируемся, пока не придет сигнал

	slog.Info("shutting down server...")

	// Даем 10 секунд на завершение текущих запросов
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("server forced to shutdown", "error", err)
	}

	slog.Info("server stopped gracefully")
}
