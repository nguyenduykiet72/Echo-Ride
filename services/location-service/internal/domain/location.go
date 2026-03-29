package domain

import (
	"context"
	"time"
)

type DriverLocation struct {
	DriverID   string  `json:"driver_id"`
	Lat        float64 `json:"lat"`
	Lng        float64 `json:"lng"`
	DistanceKm float64 `json:"distance_km,omitempty"`
}

type LocationRepository interface {
	SaveLocationBatch(ctx context.Context, locations []DriverLocation) error
	RemoveStaleDrivers(ctx context.Context, olderThan time.Time) (int64, error)
	FindNearestDrivers(ctx context.Context, lat, lng, radius float64, limit int) ([]DriverLocation, error)
}
