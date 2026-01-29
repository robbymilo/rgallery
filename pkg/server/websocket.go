package server

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/coder/websocket"
	"github.com/robbymilo/rgallery/pkg/notify"
	"github.com/robbymilo/rgallery/pkg/scanner"
	"github.com/robbymilo/rgallery/pkg/types"
)

// ServeWebSocket handles WebSocket connections for real-time notifications
func ServeWebSocket(w http.ResponseWriter, r *http.Request, c Conf) {
	// Accept WebSocket connection
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true, // Allow connections from any origin in dev mode
		OriginPatterns:     []string{"*"},
	})
	if err != nil {
		c.Logger.Error("failed to accept websocket", "error", err)
		return
	}
	defer conn.Close(websocket.StatusNormalClosure, "connection closed")

	// Get user from context
	userKey := r.Context().Value(types.UserKey{})
	var username string
	if user, ok := userKey.(types.UserKey); ok {
		username = user.UserName
	}

	// Add this connection to the notify package
	notify.AddWebSocketConnection(conn, username)
	defer notify.RemoveWebSocketConnection(conn)

	// Send initial status
	initialStatus := scanner.IsScanInProgress()
	statusMsg := notify.Notice{
		Message: "connected",
		Status:  getStatusString(initialStatus),
	}
	statusJSON, _ := json.Marshal(statusMsg)

	ctx := context.Background()
	err = conn.Write(ctx, websocket.MessageText, statusJSON)
	if err != nil {
		c.Logger.Error("failed to write initial status", "error", err)
		return
	}

	c.Logger.Info("websocket client connected")

	// Keep connection alive with periodic pings
	pingTicker := time.NewTicker(30 * time.Second)
	defer pingTicker.Stop()

	// Context for handling shutdown
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	// Read loop to detect client disconnection
	go func() {
		for {
			_, _, err := conn.Read(ctx)
			if err != nil {
				c.Logger.Debug("websocket read error, closing connection", "error", err)
				cancel()
				return
			}
		}
	}()

	// Main loop - keep connection alive
	for {
		select {
		case <-pingTicker.C:
			// Send ping to keep connection alive
			pingCtx, pingCancel := context.WithTimeout(ctx, 5*time.Second)
			err := conn.Ping(pingCtx)
			pingCancel()
			if err != nil {
				c.Logger.Debug("websocket ping failed, client disconnected", "error", err)
				return
			}
		case <-ctx.Done():
			c.Logger.Info("websocket client disconnected")
			return
		}
	}
}

func getStatusString(scanInProgress bool) string {
	if scanInProgress {
		return "scanning"
	}
	return "ok"
}
