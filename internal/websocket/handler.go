package websocket

import (
	"log"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const (
	// max message size
	maxMessageSize = 512
	// I/O read buffer size
	readBufferSize = 1024
	// I/O write buffer size
	writeBufferSize = 1024
)

// http to websocket upgrader
var upgrader = websocket.Upgrader{
	ReadBufferSize:  readBufferSize,
	WriteBufferSize: writeBufferSize,
	CheckOrigin: func(r *http.Request) bool {
		// allow all origin
		return true
	},
}

type WebSocketHandler interface {
	HandleWS(w http.ResponseWriter, r *http.Request)
}
type WebsocketHadlerDefault struct {
	Hub *ClientHub
}

func NewWebsocketHandler(hub *ClientHub) WebSocketHandler {
	return &WebsocketHadlerDefault{Hub: hub}
}

func (wh *WebsocketHadlerDefault) HandleWS(w http.ResponseWriter, r *http.Request) {
	// updgrades http connection to websocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Error upgrading:", err)
		return
	}
	defer conn.Close()
	conn.SetReadLimit(maxMessageSize)

	clientID := uuid.New().String()
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			wh.Hub.RemoveClient(clientID)
			break
		}
		wh.Hub.ProcessMessage(conn, clientID, message)
	}
}
