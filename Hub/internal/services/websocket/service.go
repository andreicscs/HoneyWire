package websocket

import (
	"context"
	"log"
	"net/http"
	"sync"
	"time"

	gorillaws "github.com/gorilla/websocket"
)

var upgrader = gorillaws.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type Service struct {
	clients   map[*gorillaws.Conn]*sync.Mutex
	clientsMu sync.Mutex
}

func NewService() *Service {
	return &Service{
		clients: make(map[*gorillaws.Conn]*sync.Mutex),
	}
}

func (s *Service) HandleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[ERROR] WS Upgrade failed: %v\n", err)
		return
	}

	writeMu := &sync.Mutex{}

	s.clientsMu.Lock()
	s.clients[conn] = writeMu
	s.clientsMu.Unlock()

	const pingPeriod = 20 * time.Second
	const readWait = 25 * time.Second // Must be greater than pingPeriod

	conn.SetReadDeadline(time.Now().Add(readWait))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(readWait))
		return nil
	})

	// Goroutine for keeping the connection alive by sending periodic pings
	go func() {
		ticker := time.NewTicker(pingPeriod)
		defer func() {
			ticker.Stop()
			conn.Close()
		}()

		for range ticker.C {
			// Send a native control frame Ping (Opcode 0x9)
			writeMu.Lock()
			err := conn.WriteMessage(gorillaws.PingMessage, nil)
			writeMu.Unlock()
			if err != nil {
				return
			}
		}
	}()

	go func() {
		defer func() {
			s.clientsMu.Lock()
			delete(s.clients, conn)
			s.clientsMu.Unlock()
			conn.Close()
		}()
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				break
			}
		}
	}()
}

func (s *Service) Broadcast(msgType string, payload interface{}) {
	// Snapshot clients to avoid holding the global lock during network I/O
	s.clientsMu.Lock()
	type clientEntry struct {
		conn *gorillaws.Conn
		mu   *sync.Mutex
	}
	snapshot := make([]clientEntry, 0, len(s.clients))
	for conn, mu := range s.clients {
		snapshot = append(snapshot, clientEntry{conn, mu})
	}
	s.clientsMu.Unlock()

	var deadClients []*gorillaws.Conn
	for _, c := range snapshot {
		c.mu.Lock()
		err := c.conn.WriteJSON(map[string]interface{}{
			"type":    msgType,
			"payload": payload,
		})
		c.mu.Unlock()
		if err != nil {
			deadClients = append(deadClients, c.conn)
		}
	}

	// Safely prune any connections that failed during broadcast
	if len(deadClients) > 0 {
		s.clientsMu.Lock()
		for _, c := range deadClients {
			delete(s.clients, c)
		}
		s.clientsMu.Unlock()

		// Close dead connections outside the lock to prevent internal deadlocks
		for _, c := range deadClients {
			c.Close()
		}
	}
}

func (s *Service) StartChartSyncBroadcaster(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.Broadcast("SYNC_CHARTS", nil)
		}
	}
}
