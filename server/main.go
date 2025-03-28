package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

const port string = ":12588"

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

type User struct {
	ID []byte
}

type Message struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

func wsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Upgrade error:", err)
		return
	}
	defer conn.Close()

	log.Println("Client connected")

	for {
		var message Message
		err := conn.ReadJSON(&message)
		if err != nil {
			log.Println("Read error:", err)
			continue
		}

		switch message.Type {
		case "msg":
			log.Println("received msg")
		}

		err = conn.WriteJSON("b")
		if err != nil {
			log.Println("Write error:", err)
			break
		}
	}

	log.Println("Client disconnected")
}
