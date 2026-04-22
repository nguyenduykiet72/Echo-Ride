package http

import (
	"echo-ride/pkg/errs"
	"echo-ride/pkg/response"
	"echo-ride/services/ride-service/internal/application"
	"echo-ride/services/ride-service/internal/domain"
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
)

type RideHandler struct {
	createRideUC application.CreateRideUseCase
	updateRideUC application.UpdateRideUseCase
	getRideUC    application.GetRideUseCase
	acceptRideUC application.AcceptRideUseCase
	updateTripUC application.UpdateTripStatusUseCase
}

func NewRideHandler(e *echo.Echo, createRideUC application.CreateRideUseCase, updateRideUC application.UpdateRideUseCase, getRideUC application.GetRideUseCase, acceptRideUC application.AcceptRideUseCase) {
	handler := &RideHandler{
		createRideUC: createRideUC,
		updateRideUC: updateRideUC,
		getRideUC:    getRideUC,
		acceptRideUC: acceptRideUC,
	}

	v1 := e.Group("/api/v1/rides")
	v1.POST("", handler.CreateRide)
	v1.GET("", handler.ListRides)
	v1.GET("/:id", handler.GetByID)
	v1.PATCH("/:id/accept", handler.AcceptRide)
	v1.PATCH("/:id/status", handler.UpdateStatus)
	v1.PATCH("/:id/trip-status", handler.UpdateTripStatus)
}

func (h *RideHandler) CreateRide(ctx *echo.Context) error {
	var req createRideRequest

	if err := ctx.Bind(&req); err != nil {
		return errs.ErrBadRequest.WithMessage("Failed to bind create ride").WithRootErr(err)
	}

	if err := ctx.Validate(&req); err != nil {
		return errs.ErrBadRequest.WithMessage("Validation failed").WithRootErr(err)
	}

	cmd := application.CreateRideCommand{
		RiderID:    req.RiderID,
		PickupLat:  req.PickupLat,
		PickupLng:  req.PickupLon,
		DropoffLat: req.DropoffLat,
		DropoffLng: req.DropoffLon,
	}

	ride, err := h.createRideUC.Execute(ctx.Request().Context(), cmd)
	if err != nil {
		return errs.ErrInternal.WithMessage(err.Error())
	}

	resp := createRideResponse{
		RideID: ride.ID,
		Status: ride.Status,
		Price:  ride.Price,
	}

	//return ctx.JSON(201, resp)
	return response.WriteSuccess(ctx, http.StatusCreated, resp, "Ride created successfully")
}

func (h *RideHandler) GetByID(ctx *echo.Context) error {
	idParam := ctx.Param("id")
	rideID, err := uuid.Parse(idParam)
	if err != nil {
		return errs.ErrBadRequest.WithMessage("Invalid ride ID").WithRootErr(err)
	}

	ride, err := h.getRideUC.GetByID(ctx.Request().Context(), rideID)
	if err != nil {
		return errs.ErrNotFound.WithMessage(err.Error())
	}

	return response.WriteSuccess(ctx, http.StatusOK, ride, "Ride retrieved successfully")
}

func (h *RideHandler) ListRides(ctx *echo.Context) error {
	filter := domain.RideFilter{
		Limit:  10,
		Offset: 0,
	}

	if s := ctx.QueryParam("status"); s != "" {
		filter.Status = &s
	}

	if v := ctx.QueryParam("rider_id"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			return errs.ErrBadRequest.WithMessage("Invalid rider_id").WithRootErr(err)
		}
		filter.RiderID = &id
	}

	if v := ctx.QueryParam("driver_id"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			return errs.ErrBadRequest.WithMessage("Invalid driver_id").WithRootErr(err)
		}
		filter.DriverID = &id
	}

	if v := ctx.QueryParam("limit"); v != "" {
		limit, err := strconv.Atoi(v)
		if err != nil {
			return errs.ErrBadRequest.WithMessage("Invalid limit").WithRootErr(err)
		}
		if limit > 100 {
			limit = 100
		}
		filter.Limit = int32(limit)
	}

	rides, err := h.getRideUC.ListRides(ctx.Request().Context(), filter)
	if err != nil {
		return errs.ErrInternal.WithMessage(err.Error())
	}

	return response.WriteSuccess(ctx, http.StatusOK, rides, "Rides listed successfully")
}

func (h *RideHandler) AcceptRide(ctx *echo.Context) error {
	rideIDStr := ctx.Param("id")
	rideID, err := uuid.Parse(rideIDStr)
	if err != nil {
		return errs.ErrBadRequest.WithMessage("Invalid ride ID").WithRootErr(err)
	}

	var req acceptRideRequest
	if err := ctx.Bind(&req); err != nil {
		return errs.ErrBadRequest.WithMessage("Failed to bind accept ride request").WithRootErr(err)
	}
	if err := ctx.Validate(&req); err != nil {
		return errs.ErrBadRequest.WithMessage("Validation failed: " + err.Error()).WithRootErr(err)
	}

	driverID, err := uuid.Parse(req.DriverID)
	ride, err := h.acceptRideUC.Execute(ctx.Request().Context(), rideID, driverID)
	if err != nil {
		return err
	}

	return response.WriteSuccess(ctx, http.StatusOK, ride, "Ride accepted successfully")
}

func (h *RideHandler) UpdateStatus(ctx *echo.Context) error {
	rideIDStr := ctx.Param("id")
	rideID, err := uuid.Parse(rideIDStr)
	if err != nil {
		return errs.ErrInvalidInput.WithMessage("Invalid ride ID")
	}

	var req updateStatusRequest
	if err := ctx.Bind(&req); err != nil {
		return errs.ErrInvalidInput.WithRootErr(err)
	}
	if err := ctx.Validate(&req); err != nil {
		return errs.ErrInvalidInput.WithRootErr(err)
	}

	ride, err := h.updateRideUC.UpdateStatus(ctx.Request().Context(), rideID, req.Status)
	if err != nil {
		return err
	}

	return response.WriteSuccess(ctx, http.StatusOK, ride, "Ride status updated")
}

func (h *RideHandler) UpdateTripStatus(ctx *echo.Context) error {
	rideIDStr := ctx.Param("id")
	rideID, err := uuid.Parse(rideIDStr)
	if err != nil {
		return errs.ErrInvalidInput.WithMessage("Invalid ride ID")
	}

	var req updateTripRequest
	if err := ctx.Bind(&req); err != nil {
		return errs.ErrInvalidInput.WithRootErr(err)
	}
	if err := ctx.Validate(&req); err != nil {
		return errs.ErrInvalidInput.WithRootErr(err)
	}

	driverID, _ := uuid.Parse(req.DriverID)
	status := domain.RideStatus(req.Status)

	ride, err := h.updateTripUC.Execute(ctx.Request().Context(), rideID, driverID, status)
	if err != nil {
		return err
	}

	return response.WriteSuccess(ctx, http.StatusOK, ride, "Trip status updated successfully")
}
