package ws

import (
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type Hub struct {
	Clients    map[uuid.UUID]*Client
	Register   chan *Client
	Unregister chan *Client
	logger     *zap.Logger
}

func NewHub(logger *zap.Logger) *Hub {
	return &Hub{
		Clients:    make(map[uuid.UUID]*Client),
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
		logger:     logger,
	}
}

func (h *Hub) Run() {
	h.logger.Info("Websocket Hub is running...")
	for {
		select {
		case client := <-h.Register:
			h.Clients[client.DriverID] = client
			h.logger.Info("Client registered", zap.String("driver_id", client.DriverID.String()), zap.Int("total_clients", len(h.Clients)))

		case client := <-h.Unregister:
			if _, ok := h.Clients[client.DriverID]; ok {
				delete(h.Clients, client.DriverID)
				close(client.Send)
				h.logger.Info("Client unregistered", zap.String("driver_id", client.DriverID.String()), zap.Int("total_clients", len(h.Clients)))
			}
		}
	}
}
