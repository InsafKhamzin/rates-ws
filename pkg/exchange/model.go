package exchange

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
