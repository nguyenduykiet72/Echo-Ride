package application

import (
	"context"
	"echo-ride/services/matching-service/internal/domain"
	"time"

	"go.uber.org/zap"
)

type TimeoutWatcher struct {
	dispatchRepo domain.DispatchRepository
	logger       *zap.Logger
}

func NewTimeoutWatcher(dispatchRepo domain.DispatchRepository, logger *zap.Logger) *TimeoutWatcher {
	return &TimeoutWatcher{
		dispatchRepo: dispatchRepo,
		logger:       logger,
	}
}

func (w *TimeoutWatcher) Start(ctx context.Context) {
	w.logger.Info("Starting timeout watcher")

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			w.logger.Info("Stopping timeout watcher")
			return
		case <-ticker.C:
			w.checkTimeouts(ctx)
		}
	}
}

func (w *TimeoutWatcher) checkTimeouts(ctx context.Context) {
	now := time.Now().Unix()

	expiredRides, err := w.dispatchRepo.GetExpiredRides(ctx, now, 100)
	if err != nil {
		w.logger.Error("Failed to get expired rides", zap.Error(err))
		return
	}

	if len(expiredRides) > 0 {
		w.logger.Info("Found expired rides", zap.Int("count", len(expiredRides)))
	}

	for _, rideID := range expiredRides {
		w.handleExpiredRide(ctx, rideID, now)
	}
}

func (w *TimeoutWatcher) handleExpiredRide(ctx context.Context, rideID string, now int64) {
	state, err := w.dispatchRepo.GetState(ctx, rideID)
	if err != nil || state == nil {
		w.logger.Error("Failed to get state", zap.String("rideID", rideID), zap.Error(err))
		w.dispatchRepo.RemoveTimeout(ctx, rideID)
		return
	}

	w.logger.Info("X-RAY: Unmarshalled State",
		zap.String("status", string(state.Status)),
		zap.Int("current_index", state.CurrentIndex),
		zap.Int("candidates_count", len(state.Candidates)),
	)

	if state.Status != domain.RideStatusFinding {
		w.dispatchRepo.RemoveTimeout(ctx, rideID)
		return
	}

	state.CurrentIndex++

	if state.CurrentIndex >= len(state.Candidates) {
		w.logger.Warn("Timeout watcher has expired ride", zap.String("rideID", rideID))
		state.Status = domain.RideStatusFailed
		state.UpdatedAt = now
		w.dispatchRepo.SaveState(ctx, *state) // why pointer? because we need to update the state in redis
		w.dispatchRepo.RemoveTimeout(ctx, rideID)

		// TODO: publish event RIDE_MATCHING_FAILED to notify other services
		return
	}

	state.UpdatedAt = now
	if err := w.dispatchRepo.SaveState(ctx, *state); err != nil {
		w.logger.Error("Failed to update state for expired ride", zap.String("rideID", rideID), zap.Error(err))
		return
	}

	newExpireAt := now + 15 // 15 seconds to wait for next driver response
	if err := w.dispatchRepo.SetTimeout(ctx, rideID, newExpireAt); err != nil {
		w.logger.Error("Failed to set new timeout for expired ride", zap.String("rideID", rideID), zap.Error(err))
		return
	}

	nextDriver := state.Candidates[state.CurrentIndex]
	w.logger.Info("Driver timeout. Dispatched to NEXT Driver",
		zap.String("ride_id", rideID),
		zap.String("next_driver_id", nextDriver.DriverID),
		zap.Int("driver_index", state.CurrentIndex),
	)
}
