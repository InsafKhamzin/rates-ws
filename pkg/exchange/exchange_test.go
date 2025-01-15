package exchange

import (
	"context"
	"crypto-ws/internal/websocket"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock WebSocket Client
type MockWebSocketClient struct {
	mock.Mock
}

func (m *MockWebSocketClient) Connect(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockWebSocketClient) Close() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockWebSocketClient) WriteJson(message interface{}) error {
	args := m.Called(message)
	return args.Error(0)
}

func (m *MockWebSocketClient) ReadMessage() ([]byte, error) {
	args := m.Called()
	return args.Get(0).([]byte), args.Error(1)
}

func TestListenRatesUpdates(t *testing.T) {
	mockClient := new(MockWebSocketClient)

	// mocked client
	exchange := &KrakenExchange{
		Client: mockClient,
	}

	mockClient.On("Connect", mock.Anything).Return(nil)
	mockClient.On("Close").Return(nil)
	mockClient.On("WriteJson", mock.Anything).Return(nil)

	// simulate the WebSocket ReadMessage() method with a valid response
	mockClient.On("ReadMessage").Return([]byte(`{"channel":"ticker","data":[{"symbol":"BTC/USD","bid":30000,"ask":31000,"last":30500,"changePct":0.5}]}`), nil)

	ctx, cancel := context.WithCancel(context.Background())

	//send cancellation in 100 ms
	go func() {
		time.Sleep(100 * time.Microsecond)
		cancel()
	}()

	broadcast := make(chan websocket.SocketMessageWithData, 100)
	err := exchange.ListenRatesUpdates(ctx, broadcast)
	assert.NoError(t, err)

	// wait to complete
	time.Sleep(1 * time.Second)
	cancel()

	mockClient.AssertExpectations(t)
}
