package application

import (
	"context"
	"echo-ride/pkg/errs"
	"echo-ride/services/ride-service/internal/domain"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

type UpdateRideUseCase interface {
	AcceptRide(ctx context.Context, rideID, driverID uuid.UUID) (*domain.Ride, error)
	UpdateStatus(ctx context.Context, rideID uuid.UUID, newStatus string) (*domain.Ride, error)
	Execute(ctx context.Context, rideID uuid.UUID, newStatus domain.RideStatus) error
}

type udpateRideUseCase struct {
	repo   domain.RideRepository
	logger *zap.Logger
	tracer trace.Tracer
}

func NewUdpateRideUseCase(repo domain.RideRepository, logger *zap.Logger) UpdateRideUseCase {
	return &udpateRideUseCase{
		repo:   repo,
		logger: logger,
		tracer: otel.Tracer("ride-service-uc"),
	}
}

func (u *udpateRideUseCase) AcceptRide(ctx context.Context, rideID, driverID uuid.UUID) (*domain.Ride, error) {
	eventData := map[string]interface{}{
		"ride_id":        rideID.String(),
		"ride_driver_id": driverID.String(),
		"status":         "ACCEPTED",
	}

	payload := domain.RideEventPayload{
		EventID:   uuid.New().String(),
		EventType: domain.EventTypeRideRequested,
		Timestamp: time.Now().Format(time.RFC3339),
		Data:      eventData,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, errs.ErrInternal.WithMessage("Failed to marshal event payload").WithRootErr(err)
	}

	ride, err := u.repo.AcceptRide(ctx, rideID, driverID, string(domain.EventTypeRideAccepted), payloadBytes)
	if err != nil {
		return nil, errs.ErrBadRequest.WithMessage("Ride is no longer available or already accepted").WithRootErr(err)
	}

	return ride, nil
}

func (u *udpateRideUseCase) UpdateStatus(ctx context.Context, rideID uuid.UUID, newStatus string) (*domain.Ride, error) {
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

func (u *udpateRideUseCase) Execute(ctx context.Context, rideID uuid.UUID, newStatus domain.RideStatus) error {
	ctx, span := u.tracer.Start(ctx, "UseCase.UpdateRideStatus")
	defer span.End()

	ride, err := u.repo.GetByID(ctx, rideID)
	if err != nil {
		span.RecordError(err)
		return errs.ErrNotFound.WithMessage("Ride not found").WithRootErr(err)
	}
	if ride == nil {
		u.logger.Warn("Ride not found", zap.String("Ride ID", rideID.String()))
		return nil
	}

	statusStr := string(newStatus)

	_, err = u.repo.UpdateStatus(ctx, rideID, statusStr, "", nil)
	if err != nil {
		span.RecordError(err)
		return errs.ErrInternal.WithMessage("Failed to update ride status").WithRootErr(err)
	}

	u.logger.Info("Ride status updated successfully",
		zap.String("ride_id", rideID.String()),
		zap.String("new_status", string(newStatus)))

	return nil
}
