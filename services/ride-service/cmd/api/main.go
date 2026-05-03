package main

import (
	"context"
	"echo-ride/pkg/tracing"
	"echo-ride/services/ride-service/config"
	"echo-ride/services/ride-service/internal/infrastructure/db"
	"echo-ride/services/ride-service/internal/infrastructure/kafka"
	"echo-ride/services/ride-service/pkg/logger"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		os.Exit(1)
	}

	log := logger.InitLogger(cfg.Server.Mode)
	defer log.Sync()
	log.Info("Starting Ride Service", zap.String("mode", cfg.Server.Mode))

	tp, err := tracing.InitTracer("ride-service", cfg.Jaeger.AgentHost+":"+fmt.Sprint(cfg.Jaeger.AgentPort))
	if err != nil {
		log.Fatal("Failed to init tracer", zap.Error(err))
	}

	defer func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			log.Fatal("Failed to shutdown tracer", zap.Error(err))
		}
	}()

	ctx, cancelDB := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelDB()

	dbPool, err := db.NewPostgresPool(ctx, cfg.Database)
	if err != nil {
		log.Fatal("Failed to connect to database", zap.Error(err))
	}
	defer dbPool.Close()
	log.Info("Database connection successful")

	useCases := buildUseCases(dbPool, log)

	consumerCtx, cancelConsumer := context.WithCancel(context.Background())
	defer cancelConsumer()

	rideConsumer := kafka.NewRideConsumer(cfg.Kafka, useCases.updateRide, log)
	go rideConsumer.Start(consumerCtx)

	e := newServer(useCases, log)

	srvAddr := fmt.Sprintf(":%s", cfg.Server.Port)
	s := http.Server{Addr: srvAddr, Handler: e}

	go func() {
		log.Info("Server is listening", zap.String("address", srvAddr))
		if err := s.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal("Failed to start server", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info("Shutting down server...")

	cancelConsumer()
	if err := rideConsumer.Close(); err != nil {
		log.Error("Failed to close kafka consumer", zap.Error(err))
	}

	ctxShutdown, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelShutdown()

	if err := s.Shutdown(ctxShutdown); err != nil {
		log.Fatal("Server forced to shutdown", zap.Error(err))
	}

	log.Info("Server exiting")
}
