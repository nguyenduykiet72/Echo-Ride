package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type AccountRole string
type AccountStatus string

const (
	RoleRider  AccountRole = "RIDER"
	RoleDriver AccountRole = "DRIVER"
	RoleAdmin  AccountRole = "ADMIN"

	StatusActive    AccountStatus = "ACTIVE"
	StatusSuspended AccountStatus = "SUSPENDED"
	StatusBanned    AccountStatus = "BANNED"
)

type Identity struct {
	ID           uuid.UUID     `json:"id"`
	Email        string        `json:"email"`
	Phone        string        `json:"phone"`
	PasswordHash string        `json:"-"`
	Role         AccountRole   `json:"role"`
	Status       AccountStatus `json:"status"`
	CreatedAt    time.Time     `json:"created_at"`
	UpdatedAt    time.Time     `json:"updated_at"`
}

type IdentityRepository interface {
	Create(ctx context.Context, identity *Identity) (*Identity, error)
	GetByEmail(ctx context.Context, email string) (*Identity, error)
	GetByPhone(ctx context.Context, phone string) (*Identity, error)
}
