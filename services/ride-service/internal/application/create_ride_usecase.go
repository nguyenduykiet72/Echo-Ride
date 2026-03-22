package application

import (
	"context"
	"echo-ride/pkg/errs"
	"echo-ride/services/ride-service/internal/domain"

	"github.com/google/uuid"
)

type CreateRideUseCase interface {
	Excecute(ctx context.Context, req CreateRideCommand) (*domain.Ride, error)
}

type CreateRideCommand struct {
	RiderID    uuid.UUID
	PickupLat  float64
	PickupLon  float64
	DropoffLat float64
	DropoffLon float64
}

type createRideUseCase struct {
	repo domain.RideRepository
}

func NewCreateRideUseCase(repo domain.RideRepository) CreateRideUseCase {
	return &createRideUseCase{
		repo: repo,
	}
}

func (c *createRideUseCase) Excecute(ctx context.Context, req CreateRideCommand) (*domain.Ride, error) {
	if req.PickupLat == req.DropoffLat && req.PickupLon == req.DropoffLon {
		//return nil, domain.ErrInvalidRide
		return nil, errs.ErrSamePickupAndDropoff
	}

	// TODO: implement price calculation based on distance and other factors - dummy price for now
	calculatedPrice := 50000.0

	newRide := &domain.Ride{
		RiderID:    req.RiderID,
		PickupLat:  req.PickupLat,
		PickupLon:  req.PickupLon,
		DropoffLat: req.DropoffLat,
		DropoffLon: req.DropoffLon,
		Price:      calculatedPrice,
	}

	createdRide, err := c.repo.Create(ctx, newRide)
	if err != nil {
		return nil, errs.ErrBadRequest
	}

	// TODO: publish event to message broker for ride created - Kafka (RideRequestedEvent)

	return createdRide, nil
}
