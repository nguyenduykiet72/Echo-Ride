package main

import (
	"context"
	pb "echo-ride/pkg/grpc/user/v1"
	"echo-ride/pkg/tracing"
	"echo-ride/services/user-service/config"
	"echo-ride/services/user-service/internal/application"
	db "echo-ride/services/user-service/internal/infrastructure"
	"echo-ride/services/user-service/internal/infrastructure/kafka"
	"echo-ride/services/user-service/internal/infrastructure/repository"
	grpcUser "echo-ride/services/user-service/internal/presentation/grpc"
	"echo-ride/services/user-service/pkg/logger"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		os.Exit(1)
	}

	log := logger.InitLogger(cfg.Server.Mode)
	defer log.Sync()
	log.Info("Starting User Service", zap.String("mode", cfg.Server.Mode))

	tp, err := tracing.InitTracer("user-service", cfg.Jaeger.AgentHost+":"+fmt.Sprint(cfg.Jaeger.AgentPort))
	if err != nil {
		log.Fatal("Failed to init tracer", zap.Error(err))
	}
	defer func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			log.Fatal("Failed to shutdown tracer", zap.Error(err))
		}
	}()

	dbCtx, cancelDB := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelDB()
	dbPool, err := db.NewPostgresPool(dbCtx, cfg.Database)
	if err != nil {
		log.Fatal("Failed to connect to database", zap.Error(err))
	}
	defer dbPool.Close()
	log.Info("Database connection successful")

	userRepo := repository.NewUserRepository(dbPool)
	getUserUC := application.NewGetUserUseCase(userRepo, log)
	updateProfileUC := application.NewUpdateProfileUseCase(userRepo, log)
	updateRoleUC := application.NewUpdateRoleUseCase(userRepo, log)
	updateStatusUC := application.NewUpdateStatusUseCase(userRepo, log)
	upsertUserUC := application.NewUpsertUserUseCase(userRepo, log)

	workerCtx, workerCancel := context.WithCancel(context.Background())
	defer workerCancel()

	consumer := kafka.NewUserConsumer(cfg.Kafka, cfg.Kafka.IdentityTopic, upsertUserUC, log)
	go consumer.Start(workerCtx)
	defer consumer.Close()

	srv := newServer(ServerConfig{
		DBPool:          dbPool,
		Config:          cfg,
		Logger:          log,
		GetUserUC:       getUserUC,
		UpdateProfileUC: updateProfileUC,
		UpdateRoleUC:    updateRoleUC,
		UpdateStatusUC:  updateStatusUC,
	})

	srvAddr := fmt.Sprintf(":%s", cfg.Server.Port)
	httpSrv := http.Server{Addr: srvAddr, Handler: srv}

	go func() {
		log.Info("HTTP server is listening", zap.String("address", srvAddr))
		if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal("Failed to start HTTP server", zap.Error(err))
		}
	}()

	grpcAddr := fmt.Sprintf(":%s", cfg.GRPC.Port)
	lis, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		log.Fatal("Failed to listen for gRPC", zap.Error(err))
	}
	grpcServer := grpc.NewServer(grpc.StatsHandler(otelgrpc.NewServerHandler()))
	pb.RegisterUserServiceServer(grpcServer, grpcUser.NewUserGrpcServer(getUserUC, log))
	go func() {
		log.Info("gRPC server is listening", zap.String("address", grpcAddr))
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatal("Failed to start gRPC server", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info("Shutting down User Service...")

	ctxShutdown, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelShutdown()
	if err := httpSrv.Shutdown(ctxShutdown); err != nil {
		log.Error("HTTP server forced to shutdown", zap.Error(err))
	}
	grpcServer.GracefulStop()

	log.Info("User Service stopped gracefully")
}
