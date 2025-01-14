package socket

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/gorilla/websocket"
)

// SocketClient client for websockets
type SocketClient interface {
	Connect(ctx context.Context) error
	Close() error
	ReadMessage() ([]byte, error)
	WriteJson(body any) error
}

type SocketClientDefalut struct {
	Url  string
	Conn *websocket.Conn
}

func NewSocketClient(url string) SocketClient {
	return &SocketClientDefalut{
		Url: url,
	}
}

func (sc *SocketClientDefalut) Connect(ctx context.Context) error {
	backoffInterval := 2 * time.Second // backoff interval
	maxBackoff := 30 * time.Second     // max backoff interval

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("connection attempt canceled: %v", ctx.Err())
		default:
			var err error
			sc.Conn, _, err = websocket.DefaultDialer.DialContext(ctx, sc.Url, nil)
			if err == nil {
				log.Printf("Successfully connected to %s. Listening...", sc.Url)
				return nil
			}

			log.Printf("Failed to connect to WebSocket, retrying in %v: %v", backoffInterval, err)

			// exponential backoff
			if backoffInterval < maxBackoff {
				backoffInterval *= 2
			}
			time.Sleep(backoffInterval)
		}
	}
}

func (sc *SocketClientDefalut) Close() error {
	return sc.Conn.Close()
}

func (sc *SocketClientDefalut) ReadMessage() ([]byte, error) {
	_, message, err := sc.Conn.ReadMessage()
	if err != nil {
		return nil, err
	}
	return message, nil
}

func (sc *SocketClientDefalut) WriteJson(body any) error {
	err := sc.Conn.WriteJSON(body)
	if err != nil {
		return err
	}
	return nil
}
