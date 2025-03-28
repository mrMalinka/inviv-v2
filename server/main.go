package main

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

const port = ":12588"

func main() {
	log.Println("Initializing inviv-v2 server")

	http.HandleFunc("/ws", wsHandler)
	log.Printf("Server starting on %s\n", port)
	log.Fatal(http.ListenAndServe(port, nil))
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// TODO: replace with actual, normal auth
		return true
	},
}

func wsHandler(w http.ResponseWriter, r *http.Request) {
	// Upgrade HTTP connection to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Upgrade error:", err)
		return
	}
	defer conn.Close()

	log.Println("Client connected")

	// Read messages from the client
	for {
		messageType, message, err := conn.ReadMessage()
		if err != nil {
			log.Println("Read error:", err)
			break
		}

		log.Printf("Received: %s", message)

		// Echo the message back to the client
		err = conn.WriteMessage(messageType, message)
		if err != nil {
			log.Println("Write error:", err)
			break
		}
	}

	log.Println("Client disconnected")
}
