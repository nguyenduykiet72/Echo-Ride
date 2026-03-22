package http

import "github.com/google/uuid"

type createRideRequest struct {
	RiderID    uuid.UUID `json:"rider_id" validate:"required,uuid"`
	PickupLat  float64   `json:"pickup_lat" validate:"required"`
	PickupLon  float64   `json:"pickup_lon" validate:"required"`
	DropoffLat float64   `json:"dropoff_lat" validate:"required"`
	DropoffLon float64   `json:"dropoff_lon" validate:"required"`
}

type createRideResponse struct {
	RideID uuid.UUID `json:"ride_id"`
	Status string    `json:"status"`
	Price  float64   `json:"price"`
}

type acceptRideRequest struct {
	DriverID string `json:"driver_id" validate:"required,uuid"`
}

type updateStatusRequest struct {
	Status string `json:"status" validate:"required,oneof=IN_PROGRESS COMPLETED CANCELED"`
}
