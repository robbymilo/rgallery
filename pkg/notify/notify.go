package notify

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/coder/websocket"
	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitex"
)

var (
	subscribers   = make(map[chan string]struct{})
	wsConnections = make(map[string][]*websocket.Conn)
	connToUser    = make(map[*websocket.Conn]string)
	mu            sync.Mutex
	wsMu          sync.Mutex
)

type Notice struct {
	Message string `json:"message"`
	Status  string `json:"status"`
}

type Notification struct {
	ID        int64  `json:"id"`
	Username  string `json:"username,omitempty"`
	Message   string `json:"message"`
	IsRead    bool   `json:"is_read"`
	CreatedAt string `json:"created_at"`
}

// Function to add a subscriber (client waiting for an event)
func AddSubscriber() chan string {

	ch := make(chan string, 1)
	mu.Lock()
	subscribers[ch] = struct{}{}
	mu.Unlock()
	return ch
}

// Function to remove a subscriber
func RemoveSubscriber(ch chan string) {
	mu.Lock()
	_, exists := subscribers[ch]
	if exists {
		delete(subscribers, ch)
		close(ch) // Close only if it exists
	}
	mu.Unlock()
}

// AddWebSocketConnection adds a WebSocket connection to receive notifications
func AddWebSocketConnection(conn *websocket.Conn, username string) {
	wsMu.Lock()
	wsConnections[username] = append(wsConnections[username], conn)
	connToUser[conn] = username
	wsMu.Unlock()
}

// RemoveWebSocketConnection removes a WebSocket connection
func RemoveWebSocketConnection(conn *websocket.Conn) {
	wsMu.Lock()
	username, ok := connToUser[conn]
	if ok {
		conns := wsConnections[username]
		for i, c := range conns {
			if c == conn {
				wsConnections[username] = append(conns[:i], conns[i+1:]...)
				break
			}
		}
		if len(wsConnections[username]) == 0 {
			delete(wsConnections, username)
		}
		delete(connToUser, conn)
	}
	wsMu.Unlock()
}

// SendToAllUsers sends a notification to all users
func SendToAllUsers(conn *sqlite.Conn, message string) error {
	// Get all users
	var usernames []string
	err := sqlitex.Execute(conn, "SELECT username FROM users", &sqlitex.ExecOptions{
		ResultFunc: func(stmt *sqlite.Stmt) error {
			usernames = append(usernames, stmt.ColumnText(0))
			return nil
		},
	})
	if err != nil {
		return err
	}

	createdAt := time.Now().UTC().Format("2006-01-02T15:04:05Z")
	stmt, err := conn.Prepare("INSERT INTO notifications (message, created_at) VALUES (?, ?)")
	if err != nil {
		return err
	}
	stmt.BindText(1, message)
	stmt.BindText(2, createdAt)
	_, err = stmt.Step()
	stmt.Finalize()
	if err != nil {
		return err
	}
	id := conn.LastInsertRowID()

	// Broadcast to all connected websocket clients
	payload := Notification{
		ID:        id,
		Message:   message,
		IsRead:    false,
		CreatedAt: createdAt,
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	wsMu.Lock()
	defer wsMu.Unlock()

	for _, username := range usernames {
		if conns, ok := wsConnections[username]; ok {
			for _, conn := range conns {
				go func(c *websocket.Conn) {
					ctx := context.Background()
					_ = c.Write(ctx, websocket.MessageText, payloadJSON)
				}(conn)
			}
		}
	}

	return nil
}
