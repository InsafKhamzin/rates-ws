package main

import (
	"context"
	"crypto-ws/internal/websocket"
	"crypto-ws/pkg/exchange"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

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

	hub := websocket.NewClientHub()
	websocketHandler := websocket.NewWebsocketHandler(hub)
	http.HandleFunc("/ws", websocketHandler.HandleWS)

	exchange := exchange.NewKrakenExchange()
	go exchange.Subscribe(hub.Publish)

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
