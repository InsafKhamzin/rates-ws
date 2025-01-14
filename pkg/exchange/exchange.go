package exchange

import (
	"context"
	"crypto-ws/pkg/socket"
	"encoding/json"
	"fmt"
	"log"
	"time"
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
	Client socket.SocketClient
}

func NewKrakenExchange(client socket.SocketClient) Exchange {
	return &KrakenExchange{
		Client: client,
	}
}

// ListenRatesUpdates connects to kraken websocket updates to listen for rates realtime with retry mechanism
func (e *KrakenExchange) ListenRatesUpdates(ctx context.Context, notifyClients func(channelName string, data any)) error {
	for {
		err := e.Client.Connect(ctx)
		if err != nil {
			return fmt.Errorf("error connecting to Kraken WebSocket: %s", err)
		}
		defer e.Client.Close()

		// subscribe to ticker data for BTC/USD
		subscribeMessage := KrakenSocketMessage{
			Method: "subscribe",
			Params: KrakenSocketMessageParams{
				Channel: "ticker",
				Symbol:  tickers,
			},
		}
		err = e.Client.WriteJson(subscribeMessage)
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
				message, err := e.Client.ReadMessage()
				if err != nil {
					log.Printf("error reading from Kraken WebSocket: %s", err)
					//breaking to outer loop to reconnect
					break
				}

				var krakenResponse KrakenUpdate
				if err := json.Unmarshal(message, &krakenResponse); err != nil {
					log.Printf("invalid msg: %s", err)
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
