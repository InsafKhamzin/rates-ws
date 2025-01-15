package websocket

import (
	"sync"

	"github.com/gorilla/websocket"
)

// Subscriptions is the map of channels that clients may subscribe to
type Subscriptions struct {
	Channels map[string]Client
	Locks    map[string]*sync.Mutex //this mutex is needed to lock channels while modifying them to avoid collisions
}

// Client is the client id and socket connection
type Client map[string]*websocket.Conn

// SocketMessageBase is the base message format for socket messages exchange
type SocketMessageBase struct {
	Event   string `json:"event"`
	Channel string `json:"channel"`
}

// ErrorResponse error response
type ErrorResponse struct {
	ErrorMessage string `json:"error_message"`
}

// SocketMessageWithData is response message with all rates
type SocketMessageWithData struct {
	SocketMessageBase
	Data any `json:"data"`
}
