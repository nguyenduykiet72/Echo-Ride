package ws

import (
	"echo-ride/pkg/errs"
	"echo-ride/services/location-service/internal/application"
	"net/http"
	"sync"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v5"
	"go.uber.org/zap"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferPool: &sync.Pool{},
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type Handler struct {
	hub     *Hub
	batcher *application.LocationBatcher
	logger  *zap.Logger
}

func NewHandler(hub *Hub, batcher *application.LocationBatcher, logger *zap.Logger) *Handler {
	return &Handler{
		hub:     hub,
		batcher: batcher,
		logger:  logger,
	}
}

func (h *Handler) ServeWS(ctx *echo.Context) error {
	driverIDStr := ctx.QueryParam("driver_id")
	driverID, err := uuid.Parse(driverIDStr)
	if err != nil {
		h.logger.Error("Invalid driver_id", zap.String("driver_id", driverIDStr), zap.Error(err))
		return errs.ErrInvalidDriverID
	}

	conn, err := upgrader.Upgrade(ctx.Response(), ctx.Request(), nil)
	if err != nil {
		h.logger.Error("Failed to upgrade to websocket", zap.Error(err))
		return errs.ErrWebsocketUpgradeFailed
	}

	client := &Client{
		Hub:      h.hub,
		DriverID: driverID,
		Conn:     conn,
		Send:     make(chan []byte, 256),
		Batcher:  h.batcher,
	}

	client.Hub.Register <- client

	go client.writePump()
	go client.readPump()

	return nil
}
