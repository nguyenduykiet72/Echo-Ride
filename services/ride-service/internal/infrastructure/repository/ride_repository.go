package repository

import (
	"context"
	"echo-ride/services/ride-service/internal/domain"
	"echo-ride/services/ride-service/internal/infrastructure/db/dbgen"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RideRepositoryImpl struct {
	db *pgxpool.Pool
	q  *dbgen.Queries
}

func NewRideRepository(db *pgxpool.Pool) domain.RideRepository {
	return &RideRepositoryImpl{
		db: db,
		q:  dbgen.New(db),
	}
}

func toPgtypeUUID(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: id, Valid: true}
}

func toUUID(id pgtype.UUID) uuid.UUID {
	return uuid.UUID(id.Bytes)
}

func toPgtypeNumeric(f float64) pgtype.Numeric {
	var n pgtype.Numeric
	_ = n.Scan(fmt.Sprintf("%f", f))
	return n
}

func numericToFloat64(n pgtype.Numeric) float64 {
	f, _ := n.Float64Value()
	return f.Float64
}

func dbRideToDomain(dbRide dbgen.TRide) *domain.Ride {
	var driverID *uuid.UUID
	if dbRide.RideDriverID.Valid {
		id := toUUID(dbRide.RideDriverID)
		driverID = &id
	}

	return &domain.Ride{
		ID:         toUUID(dbRide.RideID),
		RiderID:    toUUID(dbRide.RideRiderID),
		DriverID:   driverID,
		PickupLat:  numericToFloat64(dbRide.RidePickupLat),
		PickupLon:  numericToFloat64(dbRide.RidePickupLon),
		DropoffLat: numericToFloat64(dbRide.RideDropoffLat),
		DropoffLon: numericToFloat64(dbRide.RideDropoffLon),
		Status:     string(dbRide.RideStatus),
		Price:      numericToFloat64(dbRide.RidePrice),
	}
}

func (r *RideRepositoryImpl) Create(ctx context.Context, ride *domain.Ride) (*domain.Ride, error) {
	params := dbgen.CreateRideParams{
		RideRiderID:    toPgtypeUUID(ride.RiderID),
		RidePickupLat:  toPgtypeNumeric(ride.PickupLat),
		RidePickupLon:  toPgtypeNumeric(ride.PickupLon),
		RideDropoffLat: toPgtypeNumeric(ride.DropoffLat),
		RideDropoffLon: toPgtypeNumeric(ride.DropoffLon),
		RidePrice:      toPgtypeNumeric(ride.Price),
	}

	dbRide, err := r.q.CreateRide(ctx, params)
	if err != nil {
		return nil, err
	}

	ride.ID = toUUID(dbRide.RideID)
	ride.Status = string(dbRide.RideStatus)

	return ride, nil
}

func (r *RideRepositoryImpl) GetByID(ctx context.Context, id uuid.UUID) (*domain.Ride, error) {
	dbRide, err := r.q.GetRideByID(ctx, toPgtypeUUID(id))
	if err != nil {
		return nil, err
	}

	return dbRideToDomain(dbRide), nil
}

func (r *RideRepositoryImpl) ListRides(ctx context.Context, filter domain.RideFilter) ([]*domain.Ride, error) {
	params := dbgen.ListRidesParams{
		Limit:  filter.Limit,
		Offset: filter.Offset,
	}

	if filter.RiderID != nil {
		params.RiderID = toPgtypeUUID(*filter.RiderID)
	}

	if filter.DriverID != nil {
		params.DriverID = toPgtypeUUID(*filter.DriverID)
	}

	if filter.Status != nil {
		params.Status = dbgen.NullRideStatus{
			RideStatus: dbgen.RideStatus(*filter.Status),
			Valid:      true,
		}
	}

	rows, err := r.q.ListRides(ctx, params)
	if err != nil {
		return nil, err
	}

	result := make([]*domain.Ride, 0, len(rows))
	for _, row := range rows {
		result = append(result, dbRideToDomain(row))
	}

	return result, nil
}

func (r *RideRepositoryImpl) AcceptRide(ctx context.Context, rideID, driverID uuid.UUID) (*domain.Ride, error) {
	params := dbgen.AcceptRideParams{
		RideID:       toPgtypeUUID(rideID),
		RideDriverID: toPgtypeUUID(driverID),
	}

	dbRide, err := r.q.AcceptRide(ctx, params)
	if err != nil {
		return nil, err
	}

	return dbRideToDomain(dbRide), nil
}

func (r *RideRepositoryImpl) UpdateStatus(ctx context.Context, rideID uuid.UUID, status string) (*domain.Ride, error) {
	params := dbgen.UpdateRideStatusParams{
		RideID:     toPgtypeUUID(rideID),
		RideStatus: dbgen.RideStatus(status),
	}

	dbRide, err := r.q.UpdateRideStatus(ctx, params)
	if err != nil {
		return nil, err
	}

	return dbRideToDomain(dbRide), nil
}
