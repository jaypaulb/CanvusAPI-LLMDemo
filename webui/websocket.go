// Package webui provides the web-based user interface for CanvusLocalLLM.
// This file contains the WebSocketBroadcaster molecule for managing client connections
// and broadcasting real-time updates.
package webui

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// WebSocketBroadcaster is a molecule that manages WebSocket client connections
// and broadcasts messages to all connected clients.
//
// It composes:
//   - Message types (from ws_message.go atoms)
//   - Connection map for client management
//   - Broadcast channel for message distribution
//
// Thread-safe for concurrent client connections and message broadcasting.
type WebSocketBroadcaster struct {
	// clients maps WebSocket connections to their active status
	clients map[*websocket.Conn]clientInfo

	// clientsMu protects concurrent access to the clients map
	clientsMu sync.RWMutex

	// broadcast receives messages to send to all clients
	broadcast chan WSMessage

	// register receives new client connections
	register chan *websocket.Conn

	// unregister receives clients to remove
	unregister chan *websocket.Conn

	// upgrader handles HTTP to WebSocket upgrades
	upgrader websocket.Upgrader

	// pingInterval is how often to send ping messages
	pingInterval time.Duration

	// pongWait is how long to wait for a pong response
	pongWait time.Duration

	// writeWait is the time allowed to write a message
	writeWait time.Duration

	// maxMessageSize is the maximum message size allowed from client
	maxMessageSize int64

	// logger for WebSocket operations
	logger Logger
}

// clientInfo stores metadata about a connected client
type clientInfo struct {
	// connectedAt is when the client connected
	connectedAt time.Time

	// remoteAddr is the client's remote address
	remoteAddr string

	// send is the channel for sending messages to this client
	send chan []byte
}

// Logger interface for WebSocket logging
type Logger interface {
	Printf(format string, v ...interface{})
}

// defaultLogger wraps the standard log package
type defaultLogger struct{}

func (d *defaultLogger) Printf(format string, v ...interface{}) {
	log.Printf("[WebSocket] "+format, v...)
}

// BroadcasterConfig holds configuration for the WebSocketBroadcaster
type BroadcasterConfig struct {
	// PingInterval is how often to send ping messages (default: 30s)
	PingInterval time.Duration

	// PongWait is how long to wait for pong response (default: 60s)
	PongWait time.Duration

	// WriteWait is time allowed to write a message (default: 10s)
	WriteWait time.Duration

	// MaxMessageSize is max message size from client (default: 512 bytes)
	MaxMessageSize int64

	// BroadcastBufferSize is the broadcast channel buffer (default: 256)
	BroadcastBufferSize int

	// ClientSendBufferSize is per-client send buffer (default: 256)
	ClientSendBufferSize int

	// Logger for WebSocket operations (default: standard log)
	Logger Logger
}

// DefaultBroadcasterConfig returns the default configuration
func DefaultBroadcasterConfig() BroadcasterConfig {
	return BroadcasterConfig{
		PingInterval:         30 * time.Second,
		PongWait:             60 * time.Second,
		WriteWait:            10 * time.Second,
		MaxMessageSize:       512,
		BroadcastBufferSize:  256,
		ClientSendBufferSize: 256,
		Logger:               &defaultLogger{},
	}
}

// NewWebSocketBroadcaster creates a new WebSocketBroadcaster with default configuration.
//
// Returns a ready-to-use broadcaster. Call Start() to begin processing messages.
func NewWebSocketBroadcaster() *WebSocketBroadcaster {
	return NewWebSocketBroadcasterWithConfig(DefaultBroadcasterConfig())
}

// NewWebSocketBroadcasterWithConfig creates a new WebSocketBroadcaster with custom configuration.
//
// Parameters:
//   - config: Custom configuration for the broadcaster
//
// Returns a ready-to-use broadcaster. Call Start() to begin processing messages.
func NewWebSocketBroadcasterWithConfig(config BroadcasterConfig) *WebSocketBroadcaster {
	if config.Logger == nil {
		config.Logger = &defaultLogger{}
	}

	return &WebSocketBroadcaster{
		clients:        make(map[*websocket.Conn]clientInfo),
		broadcast:      make(chan WSMessage, config.BroadcastBufferSize),
		register:       make(chan *websocket.Conn),
		unregister:     make(chan *websocket.Conn),
		pingInterval:   config.PingInterval,
		pongWait:       config.PongWait,
		writeWait:      config.WriteWait,
		maxMessageSize: config.MaxMessageSize,
		logger:         config.Logger,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			// CheckOrigin allows connections from any origin (same-origin deployment)
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
	}
}

