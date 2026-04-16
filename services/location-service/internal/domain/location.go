package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type DriverLocation struct {
	DriverID   uuid.UUID `json:"driver_id"`
	Lat        float64   `json:"lat"`
	Lng        float64   `json:"lng"`
	DistanceKm float64   `json:"distance_km,omitempty"`
}

type LocationRequest struct {
	RideID uuid.UUID
	Lat    float64
	Lng    float64
	Radius float64
	Limit  int
}

type DriverETA struct {
	DriverID uuid.UUID
	Lat      float64
	Lng      float64
	ETA      float64
	Distance float64
}

type LocationRepository interface {
	SaveLocationBatch(ctx context.Context, locations []DriverLocation) error
	RemoveStaleDrivers(ctx context.Context, olderThan time.Time) (int64, error)
	FindNearestDrivers(ctx context.Context, lat, lng, radius float64, limit int) ([]DriverLocation, error)
	FindDriversInRadius(ctx context.Context, lat, lng, radius float64, limit int) ([]DriverLocation, error)
}

type RoutingService interface {
	CalculateETAMatrix(ctx context.Context, originLat, originLng float64, destinations []DriverLocation) ([]DriverETA, error)
}
