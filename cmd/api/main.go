package main

import (
	"context"
	"database/sql"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/filipeule/integration-pipeline/internal/config"
	"github.com/filipeule/integration-pipeline/internal/queue"
	"github.com/filipeule/integration-pipeline/internal/repository"
	"github.com/filipeule/integration-pipeline/internal/worker"
)

func main() {
	os.Exit(run())
}

func run() int {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	cfg := config.Load()

	db, err := waitForDB(cfg.DSN, 30, 2*time.Second)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		return 1
	}
	defer db.Close()

	store := repository.NewMySQLStorage(db)
	slog.Info("database connected")

	rabbitConn, err := waitForRabbit(cfg.RabbitURL, 30, 2*time.Second)
	if err != nil {
		slog.Error("rabbit_connect_failed", "error", err)
		return 1
	}
	defer rabbitConn.Close()
	slog.Info("rabbitmq connected")

	pub, err := rabbitConn.NewPublisher()
	if err != nil {
		slog.Error("failed to create publisher", "error", err)
		return 1
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// start workers
	go worker.RunLeadConsumer(ctx, store, pub, rabbitConn)
	go worker.RunSMSDistributor(ctx, store, pub, rabbitConn, cfg.WebhookSiteURL)

	app := application{
		port:      config.GetEnv("PORT", "8080"),
		config:    cfg,
		storage:   store,
		publisher: pub,
	}

	mux := app.mount()

	srv := &http.Server{
		Addr:         ":" + app.port,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		slog.Info("server start listening", "port", app.port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("error listening to server", "error", err)
		}
	}()

	<-ctx.Done()

	slog.Info("server shutting down")
	shutCtx, shutCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutCancel()

	srv.Shutdown(shutCtx)
	return 0
}

func waitForDB(dsn string, attempts int, delay time.Duration) (*sql.DB, error) {
	var err error
	for i := range attempts {
		db, e := repository.NewDB(dsn)
		if e == nil {
			return db, nil
		}

		err = e
		slog.Warn("database not ready", "attempt", i+1, "error", e)
		time.Sleep(delay)
	}

	return nil, err
}

func waitForRabbit(url string, attempts int, delay time.Duration) (*queue.Connection, error) {
	var err error
	for i := range attempts {
		c, e := queue.NewConnection(url)
		if e == nil {
			return c, nil
		}

		err = e
		slog.Warn("rabbitmq not ready", "attempt", i+1, "error", e)
		time.Sleep(delay)
	}

	return nil, err
}
