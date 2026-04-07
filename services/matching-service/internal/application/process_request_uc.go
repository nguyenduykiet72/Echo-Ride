package application

import (
	"context"
	"echo-ride/services/matching-service/internal/domain"
	"time"

	"go.uber.org/zap"
)

type ProcessRideRequestUseCase interface {
	Execute(ctx context.Context, event domain.RideRequestEvent) error
}

type processRideRequestUseCase struct {
	locationGateway domain.LocationGateway
	dispatchRepo    domain.DispatchRepository
	logger          *zap.Logger
}

func NewProcessRideRequestUseCase(locGw domain.LocationGateway, dispatchRepo domain.DispatchRepository, logger *zap.Logger) ProcessRideRequestUseCase {
	return &processRideRequestUseCase{
		locationGateway: locGw,
		dispatchRepo:    dispatchRepo,
		logger:          logger,
	}
}

func (p *processRideRequestUseCase) Execute(ctx context.Context, event domain.RideRequestEvent) error {
	p.logger.Info("Processing Ride Request", zap.String("ride_id", event.RideID))

	candidates, err := p.locationGateway.GetNearestDrivers(ctx, event.PickupLat, event.PickupLng, 3.0, 10)
	if err != nil {
		p.logger.Error("Failed to get nearest drivers", zap.Error(err))
		return err
	}

	if len(candidates) == 0 {
		p.logger.Info("No drivers found nearby", zap.String("ride_id", event.RideID))

		failedState := domain.RideDispatchState{
			RideID:       event.RideID,
			Candidates:   []domain.CandidateDriver{},
			CurrentIndex: 0,
			Status:       "FAILED",
			UpdatedAt:    time.Now().Unix(),
		}

		if err := p.dispatchRepo.SaveState(ctx, failedState); err != nil {
			p.logger.Error("Failed to save failed dispatch state", zap.Error(err))
			return err
		}

		// TODO: Publish RideFailed event to notify other services RIDE_MATCHING_FAILED

		return nil
	}

	state := domain.RideDispatchState{
		RideID:       event.RideID,
		Candidates:   candidates,
		CurrentIndex: 0,
		Status:       domain.RideStatusFinding,
		UpdatedAt:    time.Now().Unix(),
	}

	if err := p.dispatchRepo.SaveState(ctx, state); err != nil {
		p.logger.Error("Failed to save dispatch state", zap.Error(err))
		return err
	}

	expireAt := time.Now().Add(15 * time.Second).Unix() // 15 seconds to wait for driver response
	if err := p.dispatchRepo.SetTimeout(ctx, event.RideID, expireAt); err != nil {
		p.logger.Error("Failed to set dispatch timeout", zap.Error(err))
		return err
	}

	p.logger.Info("Successfully pushed to Timeout Queue",
		zap.String("ride_id", event.RideID),
		zap.Int64("expire_at", expireAt),
	)

	p.logger.Info("Found candidate drivers", zap.Int("count", len(candidates)))

	topDriver := candidates[0]
	p.logger.Info("Top candidate driver", zap.String("driver_id", topDriver.DriverID), zap.Float64("distance_km", topDriver.DistanceKm))

	return nil
}
