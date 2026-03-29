package application

import (
	"context"
	"echo-ride/services/location-service/internal/domain"
	"time"

	"go.uber.org/zap"
)

type LocationBatcher struct {
	repo          domain.LocationRepository
	inputChan     chan domain.DriverLocation
	batchSize     int
	flushInterval time.Duration // how often to flush the batch even if it's not full
	logger        *zap.Logger
}

func NewLocationBatcher(repo domain.LocationRepository, logger *zap.Logger) *LocationBatcher {
	return &LocationBatcher{
		repo:          repo,
		inputChan:     make(chan domain.DriverLocation, 5000), // buffered channel to handle bursts
		batchSize:     500,                                    // batch size of 100 locations
		flushInterval: 500 * time.Millisecond,                 // flush every 5 seconds if batch is not full
		logger:        logger,
	}
}

func (b *LocationBatcher) Push(loc domain.DriverLocation) {
	select {
	case b.inputChan <- loc:
	default:
		b.logger.Warn("Location batcher input channel is full, dropping location update", zap.String("driver_id", loc.DriverID))
	}
}

func (b *LocationBatcher) Start(ctx context.Context) {
	b.logger.Info("Location Batcher Worker started")

	buffer := make([]domain.DriverLocation, 0, b.batchSize)
	ticker := time.NewTicker(b.flushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			b.flush(buffer)
			b.logger.Info("Location Batcher Worker stopped")
			return

		case loc := <-b.inputChan:
			buffer = append(buffer, loc)
			if len(buffer) >= b.batchSize {
				b.flush(buffer)
				buffer = buffer[:0] // reset buffer
			}

		case <-ticker.C:
			if len(buffer) > 0 {
				b.flush(buffer)
				buffer = buffer[:0] // reset buffer
			}
		}
	}
}

func (b *LocationBatcher) flush(locations []domain.DriverLocation) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := b.repo.SaveLocationBatch(ctx, locations)
	if err != nil {
		b.logger.Error("Failed to save location batch", zap.Int("batch_size", len(locations)), zap.Error(err))
	} else {
		b.logger.Debug("Flushed locations to Redis", zap.Int("batch_size", len(locations)))
	}

}
