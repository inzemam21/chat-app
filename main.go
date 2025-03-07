package main

import (
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

// Client represents a connected user
type Client struct {
	conn *websocket.Conn
	send chan []byte // Channel to send messages to this client
}

// Hub manages all connected clients
type Hub struct {
	clients    map[*Client]bool // Map of connected clients
	broadcast  chan []byte      // Channel for broadcasting messages
	register   chan *Client     // Channel for new client connections
	unregister chan *Client     // Channel for client disconnections
	mutex      sync.Mutex       // For thread-safe map access
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mutex.Lock()
			h.clients[client] = true
			h.mutex.Unlock()
			fmt.Printf("Client connected. Total: %d\n", len(h.clients))

		case client := <-h.unregister:
			h.mutex.Lock()
			if _, ok := h.clients[client]; ok {
				close(client.send)
				delete(h.clients, client)
			}
			h.mutex.Unlock()
			fmt.Printf("Client disconnected. Total: %d\n", len(h.clients))

		case message := <-h.broadcast:
			h.mutex.Lock()
			for client := range h.clients {
				select {
				case client.send <- message:
				default: // If client isn't receiving, remove it
					close(client.send)
					delete(h.clients, client)
				}
			}
			h.mutex.Unlock()
		}
	}
}

func (h *Hub) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Upgrade error:", err)
		return
	}

	client := &Client{conn: conn, send: make(chan []byte, 256)}
	h.register <- client

	defer func() {
		h.unregister <- client
		conn.Close()
	}()

	// Handle sending messages to the client
	go func() {
		defer conn.Close()
		for message := range client.send {
			err := conn.WriteMessage(websocket.TextMessage, message)
			if err != nil {
				return
			}
		}
	}()

	// Handle receiving messages from the client
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			return
		}
		h.broadcast <- message
	}
}

func main() {
	hub := NewHub()
	go hub.Run()

	http.Handle("/", http.FileServer(http.Dir("./static")))
	http.HandleFunc("/ws", hub.handleWebSocket)

	fmt.Println("Server starting on :8080...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
