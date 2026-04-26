package ws

import (
	"echo-ride/pkg/errs"
	"echo-ride/services/location-service/internal/application"
	"net/http"
	"sync"

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
	hub       *Hub
	batcher   *application.LocationBatcher
	logger    *zap.Logger
	jwtSecret string
}

func NewHandler(hub *Hub, batcher *application.LocationBatcher, jwtSecret string, logger *zap.Logger) *Handler {
	return &Handler{
		hub:       hub,
		batcher:   batcher,
		logger:    logger,
		jwtSecret: jwtSecret,
	}
}

func (h *Handler) ServeWS(ctx *echo.Context) error {
	conn, err := upgrader.Upgrade(ctx.Response(), ctx.Request(), nil)
	if err != nil {
		h.logger.Error("Failed to upgrade to websocket", zap.Error(err))
		return errs.ErrWebsocketUpgradeFailed
	}

	client := &Client{
		Hub: h.hub,
		//UserID,
		Conn:            conn,
		Send:            make(chan []byte, 256),
		Batcher:         h.batcher,
		jwtSecret:       h.jwtSecret,
		isAuthenticated: false,
	}
	
	go client.writePump()
	go client.readPump()

	return nil
}
