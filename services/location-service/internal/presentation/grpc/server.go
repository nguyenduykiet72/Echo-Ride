package grpc

import (
	"context"
	"echo-ride/pkg/errs"
	"echo-ride/services/location-service/internal/domain"
	pb "echo-ride/services/location-service/pkg/grpc/generated/location/v1"

	"go.uber.org/zap"
)

type LocationGrpcServer struct {
	pb.UnimplementedLocationServiceServer
	repo   domain.LocationRepository
	logger *zap.Logger
}

func NewLocationGrpcServer(repo domain.LocationRepository, logger *zap.Logger) *LocationGrpcServer {
	return &LocationGrpcServer{
		repo:   repo,
		logger: logger,
	}
}

func (s *LocationGrpcServer) FindNearestDrivers(ctx context.Context, req *pb.FindNearestDriversRequest) (*pb.FindNearestDriversResponse, error) {
	s.logger.Info("Received gRPC request FindNearestDrivers",
		zap.Float64("pickup_lat", req.PickupLat),
		zap.Float64("pickup_lon", req.PickupLng),
	)

	if req.RadiusKm <= 0 || req.Limit <= 0 {
		return nil, errs.ErrInvalidArgument
	}

	drivers, err := s.repo.FindNearestDrivers(ctx, req.PickupLat, req.PickupLng, req.RadiusKm, int(req.Limit))
	if err != nil {
		s.logger.Error("Failed to find drivers", zap.Error(err))
		return nil, errs.ErrInternal.WithRootErr(err)
	}

	var resDrivers []*pb.DriverDistance
	for _, d := range drivers {
		resDrivers = append(resDrivers, &pb.DriverDistance{
			DriverId:   d.DriverID,
			Lat:        d.Lat,
			Lng:        d.Lng,
			DistanceKm: d.DistanceKm,
		})
	}

	return &pb.FindNearestDriversResponse{
		Drivers: resDrivers,
	}, nil
}
