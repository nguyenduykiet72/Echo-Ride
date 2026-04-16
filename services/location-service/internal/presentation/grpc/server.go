package grpc

import (
	"context"
	"echo-ride/pkg/errs"
	pb "echo-ride/pkg/grpc/location/v1"
	"echo-ride/services/location-service/internal/application"
	"echo-ride/services/location-service/internal/domain"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

type LocationGrpcServer struct {
	pb.UnimplementedLocationServiceServer
	findDriversUC application.FindDriversUseCase
	//repo   domain.LocationRepository
	logger *zap.Logger
	tracer trace.Tracer
}

//	func NewLocationGrpcServer(repo domain.LocationRepository, logger *zap.Logger) *LocationGrpcServer {
//		return &LocationGrpcServer{
//			repo:   repo,
//			logger: logger,
//		}
//	}

func NewLocationGrpcServer(findDriversUC application.FindDriversUseCase, logger *zap.Logger) *LocationGrpcServer {
	return &LocationGrpcServer{
		findDriversUC: findDriversUC,
		tracer:        otel.Tracer("location-grpc-server"),
		logger:        logger,
	}
}

func (s *LocationGrpcServer) FindNearestDrivers(ctx context.Context, req *pb.FindNearestDriversRequest) (*pb.FindNearestDriversResponse, error) {
	//s.logger.Info("Received gRPC request FindNearestDrivers",
	//	zap.Float64("pickup_lat", req.PickupLat),
	//	zap.Float64("pickup_lon", req.PickupLng),
	//)
	//
	//if req.RadiusKm <= 0 || req.Limit <= 0 {
	//	return nil, errs.ErrInvalidArgument
	//}
	//
	//drivers, err := s.repo.FindNearestDrivers(ctx, req.PickupLat, req.PickupLng, req.RadiusKm, int(req.Limit))
	//if err != nil {
	//	s.logger.Error("Failed to find drivers", zap.Error(err))
	//	return nil, errs.ErrInternal.WithRootErr(err)
	//}
	//
	//var resDrivers []*pb.DriverDistance
	//for _, d := range drivers {
	//	resDrivers = append(resDrivers, &pb.DriverDistance{
	//		DriverId:   d.DriverID.String(),
	//		Lat:        d.Lat,
	//		Lng:        d.Lng,
	//		DistanceKm: d.DistanceKm,
	//	})
	//}
	//
	//return &pb.FindNearestDriversResponse{
	//	Drivers: resDrivers,
	//}, nil

	ctx, span := s.tracer.Start(ctx, "GRPC FindNearestDrivers")
	defer span.End()

	rideID, err := uuid.Parse(req.GetRideId())
	if err != nil {
		return nil, errs.ErrInvalidArgument.WithMessage("Invalid ride ID").WithRootErr(err)
	}

	domainReq := domain.LocationRequest{
		RideID: rideID,
		Lat:    req.GetPickupLat(),
		Lng:    req.GetPickupLng(),
		Radius: req.GetRadiusKm(),
		Limit:  int(req.GetLimit()),
	}

	driversETA, err := s.findDriversUC.Execute(ctx, domainReq)
	if err != nil {
		span.RecordError(err)
		return nil, errs.ErrInternal.WithMessage("Failed to find nearest drivers").WithRootErr(err)
	}

	var pbDrivers []*pb.DriverEtaInfo
	for _, driverETA := range driversETA {
		pbDrivers = append(pbDrivers, &pb.DriverEtaInfo{
			DriverId:   driverETA.DriverID.String(),
			Lat:        driverETA.Lat,
			Lng:        driverETA.Lng,
			EtaSeconds: int32(driverETA.ETA),
			DistanceKm: driverETA.Distance,
		})
	}

	return &pb.FindNearestDriversResponse{
		Drivers: pbDrivers,
	}, nil
}
