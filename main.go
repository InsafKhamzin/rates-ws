package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

var rateSubscribers = make(map[*websocket.Conn]bool)
var broadcast = make(chan RateResponse)
var mutex = &sync.Mutex{}

func wsHandler(w http.ResponseWriter, r *http.Request) {
	// updgrades http connection to websocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println("Error upgrading:", err)
		return
	}
	defer conn.Close()

	for {
		_, message, err := conn.ReadMessage()

		if err != nil {
			mutex.Lock()
			delete(rateSubscribers, conn)
			mutex.Unlock()
			break
		}
		var request SocketMessageBase
		if err := json.Unmarshal(message, &request); err != nil {
			fmt.Println("invalid msg")
			continue
		}
		response := &SocketMessageBase{
			Event:   "subscribed",
			Channel: "rates"}

		_ = conn.WriteJSON(response)
		mutex.Lock()
		rateSubscribers[conn] = true
		mutex.Unlock()
	}
}

func handleMessages() {
	for {
		// next message from the broadcast channel
		response := <-broadcast

		// Send the message to all connected clients
		mutex.Lock()
		for client := range rateSubscribers {
			err := client.WriteJSON(response)
			//closing client if failed to write
			if err != nil {
				client.Close()
				delete(rateSubscribers, client)
			}
		}
		mutex.Unlock()
	}
}

func main() {

	server := &http.Server{
		Addr: ":8080",
	}

	// Goroutine to start the server
	go func() {
		fmt.Println("Starting server on :8080")
		err := server.ListenAndServe()
		if err != nil {
			log.Fatal("Error starting server:", err)
		}
	}()

	http.HandleFunc("/ws", wsHandler)

	go handleMessages()
	go connectToKraken()

	// Channel to listen for OS signals
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM) // Listen for SIGINT, SIGTERM

	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Attempt a graceful shutdown
	if err := server.Shutdown(ctx); err != nil {
		fmt.Printf("Server forced to shutdown: %v", err)
	}

	fmt.Println("\nShutting down server...")
}

// Connect to Kraken WebSocket API and listen for ticker updates
func connectToKraken() {
	conn, _, err := websocket.DefaultDialer.Dial("wss://ws.kraken.com/v2", nil)
	if err != nil {
		log.Fatal("Error connecting to Kraken WebSocket:", err)
	}
	defer conn.Close()

	// Subscribe to ticker data for BTC/USD
	subscribeMessage := KrakenSocketMessage{
		Method: "subscribe",
		Params: KrakenSocketMessageParams{
			Channel: "ticker",
			Symbol:  []string{"BTC/USD"},
		},
	}
	err = conn.WriteJSON(subscribeMessage)
	if err != nil {
		log.Println("Error subscribing to Kraken WebSocket:", err)
		return
	}

	// Listen for messages from Kraken
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Println("Error reading from Kraken WebSocket:", err)
			break
		}

		var krakenResponse KrakenUpdate
		if err := json.Unmarshal(message, &krakenResponse); err != nil {
			fmt.Println("invalid msg")
			continue
		}
		if krakenResponse.Channel == "ticker" {
			for _, kr := range krakenResponse.Data {
				clientResponse := RateResponse{
					Channel: "rates",
					Event:   "data",
					Data: RateResponseData{
						Symbol:    kr.Symbol,
						Bid:       kr.Bid,
						Ask:       kr.Ask,
						Spot:      kr.Last,
						Change:    kr.ChangePct,
						Timestamp: time.Now().Unix(),
					},
				}
				broadcast <- clientResponse
			}
		}
	}
}

type SocketMessageBase struct {
	Event   string `json:"event"`
	Channel string `json:"channel"`
}

type KrakenSocketMessage struct {
	Method string                    `json:"method"`
	Params KrakenSocketMessageParams `json:"params"`
}

type KrakenSocketMessageParams struct {
	Channel string   `json:"channel"`
	Symbol  []string `json:"symbol"`
}

type KrakenUpdate struct {
	Channel string `json:"channel"`
	Type    string `json:"type"`
	Data    []struct {
		Symbol    string  `json:"symbol"`
		Bid       float64 `json:"bid"`
		Ask       float64 `json:"ask"`
		Last      float64 `json:"last"`
		ChangePct float64 `json:"change_pct"`
	} `json:"data"`
}

type RateResponse struct {
	Channel string           `json:"channel"`
	Event   string           `json:"event"`
	Data    RateResponseData `json:"data"`
}

type RateResponseData struct {
	Symbol    string  `json:"symbol"`
	Timestamp int64   `json:"timestamp"`
	Bid       float64 `json:"bid"`
	Ask       float64 `json:"ask"`
	Spot      float64 `json:"spot"`
	Change    float64 `json:"change"`
}
