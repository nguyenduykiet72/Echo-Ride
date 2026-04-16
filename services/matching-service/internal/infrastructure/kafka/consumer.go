package kafka

import (
	"context"
	"echo-ride/services/matching-service/config"
	"echo-ride/services/matching-service/internal/application"
	"echo-ride/services/matching-service/internal/domain"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/segmentio/kafka-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

type IncomingKafkaEvent struct {
	EventID            string                 `json:"event_id"`
	EventAggregateID   string                 `json:"event_aggregate_id"`
	EventAggregateType string                 `json:"event_aggregate_type"`
	EventType          domain.RideEventStatus `json:"event_type"`
	EventPayload       json.RawMessage        `json:"event_payload"`
	EventStatus        string                 `json:"event_status"`
	TraceContext       map[string]string      `json:"trace_context"`
}

type RideRequestedPayload struct {
	ID         string  `json:"id"`
	RiderID    string  `json:"rider_id"`
	PickupLat  float64 `json:"pickup_lat"`
	PickupLng  float64 `json:"pickup_lng"`
	DropoffLat float64 `json:"dropoff_lat"`
	DropoffLng float64 `json:"dropoff_lng"`
}

type RideConsumer struct {
	reader    *kafka.Reader
	dlqWriter *kafka.Writer
	uc        application.ProcessRideRequestUseCase
	repo      domain.DispatchRepository
	logger    *zap.Logger
}

type RideRequestedWrapper struct {
	Data struct {
		ID        string  `json:"id"`
		RiderID   string  `json:"rider_id"`
		PickupLat float64 `json:"pickup_lat"`
		PickupLng float64 `json:"pickup_lng"`
	} `json:"data"`
}

func NewRideConsumer(
	cfg config.KafkaConfig,
	uc application.ProcessRideRequestUseCase,
	repo domain.DispatchRepository,
	logger *zap.Logger,
) *RideConsumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  cfg.Brokers,
		Topic:    cfg.Topic,
		GroupID:  cfg.GroupID,
		MinBytes: 10e3, // 10KB
		MaxBytes: 10e6, // 10MB
	})

	dlqWriter := &kafka.Writer{
		Addr:     kafka.TCP(cfg.Brokers...),
		Topic:    cfg.Topic,
		Balancer: &kafka.LeastBytes{},
	}

	return &RideConsumer{
		reader:    reader,
		dlqWriter: dlqWriter,
		uc:        uc,
		repo:      repo,
		logger:    logger,
	}
}

func (c *RideConsumer) Start(ctx context.Context) {
	c.logger.Info("Kafka Consumer started, listening to topic", zap.String("topic", c.reader.Config().Topic))

	for {
		m, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if strings.Contains(err.Error(), "context canceled") {
				c.logger.Info("Kafka Consumer context canceled", zap.String("topic", c.reader.Config().Topic))
				break
			}
			c.logger.Error("Kafka Consumer error", zap.Error(err))
			continue
		}
		c.processMessage(ctx, m)
	}
}

func (c *RideConsumer) processMessage(ctx context.Context, m kafka.Message) {
	if len(m.Value) == 0 {
		c.logger.Debug("Received tombstone message (empty value), skipping", zap.ByteString("key", m.Key))
		c.reader.CommitMessages(ctx, m) // Bỏ qua và đánh dấu đã đọc
		return
	}

	var event IncomingKafkaEvent
	if err := json.Unmarshal(m.Value, &event); err != nil {
		c.logger.Error("Failed to unmarshal Kafka message", zap.Error(err), zap.ByteString("message", m.Value))
		c.sendToDLQ(ctx, m, err.Error())
		c.reader.CommitMessages(ctx, m)
		return
	}

	if event.EventType != domain.RideEventStatusRequested && event.EventType != domain.RideEventStatusAccepted {
		c.reader.CommitMessages(ctx, m)
		return
	}

	parentCtx := otel.GetTextMapPropagator().Extract(ctx, propagation.MapCarrier(event.TraceContext))

	tracer := otel.Tracer("matching-service")
	spanCtx, span := tracer.Start(parentCtx, string("Kafka Consume "+event.EventType), trace.WithSpanKind(trace.SpanKindConsumer))
	defer span.End()

	isNewEvent, err := c.repo.CheckAndSetIdempotency(spanCtx, event.EventID)
	if err != nil {
		c.logger.Error("Failed to check idempotency", zap.Error(err), zap.String("event_id", event.EventID))
		return
	}

	if !isNewEvent {
		c.logger.Warn("Idempotency hit for event, skipping processing", zap.String("event_id", event.EventID))
		c.reader.CommitMessages(ctx, m)
		return
	}

	switch event.EventType {
	case domain.RideEventStatusRequested:
		c.handleRideRequested(spanCtx, m, event)
	case domain.RideEventStatusAccepted:
		c.handleRideAccepted(spanCtx, m, event)
	default:

		c.logger.Warn("Ignored unhandled event type",
			zap.String("event_type", string(event.EventType)),
			zap.String("event_id", event.EventID))
		c.reader.CommitMessages(ctx, m)
	}
}

