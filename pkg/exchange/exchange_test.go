package exchange

import (
	"context"
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

	// notifyClients function to capture sent data
	notifyClients := func(channelName string, data interface{}) {
		assert.Equal(t, "rates", channelName)
		rateData, ok := data.(RateResponseData)
		assert.True(t, ok)
		assert.Equal(t, "BTC/USD", rateData.Symbol)
		assert.NotZero(t, rateData.Timestamp)
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

	err := exchange.ListenRatesUpdates(ctx, notifyClients)
	assert.NoError(t, err)

	// wait to complete
	time.Sleep(1 * time.Second)
	cancel()

	mockClient.AssertExpectations(t)
}
