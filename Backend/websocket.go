package main

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow connections from any origin in development
	},
}

type Hub struct {
	clients    map[*websocket.Conn]bool
	broadcast  chan []byte
	register   chan *websocket.Conn
	unregister chan *websocket.Conn
	mutex      sync.RWMutex
}

type WebSocketMessage struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

var hub = &Hub{
	clients:    make(map[*websocket.Conn]bool),
	broadcast:  make(chan []byte),
	register:   make(chan *websocket.Conn),
	unregister: make(chan *websocket.Conn),
}

func (h *Hub) run() {
	for {
		select {
		case conn := <-h.register:
			h.mutex.Lock()
			h.clients[conn] = true
			h.mutex.Unlock()
			log.Printf("Client connected. Total clients: %d", len(h.clients))

		case conn := <-h.unregister:
			h.mutex.Lock()
			if _, ok := h.clients[conn]; ok {
				delete(h.clients, conn)
				conn.Close()
			}
			h.mutex.Unlock()
			log.Printf("Client disconnected. Total clients: %d", len(h.clients))

		case message := <-h.broadcast:
			h.mutex.RLock()
			for conn := range h.clients {
				err := conn.WriteMessage(websocket.TextMessage, message)
				if err != nil {
					log.Printf("Error writing message: %v", err)
					delete(h.clients, conn)
					conn.Close()
				}
			}
			h.mutex.RUnlock()
		}
	}
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w)
	
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	hub.register <- conn

	// Handle client disconnection
	go func() {
		defer func() {
			hub.unregister <- conn
		}()

		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				break
			}
		}
	}()
}

// BroadcastPromptCreated broadcasts when a prompt is successfully created
func BroadcastPromptCreated(prompt interface{}) {
	message := WebSocketMessage{
		Type: "prompt_created",
		Data: prompt,
	}
	
	data, err := json.Marshal(message)
	if err != nil {
		log.Printf("Error marshaling prompt created message: %v", err)
		return
	}
	
	select {
	case hub.broadcast <- data:
	default:
		log.Printf("Broadcast channel full, dropping message")
	}
}

// BroadcastPromptUpdated broadcasts when a prompt is successfully updated
func BroadcastPromptUpdated(prompt interface{}) {
	message := WebSocketMessage{
		Type: "prompt_updated",
		Data: prompt,
	}
	
	data, err := json.Marshal(message)
	if err != nil {
		log.Printf("Error marshaling prompt updated message: %v", err)
		return
	}
	
	select {
	case hub.broadcast <- data:
	default:
		log.Printf("Broadcast channel full, dropping message")
	}
}

// BroadcastJobCreated broadcasts when a job is successfully created
func BroadcastJobCreated(job interface{}) {
	message := WebSocketMessage{
		Type: "job_created",
		Data: job,
	}
	
	data, err := json.Marshal(message)
	if err != nil {
		log.Printf("Error marshaling job created message: %v", err)
		return
	}
	
	select {
	case hub.broadcast <- data:
	default:
		log.Printf("Broadcast channel full, dropping message")
	}
}

// BroadcastJobUpdated broadcasts when a job is successfully updated
func BroadcastJobUpdated(job interface{}) {
	message := WebSocketMessage{
		Type: "job_updated",
		Data: job,
	}
	
	data, err := json.Marshal(message)
	if err != nil {
		log.Printf("Error marshaling job updated message: %v", err)
		return
	}
	
	select {
	case hub.broadcast <- data:
	default:
		log.Printf("Broadcast channel full, dropping message")
	}
}

// handleBroadcast handles HTTP requests to broadcast WebSocket messages
func handleBroadcast(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w)
	if r.Method == http.MethodOptions {
		handleCORS(w, "POST, OPTIONS")
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST allowed", http.StatusMethodNotAllowed)
		return
	}

	var message WebSocketMessage
	if err := json.NewDecoder(r.Body).Decode(&message); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	data, err := json.Marshal(message)
	if err != nil {
		http.Error(w, "Error marshaling message", http.StatusInternalServerError)
		return
	}

	select {
	case hub.broadcast <- data:
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "broadcasted"})
	default:
		http.Error(w, "Broadcast channel full", http.StatusServiceUnavailable)
	}
}

func init() {
	go hub.run()
}