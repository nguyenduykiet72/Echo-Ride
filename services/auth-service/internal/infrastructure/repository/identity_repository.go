package repository

import (
	"context"
	"echo-ride/services/auth-service/internal/domain"
	"echo-ride/services/auth-service/internal/infrastructure/db/dbgen"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type IdentityRepositoryImpl struct {
	db *pgxpool.Pool
	q  *dbgen.Queries
}

func NewIdentityRepository(db *pgxpool.Pool) domain.IdentityRepository {
	return &IdentityRepositoryImpl{
		db: db,
		q:  dbgen.New(db),
	}
}

func dbIdentityToDomain(dbId dbgen.TIdentity) *domain.Identity {
	return &domain.Identity{
		ID:           uuid.UUID(dbId.IdentityID.Bytes),
		Email:        dbId.IdentityEmail,
		Phone:        dbId.IdentityPhone,
		PasswordHash: dbId.IdentityPasswordHash,
		Role:         domain.AccountRole(dbId.IdentityRole),
		Status:       domain.AccountStatus(dbId.IdentityStatus),
		CreatedAt:    dbId.IdentityCreatedAt.Time,
		UpdatedAt:    dbId.IdentityUpdatedAt.Time,
	}
}

func (i *IdentityRepositoryImpl) Create(ctx context.Context, identity *domain.Identity) (*domain.Identity, error) {
	params := dbgen.CreateIdentityParams{
		IdentityEmail:        identity.Email,
		IdentityPhone:        identity.Phone,
		IdentityPasswordHash: identity.PasswordHash,
		IdentityRole:         dbgen.AccountRole(identity.Role),
	}

	dbId, err := i.q.CreateIdentity(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to create identity: %w", err)
	}

	return dbIdentityToDomain(dbId), nil
}

func (i *IdentityRepositoryImpl) GetByEmail(ctx context.Context, email string) (*domain.Identity, error) {
	dbId, err := i.q.GetIdentityByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get identity by email: %w", err)
	}
	return dbIdentityToDomain(dbId), nil
}

func (i *IdentityRepositoryImpl) GetByPhone(ctx context.Context, phone string) (*domain.Identity, error) {
	dbId, err := i.q.GetIdentityByPhone(ctx, phone)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get identity by phone: %w", err)
	}
	return dbIdentityToDomain(dbId), nil
}
