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

type Client struct {
	conn   *websocket.Conn
	send   chan []byte
	roomID string // New field to track the client's room
}

type Hub struct {
	clients   map[*Client]bool
	rooms     map[string][]*Client // New map to track clients by room
	broadcast chan struct {
		client  *Client
		message []byte
	} // Modified to include client
	register   chan *Client
	unregister chan *Client
	mutex      sync.Mutex
}

func NewHub() *Hub {
	return &Hub{
		clients: make(map[*Client]bool),
		rooms:   make(map[string][]*Client), // Initialize rooms map
		broadcast: make(chan struct {
			client  *Client
			message []byte
		}),
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
			h.rooms[client.roomID] = append(h.rooms[client.roomID], client)
			if len(h.rooms[client.roomID]) > 2 { // Limit to 2 clients per room
				client.conn.WriteMessage(websocket.TextMessage, []byte("Room is full"))
				delete(h.clients, client)
				h.rooms[client.roomID] = h.rooms[client.roomID][:2]
				client.conn.Close()
			} else {
				h.notifyRoom(client.roomID, []byte("User joined the room"))
			}
			h.mutex.Unlock()
			fmt.Printf("Client connected to room %s. Total: %d\n", client.roomID, len(h.clients))

		case client := <-h.unregister:
			h.mutex.Lock()
			if _, ok := h.clients[client]; ok {
				h.removeFromRoom(client) // New helper function
				close(client.send)
				delete(h.clients, client)
				h.notifyRoom(client.roomID, []byte("User left the room"))
			}
			h.mutex.Unlock()
			fmt.Printf("Client disconnected from room %s. Total: %d\n", client.roomID, len(h.clients))

		case msg := <-h.broadcast:
			h.mutex.Lock()
			for _, client := range h.rooms[msg.client.roomID] {
				if client != msg.client { // Send only to the other client in the room
					select {
					case client.send <- msg.message:
					default:
						h.removeFromRoom(client)
						close(client.send)
						delete(h.clients, client)
					}
				}
			}
			h.mutex.Unlock()
		}
	}
}

func (h *Hub) notifyRoom(roomID string, message []byte) {
	for _, client := range h.rooms[roomID] {
		select {
		case client.send <- message:
		default:
		}
	}
}

func (h *Hub) removeFromRoom(client *Client) {
	clients := h.rooms[client.roomID]
	for i, c := range clients {
		if c == client {
			h.rooms[client.roomID] = append(clients[:i], clients[i+1:]...)
			break
		}
	}
	if len(h.rooms[client.roomID]) == 0 {
		delete(h.rooms, client.roomID)
	}
}

func (h *Hub) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	roomID := r.URL.Query().Get("room") // Get room ID from query parameter
	if roomID == "" {
		http.Error(w, "Room ID required", http.StatusBadRequest)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Upgrade error:", err)
		return
	}

	client := &Client{conn: conn, send: make(chan []byte, 256), roomID: roomID}
	h.register <- client

	defer func() {
		h.unregister <- client
		conn.Close()
	}()

	go func() {
		defer conn.Close()
		for message := range client.send {
			err := conn.WriteMessage(websocket.TextMessage, message)
			if err != nil {
				return
			}
		}
	}()

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			return
		}
		h.broadcast <- struct {
			client  *Client
			message []byte
		}{client, message}
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