// Start begins the message broadcasting loop.
//
// This method runs until the context is cancelled. It handles:
//   - Client registration/unregistration
//   - Broadcasting messages to all clients
//   - Periodic ping messages for connection health
//
// Parameters:
//   - ctx: Context for cancellation
func (b *WebSocketBroadcaster) Start(ctx context.Context) {
	pingTicker := time.NewTicker(b.pingInterval)
	defer pingTicker.Stop()

	b.logger.Printf("Broadcaster started")

	for {
		select {
		case <-ctx.Done():
			b.logger.Printf("Broadcaster stopping: context cancelled")
			b.closeAllClients()
			return

		case conn := <-b.register:
			b.addClient(conn)

		case conn := <-b.unregister:
			b.removeClient(conn)

		case message := <-b.broadcast:
			b.broadcastToAll(message)

		case <-pingTicker.C:
			b.sendPingToAll()
		}
	}
}

// HandleConnection handles a new WebSocket connection request.
//
// This is an HTTP handler that upgrades the connection to WebSocket
// and manages the client lifecycle.
//
// Parameters:
//   - w: HTTP response writer
//   - r: HTTP request
func (b *WebSocketBroadcaster) HandleConnection(w http.ResponseWriter, r *http.Request) {
	conn, err := b.upgrader.Upgrade(w, r, nil)
	if err != nil {
		b.logger.Printf("Failed to upgrade connection from %s: %v", r.RemoteAddr, err)
		return
	}

	// Configure connection
	conn.SetReadLimit(b.maxMessageSize)
	conn.SetReadDeadline(time.Now().Add(b.pongWait))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(b.pongWait))
		return nil
	})

	// Register the client
	b.register <- conn

	// Start read pump for this client (handles pong and close)
	// Note: Initial state should be sent via BroadcastMessage or SendInitialState
	// after the client is fully registered
	go b.readPump(conn)
}

// BroadcastMessage sends a message to all connected clients.
//
// This method is non-blocking. Messages are queued for delivery.
// If the broadcast buffer is full, the message is dropped with a warning.
//
// Parameters:
//   - msg: The message to broadcast
func (b *WebSocketBroadcaster) BroadcastMessage(msg WSMessage) {
	select {
	case b.broadcast <- msg:
		// Message queued successfully
	default:
		b.logger.Printf("Warning: broadcast buffer full, dropping message type=%s", msg.Type)
	}
}

// ClientCount returns the current number of connected clients.
//
// Thread-safe.
func (b *WebSocketBroadcaster) ClientCount() int {
	b.clientsMu.RLock()
	defer b.clientsMu.RUnlock()
	return len(b.clients)
}

// Close gracefully shuts down the broadcaster.
//
// This closes all client connections and cleans up resources.
func (b *WebSocketBroadcaster) Close() {
	b.closeAllClients()
}

// addClient registers a new client connection
func (b *WebSocketBroadcaster) addClient(conn *websocket.Conn) {
	b.clientsMu.Lock()
	defer b.clientsMu.Unlock()

	info := clientInfo{
		connectedAt: time.Now(),
		remoteAddr:  conn.RemoteAddr().String(),
		send:        make(chan []byte, 256),
	}
	b.clients[conn] = info

	// Start write pump for this client
	go b.writePump(conn, info.send)

	b.logger.Printf("Client connected: %s (total: %d)", info.remoteAddr, len(b.clients))
}

// removeClient unregisters a client and closes its connection
func (b *WebSocketBroadcaster) removeClient(conn *websocket.Conn) {
	b.clientsMu.Lock()
	defer b.clientsMu.Unlock()

	if info, ok := b.clients[conn]; ok {
		close(info.send)
		delete(b.clients, conn)
		conn.Close()
		b.logger.Printf("Client disconnected: %s (total: %d)", info.remoteAddr, len(b.clients))
	}
}

// broadcastToAll sends a message to all connected clients
func (b *WebSocketBroadcaster) broadcastToAll(msg WSMessage) {
	data, err := json.Marshal(msg)
	if err != nil {
		b.logger.Printf("Failed to marshal broadcast message: %v", err)
		return
	}

	b.clientsMu.RLock()
	defer b.clientsMu.RUnlock()

	for conn, info := range b.clients {
		select {
		case info.send <- data:
			// Message queued
		default:
			// Client send buffer full, close connection
			b.logger.Printf("Client %s send buffer full, closing", info.remoteAddr)
			go func(c *websocket.Conn) {
				b.unregister <- c
			}(conn)
		}
	}
}

