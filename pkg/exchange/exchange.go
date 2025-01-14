package exchange

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/gorilla/websocket"
)

// TODO put into env var
var tickers = []string{
	"BTC/USD", "ETH/USD", "LTC/USD", "XRP/USD", "BCH/USD", "USDC/USD", "XMR/USD", "XLM/USD",
	"USDT/USD", "QCAD/USD", "DOGE/USD", "LINK/USD", "MATIC/USD", "UNI/USD", "COMP/USD", "AAVE/USD", "DAI/USD",
	"SUSHI/USD", "SNX/USD", "CRV/USD", "DOT/USD", "YFI/USD", "MKR/USD", "PAXG/USD", "ADA/USD", "BAT/USD", "ENJ/USD",
	"AXS/USD", "DASH/USD", "EOS/USD", "BAL/USD", "KNC/USD", "ZRX/USD", "SAND/USD", "GRT/USD", "QNT/USD", "ETC/USD",
	"ETHW/USD", "1INCH/USD", "CHZ/USD", "CHR/USD", "SUPER/USD", "ELF/USD", "OMG/USD", "FTM/USD", "MANA/USD",
	"SOL/USD", "ALGO/USD", "LUNC/USD", "UST/USD", "ZEC/USD", "XTZ/USD", "AMP/USD", "REN/USD", "UMA/USD", "SHIB/USD",
	"LRC/USD", "ANKR/USD", "HBAR/USD", "EGLD/USD", "AVAX/USD", "ONE/USD", "GALA/USD", "ALICE/USD", "ATOM/USD",
	"DYDX/USD", "CELO/USD", "STORJ/USD", "SKL/USD", "CTSI/USD", "BAND/USD", "ENS/USD", "RNDR/USD", "MASK/USD",
	"APE/USD",
}

type Exchange interface {
	ListenRatesUpdates(ctx context.Context, publish func(channelName string, data any)) error
}

type KrakenExchange struct {
	Url string
}

func NewKrakenExchange() Exchange {
	return &KrakenExchange{
		Url: "wss://ws.kraken.com/v2",
	}
}

// ListenRatesUpdates connects to kraken websocket updates to listen for rates realtime with retry mechanism
func (e *KrakenExchange) ListenRatesUpdates(ctx context.Context, notifyClients func(channelName string, data any)) error {
	for {
		conn, err := connectToWebSocket(ctx, e.Url)
		if err != nil {
			return fmt.Errorf("error connecting to Kraken WebSocket: %s", err)
		}
		defer conn.Close()

		// subscribe to ticker data for BTC/USD
		subscribeMessage := KrakenSocketMessage{
			Method: "subscribe",
			Params: KrakenSocketMessageParams{
				Channel: "ticker",
				Symbol:  tickers,
			},
		}
		err = conn.WriteJSON(subscribeMessage)
		if err != nil {
			return fmt.Errorf("error subscribing to Kraken WebSocket: %s", err)
		}

		// listen for messages from Kraken
		for {
			select {
			case <-ctx.Done():
				log.Println("cancelation received. closing exchange connection")
				return nil
			default:
				_, message, err := conn.ReadMessage()
				if err != nil {
					log.Printf("error reading from Kraken WebSocket: %s", err)
					//breaking to outer loop to reconnect
					break
				}

				var krakenResponse KrakenUpdate
				if err := json.Unmarshal(message, &krakenResponse); err != nil {
					log.Println("invalid msg")
					continue
				}
				if krakenResponse.Channel == "ticker" {
					for _, kr := range krakenResponse.Data {
						clientResponse := RateResponseData{
							Symbol:    kr.Symbol,
							Bid:       kr.Bid,
							Ask:       kr.Ask,
							Spot:      kr.Last,
							Change:    kr.ChangePct,
							Timestamp: time.Now().Unix(),
						}
						//firing goroutine worker to notify clients
						go notifyClients("rates", clientResponse)
					}
				}
			}
		}
	}
}

// connectToWebSocket connects to exchange websocket with exponential backoff retries
func connectToWebSocket(ctx context.Context, url string) (*websocket.Conn, error) {
	var conn *websocket.Conn
	var err error
	backoffInterval := 2 * time.Second // backoff interval
	maxBackoff := 30 * time.Second     // max backoff interval

	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("connection attempt canceled: %v", ctx.Err())
		default:
			conn, _, err = websocket.DefaultDialer.DialContext(ctx, url, nil)
			if err == nil {
				log.Printf("Successfully connected to %s. Listening...", url)
				return conn, nil
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
