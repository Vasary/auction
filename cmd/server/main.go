package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"time"

	"go.uber.org/zap"

	"auction-core/internal/amqp"
	"auction-core/internal/auction"
	"auction-core/internal/auction/repository/postgres"
	"auction-core/internal/db"
	httpHandler "auction-core/internal/http"
	"auction-core/internal/logger"
	"auction-core/internal/metrics"
	"auction-core/internal/scheduler"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "application failed: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	log, err := logger.New()
	if err != nil {
		return fmt.Errorf("logger init: %w", err)
	}
	defer log.Sync()

	configureTime(log)

	dbDSN := os.Getenv("DATABASE_URL")
	if dbDSN == "" {
		return fmt.Errorf("DATABASE_URL is not set")
	}
	dbPool := db.New(ctx, log, dbDSN)
	defer dbPool.Close()

	amqpURL := os.Getenv("AMQP_URL")
	eventPublisher, amqpCleanup, err := amqp.New(ctx, log, amqpURL)
	if err != nil {
		return fmt.Errorf("amqp init: %w", err)
	}
	defer amqpCleanup()

	auctionRepo := postgres.NewAuctionPostgresRepository(dbPool)
	bidRepo := postgres.NewPostgresBidRepository(dbPool)
	participantRepo := postgres.NewPostgresParticipantRepository(dbPool)

	manager := auction.NewManager(auctionRepo, eventPublisher, log)

	if err := manager.RecoverSessions(ctx, auctionRepo, bidRepo, 5*time.Minute); err != nil {
		metrics.RecoveryTotal.WithLabelValues("error").Inc()
		log.Error("auction recovery failed", zap.Error(err))
	} else {
		metrics.RecoveryTotal.WithLabelValues("ok").Inc()
	}

	sched := &scheduler.Scheduler{
		Manager:    manager,
		Repository: auctionRepo,
		Interval:   1 * time.Minute,
		Logger:     log,
	}
	go sched.Start(ctx)

	handler := httpHandler.NewHandler(manager, participantRepo, auctionRepo, bidRepo, log)

	port := getPort()
	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      handler.Routes(),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Info("server started", zap.String("port", port))
		log.Info("websocket server ready to accept connections")
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("server failed", zap.Error(err))
		}
	}()

	<-ctx.Done()
	log.Info("shutdown signal received")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("server shutdown: %w", err)
	}

	log.Info("server gracefully stopped")
	return nil
}

func getPort() string {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8082"
	}
	return port
}

func configureTime(log *zap.Logger) {
	if os.Getenv("TZ") == "" {
		time.Local = time.UTC
	}

	locName := "UTC"
	if time.Local != time.UTC && time.Local != nil {
		if l := time.Local.String(); l != "" {
			locName = l
		} else {
			locName = "custom"
		}
	}

	log.Info("time configuration",
		zap.String("time_zone", locName),
		zap.String("tz_env", os.Getenv("TZ")),
	)
}