// sendToClient sends a message to a specific client
func (b *WebSocketBroadcaster) sendToClient(conn *websocket.Conn, msg WSMessage) {
	data, err := json.Marshal(msg)
	if err != nil {
		b.logger.Printf("Failed to marshal message: %v", err)
		return
	}

	b.clientsMu.RLock()
	info, ok := b.clients[conn]
	b.clientsMu.RUnlock()

	if ok {
		select {
		case info.send <- data:
			// Message queued
		default:
			b.logger.Printf("Client %s send buffer full", info.remoteAddr)
		}
	}
}

// sendPingToAll sends a ping message to all clients for connection health
func (b *WebSocketBroadcaster) sendPingToAll() {
	b.clientsMu.RLock()
	defer b.clientsMu.RUnlock()

	for conn, info := range b.clients {
		conn.SetWriteDeadline(time.Now().Add(b.writeWait))
		if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
			b.logger.Printf("Failed to ping client %s: %v", info.remoteAddr, err)
			go func(c *websocket.Conn) {
				b.unregister <- c
			}(conn)
		}
	}
}

// closeAllClients closes all client connections
func (b *WebSocketBroadcaster) closeAllClients() {
	b.clientsMu.Lock()
	defer b.clientsMu.Unlock()

	for conn, info := range b.clients {
		close(info.send)
		conn.Close()
		delete(b.clients, conn)
	}

	b.logger.Printf("All clients disconnected")
}

// readPump handles incoming messages from a client
// Currently only handles pong messages and close
func (b *WebSocketBroadcaster) readPump(conn *websocket.Conn) {
	defer func() {
		b.unregister <- conn
	}()

	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				b.logger.Printf("Unexpected close error: %v", err)
			}
			break
		}
		// Currently we don't process client messages, just keep connection alive
	}
}

// writePump handles outgoing messages to a client
func (b *WebSocketBroadcaster) writePump(conn *websocket.Conn, send <-chan []byte) {
	defer func() {
		conn.Close()
	}()

	for message := range send {
		conn.SetWriteDeadline(time.Now().Add(b.writeWait))
		if err := conn.WriteMessage(websocket.TextMessage, message); err != nil {
			b.logger.Printf("Write error: %v", err)
			return
		}
	}

	// Send close message when channel is closed
	conn.SetWriteDeadline(time.Now().Add(b.writeWait))
	conn.WriteMessage(websocket.CloseMessage, []byte{})
}

// SendInitialState sends the initial state to a newly connected client.
//
// This should be called after the client connects to provide them with
// the current system state.
//
// Parameters:
//   - conn: The client connection
//   - data: The initial state data
func (b *WebSocketBroadcaster) SendInitialState(conn *websocket.Conn, data InitialData) {
	msg := NewInitialMessage(data)
	b.sendToClient(conn, msg)
}

// BroadcastTaskUpdate broadcasts a task update to all clients.
//
// Convenience method for task_update messages.
func (b *WebSocketBroadcaster) BroadcastTaskUpdate(data TaskUpdateData) {
	b.BroadcastMessage(NewTaskUpdateMessage(data))
}

// BroadcastGPUUpdate broadcasts a GPU metrics update to all clients.
//
// Convenience method for gpu_update messages.
func (b *WebSocketBroadcaster) BroadcastGPUUpdate(data GPUUpdateData) {
	b.BroadcastMessage(NewGPUUpdateMessage(data))
}

// BroadcastCanvasUpdate broadcasts a canvas status update to all clients.
//
// Convenience method for canvas_update messages.
func (b *WebSocketBroadcaster) BroadcastCanvasUpdate(data CanvasUpdateData) {
	b.BroadcastMessage(NewCanvasUpdateMessage(data))
}

// BroadcastSystemStatus broadcasts a system status update to all clients.
//
// Convenience method for system_status messages.
func (b *WebSocketBroadcaster) BroadcastSystemStatus(data SystemStatusData) {
	b.BroadcastMessage(NewSystemStatusMessage(data))
}

// BroadcastError broadcasts an error message to all clients.
//
// Convenience method for error messages.
func (b *WebSocketBroadcaster) BroadcastError(code, message string) {
	b.BroadcastMessage(NewErrorMessage(code, message))
}
