package main

import (
	"context"
	"echo-ride/services/matching-service/config"
	"echo-ride/services/matching-service/internal/application"
	grpcClient "echo-ride/services/matching-service/internal/infrastructure/grpc-client"
	"echo-ride/services/matching-service/internal/infrastructure/kafka"
	redisInfra "echo-ride/services/matching-service/internal/infrastructure/redis"
	"echo-ride/services/matching-service/pkg/logger"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/labstack/echo/v5"
	"go.uber.org/zap"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("Fatal error loading config: %v\n", err)
		os.Exit(1)
	}

	log := logger.InitLogger(cfg.Server.Mode)
	defer log.Sync()
	log.Info("Starting Matching Service", zap.String("mode", cfg.Server.Mode))

	redisClient, errRedis := redisInfra.NewRedisClient(cfg.Redis)
	if errRedis != nil {
		log.Fatal("Failed to create Redis client", zap.Error(errRedis))
	}

	if err := redisClient.Ping(context.Background()).Err(); err != nil {
		log.Fatal("Failed to connect to Redis", zap.Error(err))
	}
	defer redisClient.Close()

	dispatchRepo := redisInfra.NewRedisDispatchRepo(redisClient)

	workerCtx, cancelWorkers := context.WithCancel(context.Background())
	defer cancelWorkers()

	locationClient, err := grpcClient.NewLocationGrpcClient(cfg.Dependencies.LocationGrpcUrl, log)
	if err != nil {
		log.Fatal("Failed to initialize Location gRPC Client", zap.Error(err))
	}
	log.Info("Connected to Location Service gRPC")

	processRideUC := application.NewProcessRideRequestUseCase(locationClient, dispatchRepo, log)

	timeoutWatcher := application.NewTimeoutWatcher(dispatchRepo, log)
	go timeoutWatcher.Start(workerCtx)

	rideConsumer := kafka.NewRideConsumer(cfg.Kafka, processRideUC, dispatchRepo, log)
	go rideConsumer.Start(workerCtx)

	e := echo.New()
	e.GET("/health", func(c *echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"service": "matching-service", "status": "ok"})
	})

	srvAddr := fmt.Sprintf(":%s", cfg.Server.Port)
	httpServer := &http.Server{Addr: srvAddr, Handler: e}

	go func() {
		log.Info("HTTP Server listening for health checks", zap.String("port", cfg.Server.Port))
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal("Failed to start HTTP server", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("Shutting down Matching Service...")

	cancelWorkers()
	if err := rideConsumer.Close(); err != nil {
		log.Error("Failed to close Kafka consumer", zap.Error(err))
	}

	ctxShutdown, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelShutdown()
	if err := httpServer.Shutdown(ctxShutdown); err != nil {
		log.Fatal("HTTP Server forced to shutdown", zap.Error(err))
	}

	log.Info("Matching Service stopped gracefully")
}
