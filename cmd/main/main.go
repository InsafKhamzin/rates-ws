package main

import (
	"context"
	"crypto-ws/internal/websocket"
	"crypto-ws/pkg/exchange"
	"crypto-ws/pkg/socket"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "net/http/pprof" // Import pprof
)

func main() {

	server := &http.Server{
		Addr: ":8080",
	}

	// goroutine to start the server
	go func() {
		log.Println("Starting server on :8080")
		err := server.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			log.Fatal("Error starting server:", err)
		}
	}()

	ctx, cancel := context.WithCancel(context.Background())

	hub := websocket.NewClientHub()
	websocketHandler := websocket.NewWebsocketHandler(hub)
	http.HandleFunc("/ws", websocketHandler.HandleWS)

	//broadcasting channel for communicating rates
	broadcast := make(chan websocket.SocketMessageWithData, 100)

	socketClient := socket.NewSocketClient("wss://ws.kraken.com/v2")
	exchange := exchange.NewKrakenExchange(socketClient)
	// goroutine to listen for rate updates in the background
	go func() {
		if err := exchange.ListenRatesUpdates(ctx, broadcast); err != nil {
			log.Printf("Error listening for rate updates: %v", err)
		}
	}()

	//gorouting for broadcasting rates to clients
	go hub.Broadcast(ctx, broadcast)

	// channel to listen for OS signals
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM) // Listen for SIGINT, SIGTERM

	<-stop
	log.Println("Shutdown signal received, shutting down gracefully...")

	//canceling context to release rate updates
	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	// attempt a graceful shutdown
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("Server forced to shutdown: %v\n", err)
	}

	log.Println("Server shut down successfully.")
}
