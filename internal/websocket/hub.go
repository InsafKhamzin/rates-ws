package websocket

import (
	"encoding/json"
	"errors"
	"log"
	"slices"
	"strings"
	"sync"

	"github.com/gorilla/websocket"
)

var (
	channelNames = []string{"rates"}
)

// ClientHub is the hub to manage and interact with clients socket subscription
type ClientHub struct {
	Subscriptions Subscriptions
}

// NewClientHub initializes client hub with provided channels
func NewClientHub() *ClientHub {
	channels := make(map[string]Client)
	locks := make(map[string]*sync.Mutex)
	for _, cn := range channelNames {
		channels[cn] = make(Client)
		locks[cn] = &sync.Mutex{}
	}

	return &ClientHub{Subscriptions: Subscriptions{
		Channels: channels,
		Locks:    locks,
	}}
}

// Subscribe adds a client to a channel's client map
func (c *ClientHub) Subscribe(conn *websocket.Conn, clientID string, channelName string) error {
	if !slices.Contains(channelNames, channelName) {
		return errors.New("channel not supported")
	}

	if channel, ok := c.Subscriptions.Channels[channelName]; ok {
		c.Subscriptions.Locks[channelName].Lock()
		channel[clientID] = conn
		c.Subscriptions.Locks[channelName].Unlock()
	}
	return nil
}

// Unsubscribe removes client from a channel's client map
func (c *ClientHub) Unsubscribe(clientID string, channelName string) error {
	if !slices.Contains(channelNames, channelName) {
		return errors.New("channel not supported")
	}

	if channel, ok := c.Subscriptions.Channels[channelName]; ok {
		c.Subscriptions.Locks[channelName].Lock()
		delete(channel, clientID)
		c.Subscriptions.Locks[channelName].Unlock()
	}
	return nil
}

// RemoveClient removes the clients from the server subscription map
func (c *ClientHub) RemoveClient(clientID string) {
	for _, channel := range c.Subscriptions.Channels {
		delete(channel, clientID)
	}
}

// Send sends error message
func (s *ClientHub) SendError(conn *websocket.Conn, errorMsg string) {
	response := ErrorResponse{ErrorMessage: errorMsg}
	conn.WriteJSON(response)
}

// Send sends json
func (s *ClientHub) SendJson(conn *websocket.Conn, data any) {
	conn.WriteJSON(data)
}

// ProcessMessage handle messages from client
func (c *ClientHub) ProcessMessage(conn *websocket.Conn, clientID string, msg []byte) {
	// parse message
	m := SocketMessageBase{}
	if err := json.Unmarshal(msg, &m); err != nil {
		c.SendError(conn, "invalid request format")
		return
	}

	event := strings.TrimSpace(strings.ToLower(m.Event))
	switch event {
	case "subscribe":
		err := c.Subscribe(conn, clientID, m.Channel)
		if err != nil {
			c.SendError(conn, err.Error())
			break
		}
		c.SendJson(conn, SocketMessageBase{Event: "subscribed", Channel: m.Channel})
		log.Printf("Client %s subscribed to %s", clientID, m.Channel)

	case "unsubscribe":
		err := c.Unsubscribe(clientID, m.Channel)
		if err != nil {
			c.SendError(conn, err.Error())
			break
		}
		c.SendJson(conn, SocketMessageBase{Event: "unsubscribed", Channel: m.Channel})
		log.Printf("Client %s unsubscribed from %s", clientID, m.Channel)

	default:
		c.SendError(conn, "unsuported event")
	}
}

// Publish sends a message to all subscribing clients of a topic
func (c *ClientHub) Publish(channelName string, data any) {
	channel, ok := c.Subscriptions.Channels[channelName]
	if !ok {
		return
	}

	//if not clients subscribed
	if len(channel) == 0 {
		return
	}

	response := Response{
		SocketMessageBase: SocketMessageBase{
			Channel: channelName,
			Event:   "data",
		},
		Data: data,
	}

	msg, err := json.Marshal(response)
	if err != nil {
		log.Printf("error while marshling to json: %s", err)
	}

	lock := c.Subscriptions.Locks[channelName]
	lock.Lock()
	defer lock.Unlock()

	for clientID, conn := range channel {
		err := conn.WriteMessage(websocket.TextMessage, msg)
		//removing client if failed to write
		if err != nil {
			c.RemoveClient(clientID)
		}
	}
}
