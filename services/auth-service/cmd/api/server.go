package main

import (
	"echo-ride/pkg/middlewares"
	"echo-ride/pkg/response"
	"echo-ride/services/auth-service/config"
	"echo-ride/services/auth-service/internal/application"
	"echo-ride/services/auth-service/internal/infrastructure/repository"
	authHttp "echo-ride/services/auth-service/internal/presentation/http"
	"echo-ride/services/auth-service/pkg/jwt"
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

type ServerConfig struct {
	DBPool *pgxpool.Pool
	Config *config.Config
	Logger *zap.Logger
}

func newServer(config ServerConfig) *echo.Echo {
	e := echo.New()
	e.Validator = &customValidator{v: validator.New()}
	e.Use(middleware.Recover())
	e.Use(middlewares.OTelMiddleware("auth-service"))
	e.HTTPErrorHandler = response.CustomHTTPErrorHandler(config.Logger)

	e.GET("/health", func(c *echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok", "service": "auth-service"})
	})

	identityRepo := repository.NewIdentityRepository(config.DBPool)
	tokenMaker := jwt.NewTokenMaker(config.Config.JWT.SecretKey)
	loginUC := application.NewLoginUseCase(identityRepo, tokenMaker, config.Logger)
	registerUC := application.NewRegisterUseCase(identityRepo, config.Logger)
	authHttp.NewAuthHandler(e, registerUC, loginUC)

	return e
}
