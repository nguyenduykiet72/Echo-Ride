package application

import (
	"context"
	"echo-ride/pkg/errs"
	"echo-ride/services/ride-service/internal/domain"

	"github.com/google/uuid"
)

type UpdateRideUseCase interface {
	AcceptRide(ctx context.Context, rideID, driverID uuid.UUID) (*domain.Ride, error)
	UpdateStatus(ctx context.Context, rideID uuid.UUID, newStatus string) (*domain.Ride, error)
}

type udpateRideUseCase struct {
	repo domain.RideRepository
}

func NewUdpateRideUseCase(repo domain.RideRepository) UpdateRideUseCase {
	return &udpateRideUseCase{
		repo: repo,
	}
}

func (u udpateRideUseCase) AcceptRide(ctx context.Context, rideID, driverID uuid.UUID) (*domain.Ride, error) {
	ride, err := u.repo.AcceptRide(ctx, rideID, driverID)
	if err != nil {
		return nil, errs.ErrBadRequest.WithMessage("Ride is no longer available or already accepted").WithRootErr(err)
	}

	// TODO: publish event to message broker for ride accepted - Kafka (RideAcceptedEvent)
	return ride, nil
}

func (u udpateRideUseCase) UpdateStatus(ctx context.Context, rideID uuid.UUID, newStatus string) (*domain.Ride, error) {
	currentRide, err := u.repo.GetByID(ctx, rideID)
	if err != nil {
		return nil, errs.ErrNotFound.WithMessage("Ride not found").WithRootErr(err)
	}

	if !isValidStatusTransition(currentRide.Status, newStatus) {
		return nil, errs.ErrBadRequest.WithMessage(
			"Invalid state transition from " + currentRide.Status + " to " + newStatus,
		)
	}

	updatedRide, err := u.repo.UpdateStatus(ctx, rideID, newStatus)
	if err != nil {
		return nil, errs.ErrBadRequest.WithMessage("Failed to update ride status").WithRootErr(err)
	}

	return updatedRide, nil
}

func isValidStatusTransition(current, new string) bool {
	transitions := map[string]map[string]bool{
		"REQUESTED":   {"CANCELED": true},
		"ACCEPTED":    {"IN_PROGRESS": true, "CANCELED": true},
		"IN_PROGRESS": {"COMPLETED": true, "CANCELED": true},
		"COMPLETED":   {},
		"CANCELED":    {},
	}

	validNextStates, exists := transitions[current]
	if !exists {
		return false
	}

	return validNextStates[new]
}
