package application

import (
	"context"
	"echo-ride/pkg/errs"
	"echo-ride/services/ride-service/internal/domain"
	"encoding/json"
	"time"

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
	eventData := map[string]interface{}{
		"ride_id":        rideID,
		"ride_driver_id": driverID,
		"status":         "ACCEPTED",
	}

	payload := domain.RideEventPayload{
		EventID:   uuid.New().String(),
		EventType: "RIDE_ACCEPTED",
		Timestamp: time.Now().Format(time.RFC3339),
		Data:      eventData,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, errs.ErrInternal.WithMessage("Failed to marshal event payload").WithRootErr(err)
	}

	ride, err := u.repo.AcceptRide(ctx, rideID, driverID, "RIDE_ACCEPTED", payloadBytes)
	if err != nil {
		return nil, errs.ErrBadRequest.WithMessage("Ride is no longer available or already accepted").WithRootErr(err)
	}

	return ride, nil
}

func (u udpateRideUseCase) UpdateStatus(ctx context.Context, rideID uuid.UUID, newStatus string) (*domain.Ride, error) {
	currentRide, err := u.repo.GetByID(ctx, rideID)
	if err != nil {
		return nil, errs.ErrNotFound.WithMessage("Ride not found").WithRootErr(err)
	}

	if !isValidStatusTransition(currentRide.Status, newStatus) {
		return nil, errs.ErrBadRequest.WithMessage("Invalid state transition from " + currentRide.Status + " to " + newStatus)
	}

	currentRide.Status = newStatus

	payload := domain.RideEventPayload{
		EventID:   uuid.New().String(),
		EventType: "RIDE_STATUS_UPDATED",
		Timestamp: time.Now().Format(time.RFC3339),
		Data:      currentRide, // include the entire ride data in the event payload
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, errs.ErrInternal.WithMessage("Failed to marshal event payload").WithRootErr(err)
	}

	updatedRide, err := u.repo.UpdateStatus(ctx, rideID, newStatus, "RIDE_STATUS_UPDATED", payloadBytes)
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
