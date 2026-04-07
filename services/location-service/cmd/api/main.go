package main

import (
	"context"
	pb "echo-ride/pkg/grpc/location/v1"
	"echo-ride/services/location-service/config"
	"echo-ride/services/location-service/internal/application"
	redisInfra "echo-ride/services/location-service/internal/infrastructure/redis"
	grpcLocation "echo-ride/services/location-service/internal/presentation/grpc"
	"echo-ride/services/location-service/internal/presentation/ws"
	"echo-ride/services/location-service/pkg/logger"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("Fatal error loading config: %v\n", err)
		os.Exit(1)
	}

	log := logger.InitLogger(cfg.Server.Mode)
	defer log.Sync()
	log.Info("Starting Location Service", zap.String("mode", cfg.Server.Mode))

	redisClient, errRedis := redisInfra.NewRedisClient(cfg.Redis)
	if errRedis != nil {
		log.Fatal("Failed to create Redis client", zap.Error(errRedis))
	}

	if err := redisClient.Ping(context.Background()).Err(); err != nil {
		log.Fatal("Failed to connect to Redis", zap.Error(err))
	}
	defer redisClient.Close()

	locationRepo := redisInfra.NewRedisLocationRepo(redisClient)

	workerCtx, workerCancel := context.WithCancel(context.Background())
	defer workerCancel()

	batcher := application.NewLocationBatcher(locationRepo, log)
	go batcher.Start(workerCtx)

	locationCleaner := application.NewLocationCleaner(locationRepo, 3*time.Minute, log)
	go locationCleaner.Start(workerCtx)

	hub := ws.NewHub(log)
	go hub.Run()

	wsHandler := ws.NewHandler(hub, batcher, log)

	e := newServer(wsHandler, log)

	srvAddr := fmt.Sprintf(":%s", cfg.Server.Port)
	s := http.Server{Addr: srvAddr, Handler: e}

	go func() {
		log.Info("Server is listening", zap.String("address", srvAddr))
		if err := s.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal("Failed to start server", zap.Error(err))
		}
	}()

	grpcAddr := fmt.Sprintf(":%s", cfg.Grpc.Port)
	lis, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		log.Fatal("Failed to listen for gRPC", zap.Error(err))
	}

	grpcServer := grpc.NewServer()
	locationGrpcHandler := grpcLocation.NewLocationGrpcServer(locationRepo, log)
	pb.RegisterLocationServiceServer(grpcServer, locationGrpcHandler)
	go func() {
		log.Info("gRPC server is listening", zap.String("address", grpcAddr))
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatal("Failed to start gRPC server", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("Shutting down Location Service...")

	ctxShutdown, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelShutdown()

	if err := s.Shutdown(ctxShutdown); err != nil {
		log.Fatal("Server forced to shutdown", zap.Error(err))
	}

	log.Info("Location Service stopped gracefully")
}
