package application

import (
	"context"
	"echo-ride/pkg/errs"
	"echo-ride/services/auth-service/internal/domain"
	"echo-ride/services/auth-service/pkg/hash"

	"go.uber.org/zap"
)

type RegisterRequest struct {
	Email    string             `json:"email"`
	Phone    string             `json:"phone"`
	Password string             `json:"password"`
	Role     domain.AccountRole `json:"role"`
}

type RegisterUseCase interface {
	Execute(ctx context.Context, req RegisterRequest) (*domain.Identity, error)
}

type registerUC struct {
	repo   domain.IdentityRepository
	logger *zap.Logger
}

func NewRegisterUseCase(repo domain.IdentityRepository, logger *zap.Logger) RegisterUseCase {
	return &registerUC{
		repo:   repo,
		logger: logger,
	}
}

func (r *registerUC) Execute(ctx context.Context, req RegisterRequest) (*domain.Identity, error) {
	existingPhone, _ := r.repo.GetByPhone(ctx, req.Phone)
	if existingPhone != nil {
		return nil, errs.ErrConflict.WithMessage("Phone number already exists")
	}

	existingEmail, _ := r.repo.GetByEmail(ctx, req.Email)
	if existingEmail != nil {
		return nil, errs.ErrConflict.WithMessage("Email already exists")
	}

	hashedPassword, err := hash.HashPassword(req.Password)
	if err != nil {
		r.logger.Error("failed to hash password", zap.Error(err))
		return nil, errs.ErrInternal.WithMessage("Failed to hash password")
	}

	newIdentity := &domain.Identity{
		Email:        req.Email,
		Phone:        req.Phone,
		PasswordHash: hashedPassword,
		Role:         req.Role,
	}

	createdIdentity, err := r.repo.Create(ctx, newIdentity)
	if err != nil {
		r.logger.Error("failed to create identity", zap.Error(err))
		return nil, errs.ErrInternal.WithMessage("Failed to create identity")
	}

	return createdIdentity, nil
}
