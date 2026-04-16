package kafka

import (
	"context"
	"echo-ride/services/ride-service/internal/application"
	"echo-ride/services/ride-service/internal/domain"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

type IncomingKafkaEvent struct {
	EventID      string `json:"event_id"`
	EventType    string `json:"event_type"`
	EventPayload string `json:"event_payload"`
}

type FailedPayloadWrapper struct {
	RideID string `json:"ride_id"`
	Reason string `json:"reason"`
}

type RideConsumer struct {
	reader             *kafka.Reader
	updateRideStatusUC application.UpdateRideUseCase
	logger             *zap.Logger
}

func NewRideConsumer(
	cfg kafka.ReaderConfig,
	updateRideStatusUC application.UpdateRideUseCase,
	logger *zap.Logger,
) *RideConsumer {
	return &RideConsumer{
		reader:             kafka.NewReader(cfg),
		updateRideStatusUC: updateRideStatusUC,
		logger:             logger,
	}
}

func (c *RideConsumer) Start(ctx context.Context) {
	c.logger.Info("Ride Service Consumer started, listening to topic")

	for {
		m, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			c.logger.Error("Failed to fetch message", zap.Error(err))
			continue
		}

		var event IncomingKafkaEvent
		if err := json.Unmarshal(m.Value, &event); err != nil {
			c.logger.Error("Failed to parse event", zap.Error(err))
			c.reader.CommitMessages(ctx, m)
			continue
		}

		switch event.EventType {
		case "RIDE_FAILED", "FAILED":
			c.handleMatchingFailed(ctx, m, event)
		default:
			c.reader.CommitMessages(ctx, m)
		}
	}
}

func (c *RideConsumer) handleMatchingFailed(ctx context.Context, message kafka.Message, event IncomingKafkaEvent) {
	var payload FailedPayloadWrapper

	if err := json.Unmarshal([]byte(event.EventPayload), &payload); err != nil {
		c.logger.Error("Failed to parse FAILED payload", zap.Error(err))
		c.reader.CommitMessages(ctx, message)
		return
	}

	rideUUID, err := uuid.Parse(payload.RideID)
	if err != nil {
		c.logger.Error("Invalid ride_id UUID format", zap.Error(err), zap.String("ride_id", payload.RideID))
		c.reader.CommitMessages(ctx, message)
		return
	}

	err = c.updateRideStatusUC.Execute(ctx, rideUUID, domain.EventTypeRideFailed)
	if err != nil {
		c.logger.Error("Failed to update ride status", zap.Error(err))
		return
	}

	c.reader.CommitMessages(ctx, message)
	c.logger.Info("Successfully handled matching failed event", zap.String("ride_id", rideUUID.String()))
}
