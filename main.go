package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	hub := NewHub()
	go hub.Run()

	http.Handle("/", http.FileServer(http.Dir("./static")))
	http.HandleFunc("/ws", hub.HandleWebSocket)

	fmt.Println("Server starting on :8080...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