func (c *RideConsumer) handleRideRequested(ctx context.Context, m kafka.Message, event IncomingKafkaEvent) {
	var payloadStr string
	if err := json.Unmarshal(event.EventPayload, &payloadStr); err != nil {
		c.logger.Error("Failed to unmarshal REQUESTED payload string", zap.Error(err))
		c.sendToDLQ(ctx, m, "invalid_payload_string")
		c.reader.CommitMessages(ctx, m)
		return
	}

	var payload RideRequestedWrapper
	if err := json.Unmarshal([]byte(payloadStr), &payload); err != nil {
		c.logger.Error("Failed to unmarshal REQUESTED payload", zap.Error(err))
		c.sendToDLQ(ctx, m, "invalid_payload")
		c.reader.CommitMessages(ctx, m)
		return
	}

	domainEvent := domain.RideRequestEvent{
		RideID:    payload.Data.ID,
		RiderID:   payload.Data.RiderID,
		PickupLat: payload.Data.PickupLat,
		PickupLng: payload.Data.PickupLng,
	}

	maxRetries := 3
	backoff := 1 * time.Second

	var err error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		err = c.uc.Execute(ctx, domainEvent)
		if err == nil {
			break
		}
		c.logger.Warn("Failed to execute UseCase", zap.Error(err), zap.Int("attempt", attempt))
		if attempt == maxRetries {
			break
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
			backoff *= 2
		}
	}

	if err != nil {
		c.logger.Error("Failed to process ride request event after retries", zap.Error(err), zap.String("ride_id", payload.Data.ID))
		c.sendToDLQ(ctx, m, err.Error())
	}

	if err := c.reader.CommitMessages(ctx, m); err != nil {
		c.logger.Error("Failed to commit message after processing", zap.Error(err), zap.String("ride_id", payload.Data.ID))
	} else {
		c.logger.Info("Successfully processed and committed message", zap.String("ride_id", payload.Data.ID))
	}
}

func (c *RideConsumer) handleRideAccepted(ctx context.Context, m kafka.Message, event IncomingKafkaEvent) {
	rideID := event.EventAggregateID
	c.logger.Info("Ride has been ACCEPTED. Stopping dispatcher...", zap.String("ride_id", rideID))

	if err := c.repo.RemoveTimeout(ctx, rideID); err != nil {
		c.logger.Error("Failed to remove timeout for accepted ride", zap.String("ride_id", rideID), zap.Error(err))
	}

	err := c.repo.DeleteState(ctx, rideID)
	if err != nil {
		c.logger.Error("Failed to delete state for accepted ride", zap.String("ride_id", rideID), zap.Error(err))
	}

	if err := c.reader.CommitMessages(ctx, m); err != nil {
		c.logger.Error("Failed to commit message after processing ACCEPTED event", zap.Error(err), zap.String("ride_id", rideID))
	} else {
		c.logger.Info("Successfully processed ACCEPTED event and committed message", zap.String("ride_id", rideID))
	}
}

func (c *RideConsumer) sendToDLQ(ctx context.Context, m kafka.Message, errorMsg string) {
	headers := []kafka.Header{
		{Key: "error-reason", Value: []byte(errorMsg)},
		{Key: "original-topic", Value: []byte(c.reader.Config().Topic)},
		{Key: "original-partition", Value: []byte(fmt.Sprintf("%d", m.Partition))},
		{Key: "original-offset", Value: []byte(fmt.Sprintf("%d", m.Offset))},
		{Key: "failure-timestamp", Value: []byte(time.Now().Format(time.RFC3339))},
	}

	headers = append(headers, m.Headers...)

	dlsMsg := kafka.Message{
		Key:     m.Key,
		Value:   m.Value,
		Headers: headers,
	}

	if err := c.dlqWriter.WriteMessages(ctx, dlsMsg); err != nil {
		c.logger.Error("Failed to write message to DLQ", zap.Error(err), zap.ByteString("message", m.Value))
	} else {
		c.logger.Info("Message sent to DLQ", zap.ByteString("message", m.Value), zap.String("error", errorMsg))
	}
}

func (c *RideConsumer) Close() error {
	c.dlqWriter.Close()
	return c.reader.Close()
}
