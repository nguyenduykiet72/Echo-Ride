package ws

import (
	"bytes"
	"context"
	"echo-ride/services/location-service/internal/application"
	"encoding/json"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512
)

type Client struct {
	Hub     *Hub
	UserID  uuid.UUID
	Conn    *websocket.Conn
	Send    chan []byte
	Batcher *application.LocationBatcher
}

func (c *Client) readPump() {
	defer func() {
		c.Hub.Unregister <- c
		c.Conn.Close()
	}()

	c.Conn.SetReadLimit(maxMessageSize)
	c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Println("read error:", err)
			}
			break
		}

		var rawMsg struct {
			Type string          `json:"type"`
			Data json.RawMessage `json:"data"`
		}

		if err := json.Unmarshal(message, &rawMsg); err != nil {
			log.Printf("Invalid base message format from user %s: %s", c.UserID.String(), string(message))
			continue
		}

		switch rawMsg.Type {
		case "DRIVER_LOCATION_SYNC":
			var payload struct {
				RideID string  `json:"ride_id"`
				Lat    float64 `json:"lat"`
				Lng    float64 `json:"lng"`
			}
			if err := json.Unmarshal(rawMsg.Data, &payload); err != nil {
				log.Printf("Invalid location payload from driver %s", c.UserID.String())
				continue
			}

			if payload.RideID != "" {
				locMsg := LocationUpdateMsg{
					RideID:   payload.RideID,
					DriverID: c.UserID.String(),
					Lat:      payload.Lat,
					Lng:      payload.Lng,
				}
				c.Hub.BroadcastLocationToRedis(context.Background(), locMsg)
			}
		case "SUBSCRIBE_RIDE":
			var payload struct {
				RideID string `json:"ride_id"`
			}
			if err := json.Unmarshal(rawMsg.Data, &payload); err != nil || payload.RideID == "" {
				log.Printf("Invalid subscribe payload from rider %s", c.UserID.String())
				continue
			}
		default:
			log.Printf("Unknown message type '%s' from user %s", rawMsg.Type, c.UserID.String())
		}
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			n := len(c.Send)
			for i := 0; i < n; i++ {
				w.Write(bytes.TrimSpace(<-c.Send))
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
