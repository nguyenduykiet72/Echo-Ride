package domain

import (
	"context"

	"github.com/google/uuid"
)

type Ride struct {
	ID         uuid.UUID
	RiderID    uuid.UUID
	DriverID   *uuid.UUID
	PickupLat  float64
	PickupLon  float64
	DropoffLat float64
	DropoffLon float64
	Status     string
	Price      float64
}

type RideFilter struct {
	RiderID  *uuid.UUID
	DriverID *uuid.UUID
	Status   *string // optional filter by status (e.g., "requested", "accepted", "completed") why pointer because we want to distinguish between "no filter" (nil) and "filter by empty string" ("")
	Limit    int32
	Offset   int32
}

type RideEventPayload struct {
	EventID   string      `json:"event_id"`
	EventType string      `json:"event_type"`
	Timestamp string      `json:"timestamp"`
	Data      interface{} `json:"data"`
}

type RideRepository interface {
	Create(ctx context.Context, ride *Ride, eventType string, eventPayload []byte) (*Ride, error)
	GetByID(ctx context.Context, id uuid.UUID) (*Ride, error)
	ListRides(ctx context.Context, filter RideFilter) ([]*Ride, error)
	AcceptRide(ctx context.Context, rideID, driverID uuid.UUID, eventType string, eventPayload []byte) (*Ride, error)
	UpdateStatus(ctx context.Context, rideID uuid.UUID, status string, eventType string, eventPayload []byte) (*Ride, error)
}
