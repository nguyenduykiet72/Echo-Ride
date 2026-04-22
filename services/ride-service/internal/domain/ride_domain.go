package domain

import (
	"context"

	"github.com/google/uuid"
)

type RideStatus string
type EventType string
type OutboxState string

const (
	RideStatusRequested  RideStatus = "REQUESTED"
	RideStatusAccepted   RideStatus = "ACCEPTED"
	RideStatusInProgress RideStatus = "IN_PROGRESS"
	RideStatusCompleted  RideStatus = "COMPLETED"
	RideStatusCancelled  RideStatus = "CANCELLED"
	RideStatusFailed     RideStatus = "FAILED"
)

const (
	OutboxStateRide OutboxState = "RIDE"
)

const (
	EventTypeRideRequested EventType = "RIDE_REQUESTED"
	EventTypeRideAccepted  EventType = "RIDE_ACCEPTED"
	EventTypeRideCompleted EventType = "RIDE_COMPLETED"
	EventTypeRideCancelled EventType = "RIDE_CANCELLED"
	EventTypeRideFailed    EventType = "RIDE_FAILED"
)

type Ride struct {
	ID         uuid.UUID  `json:"id"`
	RiderID    uuid.UUID  `json:"rider_id"`
	DriverID   *uuid.UUID `json:"driver_id,omitempty"` // pointer to allow null value when no driver assigned
	PickupLat  float64    `json:"pickup_lat"`
	PickupLng  float64    `json:"pickup_lng"`
	DropoffLat float64    `json:"dropoff_lat"`
	DropoffLng float64    `json:"dropoff_lng"`
	Status     RideStatus `json:"status"`
	Price      float64    `json:"price"`
}

type RideFilter struct {
	RiderID  *uuid.UUID
	DriverID *uuid.UUID
	Status   *string // optional filter by status (e.g., "requested", "accepted", "completed") why pointer because we want to distinguish between "no filter" (nil) and "filter by empty string" ("")
	Limit    int32
	Offset   int32
}

type RideEventPayload struct {
	EventID      string            `json:"event_id"`
	EventType    EventType         `json:"event_type"`
	Timestamp    string            `json:"timestamp"`
	Data         interface{}       `json:"data"`
	TraceContext map[string]string `json:"trace_context,omitempty"`
}

type RideRepository interface {
	Create(ctx context.Context, ride *Ride, eventType string, eventPayload []byte) (*Ride, error)
	GetByID(ctx context.Context, id uuid.UUID) (*Ride, error)
	ListRides(ctx context.Context, filter RideFilter) ([]*Ride, error)
	AcceptRide(ctx context.Context, rideID, driverID uuid.UUID, eventType string, eventPayload []byte) (*Ride, error)
	UpdateStatus(ctx context.Context, rideID uuid.UUID, status RideStatus, eventType string, eventPayload []byte) (*Ride, error)
	UpdateTripStatus(ctx context.Context, rideID, driverID uuid.UUID, oldStatus, newStatus RideStatus, eventType string, eventPayload []byte) (*Ride, error)
}
