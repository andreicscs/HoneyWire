package websocket

import (
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
	clients   map[*gorillaws.Conn]bool
	clientsMu sync.Mutex
}

func NewService() *Service {
	return &Service{
		clients: make(map[*gorillaws.Conn]bool),
	}
}

func (s *Service) HandleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[ERROR] WS Upgrade failed: %v\n", err)
		return
	}

	s.clientsMu.Lock()
	s.clients[conn] = true
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
			if err := conn.WriteMessage(gorillaws.PingMessage, nil); err != nil {
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
	var deadClients []*gorillaws.Conn
	s.clientsMu.Lock()
	for client := range s.clients {
		err := client.WriteJSON(map[string]interface{}{
			"type":    msgType,
			"payload": payload,
		})
		if err != nil {
			deadClients = append(deadClients, client)
		}
	}
	for _, c := range deadClients {
		delete(s.clients, c)
		c.Close()
	}
	s.clientsMu.Unlock()
}
