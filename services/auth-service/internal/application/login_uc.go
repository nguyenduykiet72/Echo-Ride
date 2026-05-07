package application

import (
	"context"
	"echo-ride/pkg/errs"
	"echo-ride/services/auth-service/internal/domain"
	"echo-ride/services/auth-service/pkg/hash"
	"echo-ride/services/auth-service/pkg/jwt"
	"time"

	"go.uber.org/zap"
)

type LoginRequest struct {
	Phone    string `json:"phone"`
	Password string `json:"password"`
}

type LoginResponse struct {
	AccessToken string `json:"accessToken"`
}

type LoginUseCase interface {
	Execute(ctx context.Context, req LoginRequest) (*LoginResponse, error)
}

type loginUC struct {
	repo       domain.IdentityRepository
	tokenMaker *jwt.TokenMaker
	logger     *zap.Logger
}

func NewLoginUseCase(repo domain.IdentityRepository, tokenMaker *jwt.TokenMaker, logger *zap.Logger) LoginUseCase {
	return &loginUC{
		repo:       repo,
		tokenMaker: tokenMaker,
		logger:     logger,
	}
}

func (l *loginUC) Execute(ctx context.Context, req LoginRequest) (*LoginResponse, error) {
	identity, err := l.repo.GetByPhone(ctx, req.Phone)
	if err != nil {
		return nil, errs.ErrInternal.WithMessage("Database error")
	}
	if identity == nil {
		return nil, errs.ErrUnauthorized.WithMessage("Invalid phone number or password")
	}

	if identity.Status != domain.StatusActive {
		return nil, errs.ErrForbidden.WithMessage("Account is locked or suspended")
	}

	err = hash.CheckPasswordHash(req.Password, identity.PasswordHash)
	if err != nil {
		return nil, errs.ErrUnauthorized.WithMessage("Invalid phone number or password")
	}

	token, err := l.tokenMaker.GenerateToken(identity.ID, string(identity.Role), string(identity.Status), 24*time.Hour)
	if err != nil {
		l.logger.Error("failed to generate token", zap.Error(err))
		return nil, errs.ErrInternal.WithMessage("Failed to generate access token")
	}

	return &LoginResponse{
		AccessToken: token,
	}, nil
}
