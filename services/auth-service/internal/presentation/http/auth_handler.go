package http

import (
	"echo-ride/pkg/errs"
	"echo-ride/pkg/response"
	"echo-ride/services/auth-service/internal/application"
	"echo-ride/services/auth-service/internal/domain"
	"net/http"

	"github.com/labstack/echo/v5"
)

type AuthHandler struct {
	registerUC application.RegisterUseCase
	loginUC    application.LoginUseCase
}

func NewAuthHandler(e *echo.Echo, registerUC application.RegisterUseCase, loginUC application.LoginUseCase) {
	handler := &AuthHandler{
		registerUC: registerUC,
		loginUC:    loginUC,
	}

	v1 := e.Group("/api/v1/auth")
	v1.POST("/register", handler.Register)
	v1.POST("/login", handler.Login)
}

func (h *AuthHandler) Register(ctx *echo.Context) error {
	var req registerRequest
	if err := ctx.Bind(&req); err != nil {
		return errs.ErrBadRequest.WithMessage("Invalid request body").WithRootErr(err)
	}
	if err := ctx.Validate(&req); err != nil {
		return errs.ErrBadRequest.WithMessage("Validation failed: " + err.Error())
	}

	ucReq := application.RegisterRequest{
		Email:    req.Email,
		Phone:    req.Phone,
		Password: req.Password,
		Role:     domain.AccountRole(req.Role),
	}

	identity, err := h.registerUC.Execute(ctx.Request().Context(), ucReq)
	if err != nil {
		return err
	}

	return response.WriteSuccess(ctx, http.StatusCreated, identity, "Account created successfully")
}

func (h *AuthHandler) Login(ctx *echo.Context) error {
	var req loginRequest
	if err := ctx.Bind(&req); err != nil {
		return errs.ErrBadRequest.WithMessage("Invalid request body")
	}
	if err := ctx.Validate(&req); err != nil {
		return errs.ErrBadRequest.WithMessage("Validation failed: " + err.Error())
	}

	ucReq := application.LoginRequest{
		Phone:    req.Phone,
		Password: req.Password,
	}

	res, err := h.loginUC.Execute(ctx.Request().Context(), ucReq)
	if err != nil {
		return err
	}

	return response.WriteSuccess(ctx, http.StatusOK, res, "Login successful")
}
