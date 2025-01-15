package websocket

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"slices"
	"strings"
	"sync"

	"github.com/gorilla/websocket"
)

// events
const (
	subscribeEvent    = "subscribe"
	subscribedEvent   = "subscribed"
	unsubscribeEvent  = "unsubscribe"
	unsubscribedEvent = "unsubscribed"
	dataEvent         = "data"
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
	for channelName, channel := range c.Subscriptions.Channels {
		c.Subscriptions.Locks[channelName].Lock()
		delete(channel, clientID)
		c.Subscriptions.Locks[channelName].Unlock()
	}
	log.Printf("Client %s disconnected", clientID)
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
	case subscribeEvent:
		err := c.Subscribe(conn, clientID, m.Channel)
		if err != nil {
			c.SendError(conn, err.Error())
			break
		}
		c.SendJson(conn, SocketMessageBase{Event: subscribedEvent, Channel: m.Channel})
		log.Printf("Client %s subscribed to %s", clientID, m.Channel)

	case unsubscribeEvent:
		err := c.Unsubscribe(clientID, m.Channel)
		if err != nil {
			c.SendError(conn, err.Error())
			break
		}
		c.SendJson(conn, SocketMessageBase{Event: unsubscribedEvent, Channel: m.Channel})
		log.Printf("Client %s unsubscribed from %s", clientID, m.Channel)

	default:
		c.SendError(conn, "unsuported event")
	}
}

// Broadcast broadcasting messages to clients
func (c *ClientHub) Broadcast(ctx context.Context, broadcast <-chan SocketMessageWithData) {
	for {
		select {
		case <-ctx.Done():
			log.Println("Broadcast received cancellation signal.")
			return
		case message, ok := <-broadcast:
			if !ok {
				log.Println("Broadcast channel closed.")
				return
			}
			channel, ok := c.Subscriptions.Channels[message.Channel]
			if !ok {
				continue
			}

			//if no clients subscribed
			if len(channel) == 0 {
				continue
			}

			msgRaw, err := json.Marshal(message)
			if err != nil {
				log.Printf("error while marshling to json: %s", err)
				continue
			}

			// collect failed clients
			var failedClients []string
			var connsToNotify []struct {
				ID   string
				Conn *websocket.Conn
			}

			lock := c.Subscriptions.Locks[message.Channel]
			lock.Lock()
			for clientID, conn := range channel {
				connsToNotify = append(connsToNotify, struct {
					ID   string
					Conn *websocket.Conn
				}{ID: clientID, Conn: conn})
			}
			lock.Unlock()

			for _, client := range connsToNotify {
				err := client.Conn.WriteMessage(websocket.TextMessage, msgRaw)
				if err != nil {
					//record client that are failed to receive. meaning they are inactive.
					failedClients = append(failedClients, client.ID)
				}
			}

			// remove failed clients
			lock.Lock()
			for _, clientID := range failedClients {
				delete(channel, clientID)
			}
			lock.Unlock()
		}
	}
}
