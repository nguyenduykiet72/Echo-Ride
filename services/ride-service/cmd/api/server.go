package main

import (
	"echo-ride/pkg/middlewares"
	"echo-ride/pkg/response"
	"echo-ride/services/ride-service/internal/application"
	"echo-ride/services/ride-service/internal/infrastructure/repository"
	rideHttp "echo-ride/services/ride-service/internal/presentation/http"
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/jackc/pgx/v5/pgxpool"
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

func newServer(dbPool *pgxpool.Pool, log *zap.Logger) *echo.Echo {
	e := echo.New()
	e.Validator = &customValidator{v: validator.New()}
	e.Use(middleware.Recover())
	e.Use(middlewares.OTelMiddleware("ride-service"))
	e.HTTPErrorHandler = response.CustomHTTPErrorHandler(log)

	e.GET("/health", func(c *echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok", "service": "ride-service"})
	})

	rideRepo := repository.NewRideRepository(dbPool)
	createRideUC := application.NewCreateRideUseCase(rideRepo)
	updateRideUC := application.NewUdpateRideUseCase(rideRepo, log)
	getRideUC := application.NewGetRideUseCase(rideRepo)
	acceptRideUC := application.NewAcceptRideUseCase(rideRepo, log)
	rideHttp.NewRideHandler(e, createRideUC, updateRideUC, getRideUC, acceptRideUC)

	return e
}
