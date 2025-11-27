package http

import (
	"net/http"
	"sync"

	"github.com/PhucNguyen204/vcs-infrastructure-provisioning-service/dto"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins in development
	},
}

type WebSocketHandler struct {
	clients    map[*websocket.Conn]bool
	broadcast  chan dto.InfrastructureStatusUpdate
	register   chan *websocket.Conn
	unregister chan *websocket.Conn
	mu         sync.RWMutex
	logger     interface {
		Info(msg string, fields ...zap.Field)
		Error(msg string, fields ...zap.Field)
	}
}

func NewWebSocketHandler(logger interface {
	Info(msg string, fields ...zap.Field)
	Error(msg string, fields ...zap.Field)
}) *WebSocketHandler {
	return &WebSocketHandler{
		clients:    make(map[*websocket.Conn]bool),
		broadcast:  make(chan dto.InfrastructureStatusUpdate, 256),
		register:   make(chan *websocket.Conn),
		unregister: make(chan *websocket.Conn),
		logger:     logger,
	}
}

func (h *WebSocketHandler) HandleWebSocket(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.logger.Error("websocket upgrade failed", zap.Error(err))
		return
	}

	h.register <- conn

	go h.writePump(conn)
	h.readPump(conn)
}

func (h *WebSocketHandler) Start() {
	for {
		select {
		case conn := <-h.register:
			h.mu.Lock()
			h.clients[conn] = true
			h.mu.Unlock()
			h.logger.Info("websocket client connected", zap.Int("total_clients", len(h.clients)))

		case conn := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[conn]; ok {
				delete(h.clients, conn)
				conn.Close()
			}
			h.mu.Unlock()
			h.logger.Info("websocket client disconnected", zap.Int("total_clients", len(h.clients)))

		case message := <-h.broadcast:
			h.mu.RLock()
			for conn := range h.clients {
				err := conn.WriteJSON(message)
				if err != nil {
					h.logger.Error("websocket write error", zap.Error(err))
					h.unregister <- conn
				}
			}
			h.mu.RUnlock()
		}
	}
}

func (h *WebSocketHandler) BroadcastUpdate(update dto.InfrastructureStatusUpdate) {
	select {
	case h.broadcast <- update:
	default:
		h.logger.Error("broadcast channel full, dropping message")
	}
}

func (h *WebSocketHandler) readPump(conn *websocket.Conn) {
	defer func() {
		h.unregister <- conn
		conn.Close()
	}()

	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				h.logger.Error("websocket read error", zap.Error(err))
			}
			break
		}
	}
}

func (h *WebSocketHandler) writePump(conn *websocket.Conn) {
	defer conn.Close()

	for {
		message, ok := <-h.broadcast
		if !ok {
			conn.WriteMessage(websocket.CloseMessage, []byte{})
			return
		}

		if err := conn.WriteJSON(message); err != nil {
			h.logger.Error("websocket write error", zap.Error(err))
			return
		}
	}
}

