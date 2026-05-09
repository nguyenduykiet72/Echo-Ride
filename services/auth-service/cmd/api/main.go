package main

import (
	"context"
	"echo-ride/pkg/tracing"
	"echo-ride/services/auth-service/config"
	"echo-ride/services/auth-service/internal/application"
	db "echo-ride/services/auth-service/internal/infrastructure"
	grpcclient "echo-ride/services/auth-service/internal/infrastructure/grpc-client"
	"echo-ride/services/auth-service/internal/infrastructure/kafka"
	redisInfra "echo-ride/services/auth-service/internal/infrastructure/redis"
	"echo-ride/services/auth-service/internal/infrastructure/repository"
	authMiddleware "echo-ride/services/auth-service/internal/presentation/middleware"
	"echo-ride/services/auth-service/pkg/jwt"
	"echo-ride/services/auth-service/pkg/logger"
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
	log.Info("Starting Auth Service", zap.String("mode", cfg.Server.Mode))

	tp, err := tracing.InitTracer("auth-service", cfg.Jaeger.AgentHost+":"+fmt.Sprint(cfg.Jaeger.AgentPort))
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

	redisClient := redisInfra.NewRedisClient(cfg.Redis.Addr, cfg.Redis.Password, cfg.Redis.DB)
	if err := redisClient.Ping(context.Background()).Err(); err != nil {
		log.Fatal("Failed to connect to Redis", zap.Error(err))
	}
	defer redisClient.Close()
	blacklist := redisInfra.NewBlacklist(redisClient)

	userClient, err := grpcclient.NewUserServiceClient(cfg.UserService.GRPCAddr)
	if err != nil {
		log.Fatal("Failed to create user-service gRPC client", zap.Error(err))
	}
	defer userClient.Close()

	identityRepo := repository.NewIdentityRepository(dbPool)
	refreshRepo := repository.NewRefreshTokenRepository(dbPool)
	tokenMaker := jwt.NewTokenMaker(cfg.JWT.SecretKey)

	accessTTL := time.Duration(cfg.JWT.AccessTokenTTLMin) * time.Minute
	refreshTTL := time.Duration(cfg.JWT.RefreshTokenTTLHours) * time.Hour

	registerUC := application.NewRegisterUseCase(identityRepo, log)
	loginUC := application.NewLoginUseCase(application.LoginUCDeps{
		IdentityRepo:    identityRepo,
		RefreshRepo:     refreshRepo,
		UserClient:      userClient,
		TokenMaker:      tokenMaker,
		AccessTokenTTL:  accessTTL,
		RefreshTokenTTL: refreshTTL,
		Logger:          log,
	})
	refreshUC := application.NewRefreshUseCase(application.RefreshUCDeps{
		RefreshRepo:     refreshRepo,
		UserClient:      userClient,
		TokenMaker:      tokenMaker,
		AccessTokenTTL:  accessTTL,
		RefreshTokenTTL: refreshTTL,
		Logger:          log,
	})
	logoutUC := application.NewLogoutUseCase(refreshRepo, blacklist, log)
	handleUserUC := application.NewHandleUserEventUseCase(refreshRepo, log)

	jwtAuth := authMiddleware.JWTAuth(tokenMaker, blacklist)

	workerCtx, workerCancel := context.WithCancel(context.Background())
	defer workerCancel()

	consumer := kafka.NewAuthConsumer(cfg.Kafka, cfg.Kafka.UserTopic, handleUserUC, log)
	go consumer.Start(workerCtx)
	defer consumer.Close()

	srv := newServer(ServerConfig{
		DBPool:     dbPool,
		Config:     cfg,
		Logger:     log,
		RegisterUC: registerUC,
		LoginUC:    loginUC,
		RefreshUC:  refreshUC,
		LogoutUC:   logoutUC,
		JWTAuth:    jwtAuth,
	})

	srvAddr := fmt.Sprintf(":%s", cfg.Server.Port)
	httpSrv := http.Server{Addr: srvAddr, Handler: srv}

	go func() {
		log.Info("Server is listening", zap.String("address", srvAddr))
		if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal("Failed to start server", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info("Shutting down server...")

	ctxShutdown, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelShutdown()
	if err := httpSrv.Shutdown(ctxShutdown); err != nil {
		log.Error("Server forced to shutdown", zap.Error(err))
	}
	log.Info("Server exiting")
}
