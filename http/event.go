package http

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Websocket metrics.
var (
	websocketConnections = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "wtf_http_websocket_connections",
		Help: "Total number of connected websocket users",
	})
)

// registerEventRoutes is a helper function to register event routes.
func (s *Server) registerEventRoutes(r *mux.Router) {
	r.HandleFunc("/events", s.handleEvents)
}

// handleEvents handles the "GET /events" route. This route provides real-time
// event notification over Websockets.
func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	websocketConnections.Inc()
	defer websocketConnections.Dec()

	// Upgrade HTTP connection to use websockets.
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		LogError(r, err)
		return
	}

	ctx, cancel := context.WithCancel(r.Context())
	r = r.WithContext(ctx)
	conn.SetCloseHandler(func(code int, text string) error {
		cancel()
		return nil
	})

	// We defer the connection close to ensure it is disconnected when we
	// exit this function. This can occur if the HTTP request disconnects or
	// if the subscription from the event service closes.
	defer conn.Close()

	// Ignore all incoming messages.
	go ignoreWebSocketReaders(conn)

	// Subscribe to all events for the current user.
	sub, err := s.EventService.Subscribe(r.Context())
	if err != nil {
		LogError(r, err)
		return
	}
	defer sub.Close()

	// Stream all events to outgoing websocket writer.
	for {
		select {
		case <-r.Context().Done():
			return // disconnect when HTTP connection disconnects

		case event, ok := <-sub.C():
			// If subscription is closed then exit.
			if !ok {
				return
			}

			// Marshal event data to JSON.
			buf, err := json.Marshal(event)
			if err != nil {
				LogError(r, err)
				return
			}

			// Write JSON data out to the websocket connection.
			if err := conn.WriteMessage(websocket.TextMessage, buf); err != nil {
				LogError(r, err)
				return
			}
		}
	}
}

// ignoreWebSocketReaders ignores all incoming WS messages on conn.
// This is required by the underlying library if we don't care about sent messages.
//
// This implementation was borrowed from gorilla's docs:
// https://godoc.org/github.com/gorilla/websocket#hdr-Control_Messages
func ignoreWebSocketReaders(conn *websocket.Conn) {
	for {
		if _, _, err := conn.NextReader(); err != nil {
			conn.Close()
			return
		}
	}
}

// upgrader is used to upgrade an HTTP connection to a Websocket connection.
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}
