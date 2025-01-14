package exchange

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/gorilla/websocket"
)

type Exchange interface {
	Subscribe(publish func(channelName string, data any)) error
}

type KrakenExchange struct {
	Url string
}

func NewKrakenExchange() Exchange {
	return &KrakenExchange{
		Url: "wss://ws.kraken.com/v2",
	}
}

func (e *KrakenExchange) Subscribe(publish func(channelName string, data any)) error {
	conn, _, err := websocket.DefaultDialer.Dial(e.Url, nil)
	if err != nil {
		return fmt.Errorf("error connecting to Kraken WebSocket: %s", err)
	}
	defer conn.Close()

	// Subscribe to ticker data for BTC/USD
	subscribeMessage := KrakenSocketMessage{
		Method: "subscribe",
		Params: KrakenSocketMessageParams{
			Channel: "ticker",
			Symbol:  []string{"BTC/USD"}, //TODO
		},
	}
	err = conn.WriteJSON(subscribeMessage)
	if err != nil {
		return fmt.Errorf("error subscribing to Kraken WebSocket: %s", err)
	}

	// Listen for messages from Kraken
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			return fmt.Errorf("error reading from Kraken WebSocket: %s", err)
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
				publish("rates", clientResponse)
			}
		}
	}
}
