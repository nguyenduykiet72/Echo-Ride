package main

import (
	"context"
	"echo-ride/pkg/response"
	"echo-ride/services/ride-service/config"
	"echo-ride/services/ride-service/internal/application"
	"echo-ride/services/ride-service/internal/infrastructure/db"
	"echo-ride/services/ride-service/internal/infrastructure/repository"
	rideHttp "echo-ride/services/ride-service/internal/presentation/http"
	"echo-ride/services/ride-service/pkg/logger"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
	"go.uber.org/zap"
)

type customValidator struct {
	v *validator.Validate
}

func (cv *customValidator) Validate(i interface{}) error {
	return cv.v.Struct(i)
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		os.Exit(1)
	}

	log := logger.InitLogger(cfg.Server.Mode)
	defer log.Sync()
	log.Info("Starting Ride Service", zap.String("mode", cfg.Server.Mode))

	ctx, cancelDB := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelDB()

	dbPool, err := db.NewPostgresPool(ctx, cfg.Database)
	if err != nil {
		log.Fatal("Failed to connect to database", zap.Error(err))
	}
	defer dbPool.Close()
	log.Info("Database connection successful")

	e := echo.New()
	e.Validator = &customValidator{v: validator.New()}
	e.Use(middleware.Recover(), middleware.RequestID())
	e.HTTPErrorHandler = response.CustomHTTPErrorHandler(log)

	e.GET("/health", func(c *echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok", "service": "ride-service"})
	})

	rideRepo := repository.NewRideRepository(dbPool)
	createRideUC := application.NewCreateRideUseCase(rideRepo)
	updateRideUC := application.NewUdpateRideUseCase(rideRepo)
	getRideUC := application.NewGetRideUseCase(rideRepo)
	rideHttp.NewRideHander(e, createRideUC, updateRideUC, getRideUC)

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
	<-quit // Wait for shutdown signal
	log.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := s.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown", zap.Error(err))
	}

	log.Info("Server exiting")
}
