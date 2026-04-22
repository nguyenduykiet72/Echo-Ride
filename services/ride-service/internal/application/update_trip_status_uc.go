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

type UpdateTripStatusUseCase interface {
	Execute(ctx context.Context, rideID, driverID uuid.UUID, newStatus domain.RideStatus) (*domain.Ride, error)
}

type updateTripStatusUC struct {
	repo   domain.RideRepository
	logger *zap.Logger
	tracer trace.Tracer
}

func NewUpdateTripStatusUseCase(repo domain.RideRepository, logger *zap.Logger) UpdateTripStatusUseCase {
	return &updateTripStatusUC{
		repo:   repo,
		logger: logger,
		tracer: otel.Tracer("ride-service-uc"),
	}
}

func (u *updateTripStatusUC) Execute(ctx context.Context, rideID, driverID uuid.UUID, newStatus domain.RideStatus) (*domain.Ride, error) {
	ctx, span := u.tracer.Start(ctx, "UseCase.UpdateTripStatus")
	defer span.End()

	var expectedOldStatus domain.RideStatus
	switch newStatus {
	case domain.RideStatusInProgress:
		expectedOldStatus = domain.RideStatusAccepted
	case domain.RideStatusCompleted:
		expectedOldStatus = domain.RideStatusInProgress
	default:
		return nil, errs.ErrBadRequest.WithMessage("Invalid status transition requested")
	}

	rideInfo, err := u.repo.GetByID(ctx, rideID)
	if err != nil || rideInfo == nil {
		return nil, errs.ErrNotFound.WithMessage("Ride not found")
	}

	payload := domain.RideEventPayload{
		EventID:   uuid.New().String(),
		EventType: domain.EventType(newStatus),
		Timestamp: time.Now().Format(time.RFC3339),
		Data: map[string]interface{}{
			"ride_id":   rideID.String(),
			"rider_id":  rideInfo.RiderID.String(), // Assuming driverID is used as rider_id for simplicity
			"driver_id": driverID.String(),
			"status":    string(newStatus),
		},
	}
	payloadBytes, _ := json.Marshal(payload)

	ride, err := u.repo.UpdateTripStatus(ctx, rideID, driverID, expectedOldStatus, newStatus, string(newStatus), payloadBytes)
	if err != nil {
		span.RecordError(err)
		return nil, errs.ErrConflict.WithMessage("Cannot update status. Either ride not found, wrong driver, or invalid state transition.").WithRootErr(err)
	}

	return ride, nil
}
