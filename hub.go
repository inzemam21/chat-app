package main

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

type Client struct {
	conn     *websocket.Conn
	send     chan []byte
	roomID   string
	username string
}

type Message struct {
	Timestamp string
	Content   string
}

type Hub struct {
	clients   map[*Client]bool
	rooms     map[string][]*Client
	messages  map[string][]Message
	broadcast chan struct {
		client  *Client
		message []byte
	}
	typing chan struct {
		client   *Client
		isTyping bool
	}
	register   chan *Client
	unregister chan *Client
	mutex      sync.Mutex
}

func NewHub() *Hub {
	return &Hub{
		clients:  make(map[*Client]bool),
		rooms:    make(map[string][]*Client),
		messages: make(map[string][]Message),
		broadcast: make(chan struct {
			client  *Client
			message []byte
		}),
		typing: make(chan struct {
			client   *Client
			isTyping bool
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
			if len(h.rooms[client.roomID]) > 2 {
				client.conn.WriteMessage(websocket.TextMessage, []byte("Room is full"))
				delete(h.clients, client)
				h.rooms[client.roomID] = h.rooms[client.roomID][:2]
				client.conn.Close()
			} else {
				for _, msg := range h.messages[client.roomID] {
					payload := fmt.Sprintf("%s|%s", msg.Timestamp, msg.Content)
					client.send <- []byte(payload)
				}
				h.notifyRoom(client, []byte(fmt.Sprintf("system:%s joined the room", client.username)))
				h.broadcastStatus(client.roomID)
			}
			h.mutex.Unlock()
			fmt.Printf("%s connected to room %s. Total: %d\n", client.username, client.roomID, len(h.clients))

		case client := <-h.unregister:
			h.mutex.Lock()
			if _, ok := h.clients[client]; ok {
				h.removeFromRoom(client)
				close(client.send)
				delete(h.clients, client)
				h.notifyRoom(client, []byte(fmt.Sprintf("system:%s left the room", client.username)))
				h.broadcastStatus(client.roomID)
			}
			h.mutex.Unlock()
			fmt.Printf("%s disconnected from room %s. Total: %d\n", client.username, client.roomID, len(h.clients))

		case msg := <-h.broadcast:
			h.mutex.Lock()
			timestamp := time.Now().Format("15:04:05")
			payload := fmt.Sprintf("%s|%s: %s", timestamp, msg.client.username, string(msg.message))
			h.messages[msg.client.roomID] = append(h.messages[msg.client.roomID], Message{
				Timestamp: timestamp,
				Content:   fmt.Sprintf("%s: %s", msg.client.username, string(msg.message)),
			})
			for _, client := range h.rooms[msg.client.roomID] {
				if client != msg.client {
					select {
					case client.send <- []byte(payload):
					default:
						h.removeFromRoom(client)
						close(client.send)
						delete(h.clients, client)
					}
				}
			}
			h.mutex.Unlock()

		case typingMsg := <-h.typing:
			h.mutex.Lock()
			message := []byte("typing:0")
			if typingMsg.isTyping {
				message = []byte(fmt.Sprintf("typing:1:%s", typingMsg.client.username))
			}
			for _, client := range h.rooms[typingMsg.client.roomID] {
				if client != typingMsg.client {
					select {
					case client.send <- message:
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

func (h *Hub) notifyRoom(sender *Client, message []byte) {
	for _, client := range h.rooms[sender.roomID] {
		if client != sender { // Exclude the sender
			select {
			case client.send <- message:
			default:
			}
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

func (h *Hub) broadcastStatus(roomID string) {
	clients := h.rooms[roomID]
	if len(clients) == 0 {
		return
	}
	for _, client := range clients {
		var statusMsg string
		if len(clients) == 1 {
			statusMsg = "status:Other:Offline"
		} else {
			for _, other := range clients {
				if other != client {
					statusMsg = fmt.Sprintf("status:%s:Online", other.username)
					break
				}
			}
		}
		fmt.Println("Broadcasting to", client.username, ":", statusMsg)
		select {
		case client.send <- []byte(statusMsg):
		default:
		}
	}
}

func (h *Hub) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println("WebSocket upgrade error:", err)
		return
	}

	roomID := r.URL.Query().Get("room")
	username := r.URL.Query().Get("username")
	if roomID == "" || username == "" {
		conn.Close()
		return
	}

	client := &Client{
		conn:     conn,
		send:     make(chan []byte, 256),
		roomID:   roomID,
		username: username,
	}

	h.register <- client

	defer func() {
		h.unregister <- client
		conn.Close()
	}()

	go func() {
		defer func() {
			h.unregister <- client
		}()
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					fmt.Println("WebSocket read error:", err)
				}
				return
			}
			if string(message) == "typing:1" {
				h.typing <- struct {
					client   *Client
					isTyping bool
				}{client, true}
			} else if string(message) == "typing:0" {
				h.typing <- struct {
					client   *Client
					isTyping bool
				}{client, false}
			} else {
				h.broadcast <- struct {
					client  *Client
					message []byte
				}{client, message}
			}
		}
	}()

	for message := range client.send {
		err := conn.WriteMessage(websocket.TextMessage, message)
		if err != nil {
			fmt.Println("WebSocket write error:", err)
			return
		}
	}
}
