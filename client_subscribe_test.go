package courier

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	"***REMOVED***/metrics"
)

type ClientSubscribeSuite struct {
	suite.Suite
}

func TestClientSubscriberSuite(t *testing.T) {
	suite.Run(t, new(ClientSubscribeSuite))
}

func (s *ClientSubscribeSuite) TestSubscribe() {
	callback := func(_ context.Context, _ PubSub, _ Decoder) {}
	testcases := []struct {
		name           string
		pahoMock       func(*mock.Mock) *mockToken
		wantErr        bool
		useMiddlewares []SubscriberMiddlewareFunc
	}{
		{
			name: "Success",
			pahoMock: func(m *mock.Mock) *mockToken {
				t := &mockToken{}
				t.On("WaitTimeout", 10*time.Second).Return(true)
				t.On("Error").Return(nil)
				m.On("Subscribe", "topic", byte(QOSOne), mock.AnythingOfType("mqtt.MessageHandler")).
					Return(t)
				return t
			},
		},
		{
			name: "AssertingSubscriberMiddleware",
			pahoMock: func(m *mock.Mock) *mockToken {
				t := &mockToken{}
				t.On("WaitTimeout", 10*time.Second).Return(true)
				t.On("Error").Return(nil)
				m.On("Subscribe", "topic", byte(QOSZero), mock.AnythingOfType("mqtt.MessageHandler")).
					Return(t)
				return t
			},
			useMiddlewares: []SubscriberMiddlewareFunc{
				func(subscriber Subscriber) Subscriber {
					return NewSubscriberFuncs(
						func(ctx context.Context, topic string, qos QOSLevel, callback MessageHandler) error {
							s.Equal("topic", topic)
							s.Equal(QOSOne, qos)
							return subscriber.Subscribe(ctx, topic, QOSZero, callback)
						},
						subscriber.SubscribeMultiple,
					)
				},
				func(subscriber Subscriber) Subscriber {
					return NewSubscriberFuncs(
						func(ctx context.Context, topic string, qos QOSLevel, callback MessageHandler) error {
							s.Equal("topic", topic)
							s.Equal(QOSZero, qos)
							return subscriber.Subscribe(ctx, topic, qos, callback)
						},
						subscriber.SubscribeMultiple,
					)
				},
			},
		},
		{
			name: "WaitTimeout",
			pahoMock: func(m *mock.Mock) *mockToken {
				t := &mockToken{}
				t.On("WaitTimeout", 10*time.Second).Return(false)
				m.On("Subscribe", "topic", byte(QOSOne), mock.AnythingOfType("mqtt.MessageHandler")).
					Return(t)
				return t
			},
			wantErr: true,
		},
		{
			name: "Error",
			pahoMock: func(m *mock.Mock) *mockToken {
				t := &mockToken{}
				t.On("WaitTimeout", 10*time.Second).Return(true)
				t.On("Error").Return(errors.New("error"))
				m.On("Subscribe", "topic", byte(QOSOne), mock.AnythingOfType("mqtt.MessageHandler")).
					Return(t)
				return t
			},
			wantErr: true,
		},
	}
	for _, t := range testcases {
		s.Run(t.name, func() {
			c, err := NewClient(WithCustomMetrics(metrics.NewPrometheus()))
			s.NoError(err)

			if t.useMiddlewares != nil {
				c.UseSubscriberMiddleware(t.useMiddlewares...)
			}

			mc := &mockClient{}
			c.mqttClient = mc
			tk := t.pahoMock(&mc.Mock)

			err = c.Subscribe(context.Background(), "topic", QOSOne, callback)

			if !t.wantErr {
				s.NoError(err)
			} else {
				s.Error(err)
			}
			mc.AssertExpectations(s.T())
			tk.AssertExpectations(s.T())
		})
	}
}

func (s *ClientSubscribeSuite) TestSubscribeMultiple() {
	callback := func(_ context.Context, _ PubSub, _ Decoder) {}
	topics := map[string]QOSLevel{"topic": QOSOne}
	testcases := []struct {
		name           string
		pahoMock       func(*mock.Mock) *mockToken
		wantErr        bool
		useMiddlewares []SubscriberMiddlewareFunc
	}{
		{
			name: "Success",
			pahoMock: func(m *mock.Mock) *mockToken {
				t := &mockToken{}
				t.On("WaitTimeout", 10*time.Second).Return(true)
				t.On("Error").Return(nil)
				m.On("SubscribeMultiple", routeFilters(topics), mock.AnythingOfType("mqtt.MessageHandler")).
					Return(t)
				return t
			},
		},
		{
			name: "AssertingSubscriberMiddleware",
			pahoMock: func(m *mock.Mock) *mockToken {
				t := &mockToken{}
				t.On("WaitTimeout", 10*time.Second).Return(true)
				t.On("Error").Return(nil)
				m.On("SubscribeMultiple",
					routeFilters(map[string]QOSLevel{"topic": QOSZero}),
					mock.AnythingOfType("mqtt.MessageHandler")).
					Return(t)
				return t
			},
			useMiddlewares: []SubscriberMiddlewareFunc{
				func(subscriber Subscriber) Subscriber {
					return NewSubscriberFuncs(
						subscriber.Subscribe,
						func(ctx context.Context, topicsWithQos map[string]QOSLevel, callback MessageHandler) error {
							s.Equal(topics, topicsWithQos)
							topicsWithQos["topic"] = QOSZero
							return subscriber.SubscribeMultiple(ctx, topicsWithQos, callback)
						},
					)
				},
				func(subscriber Subscriber) Subscriber {
					return NewSubscriberFuncs(
						subscriber.Subscribe,
						func(ctx context.Context, topicsWithQos map[string]QOSLevel, callback MessageHandler) error {
							s.Equal(map[string]QOSLevel{"topic": QOSZero}, topicsWithQos)
							return subscriber.SubscribeMultiple(ctx, topicsWithQos, callback)
						},
					)
				},
			},
		},
		{
			name: "WaitTimeout",
			pahoMock: func(m *mock.Mock) *mockToken {
				t := &mockToken{}
				t.On("WaitTimeout", 10*time.Second).Return(false)
				m.On("SubscribeMultiple", routeFilters(topics), mock.AnythingOfType("mqtt.MessageHandler")).
					Return(t)
				return t
			},
			wantErr: true,
		},
		{
			name: "Error",
			pahoMock: func(m *mock.Mock) *mockToken {
				t := &mockToken{}
				t.On("WaitTimeout", 10*time.Second).Return(true)
				t.On("Error").Return(errors.New("error"))
				m.On("SubscribeMultiple", routeFilters(topics), mock.AnythingOfType("mqtt.MessageHandler")).
					Return(t)
				return t
			},
			wantErr: true,
		},
	}
	for _, t := range testcases {
		s.Run(t.name, func() {
			c, err := NewClient(WithCustomMetrics(metrics.NewPrometheus()))
			s.NoError(err)

			if t.useMiddlewares != nil {
				c.UseSubscriberMiddleware(t.useMiddlewares...)
			}

			mc := &mockClient{}
			c.mqttClient = mc
			tk := t.pahoMock(&mc.Mock)

			err = c.SubscribeMultiple(context.Background(), topics, callback)

			if !t.wantErr {
				s.NoError(err)
			} else {
				s.Error(err)
			}
			mc.AssertExpectations(s.T())
			tk.AssertExpectations(s.T())
		})
	}
}

func (s *ClientSubscribeSuite) TestSubscribeMiddleware() {
	callback := func(_ context.Context, _ PubSub, _ Decoder) {}
	c, err := NewClient(WithCustomMetrics(metrics.NewPrometheus()))
	s.NoError(err)

	mc := &mockClient{}
	mc.Test(s.T())
	c.mqttClient = mc

	t := &mockToken{}
	t.On("WaitTimeout", mock.Anything).Return(true)
	t.On("Error").Return(nil)
	mc.On("Subscribe", "topic", byte(QOSZero), mock.AnythingOfType("mqtt.MessageHandler")).
		Return(t)
	topics := map[string]QOSLevel{"topic": QOSZero}
	mc.On("SubscribeMultiple", routeFilters(topics), mock.AnythingOfType("mqtt.MessageHandler")).
		Return(t)

	tm := &testSubscribeMiddleware{}

	c.UseSubscriberMiddleware(tm.Middleware)
	s.Require().Len(c.sMiddlewares, 1)
	s.Equal(0, tm.timesSubscribeCalled)
	s.Equal(0, tm.timesSubscribeMultipleCalled)

	s.NoError(c.Subscribe(context.Background(), "topic", QOSZero, callback))
	s.NoError(c.SubscribeMultiple(context.Background(), topics, callback))
	s.Equal(1, tm.timesSubscribeCalled)
	s.Equal(1, tm.timesSubscribeMultipleCalled)

	c.UseSubscriberMiddleware(tm.Middleware)
	s.Require().Len(c.sMiddlewares, 2)
	s.Equal(1, tm.timesSubscribeCalled)
	s.Equal(1, tm.timesSubscribeMultipleCalled)

	s.NoError(c.Subscribe(context.Background(), "topic", QOSZero, callback))
	s.NoError(c.SubscribeMultiple(context.Background(), topics, callback))
	s.Equal(3, tm.timesSubscribeCalled)
	s.Equal(3, tm.timesSubscribeMultipleCalled)
}

type testSubscribeMiddleware struct {
	timesSubscribeCalled         int
	timesSubscribeMultipleCalled int
}

func (tm *testSubscribeMiddleware) Middleware(s Subscriber) Subscriber {
	return NewSubscriberFuncs(
		func(ctx context.Context, topic string, qos QOSLevel, callback MessageHandler) error {
			tm.timesSubscribeCalled++
			return s.Subscribe(ctx, topic, qos, callback)
		},
		func(ctx context.Context, topicsWithQos map[string]QOSLevel, callback MessageHandler) error {
			tm.timesSubscribeMultipleCalled++
			return s.SubscribeMultiple(ctx, topicsWithQos, callback)
		},
	)
}

func (s *ClientSubscribeSuite) Test_callbackWrapper() {
	c, err := NewClient(WithCustomMetrics(metrics.NewPrometheus()))
	s.NoError(err)

	f := callbackWrapper(c, func(_ context.Context, _ PubSub, _ Decoder) {
		s.T().Logf("callback called")
	})

	f(c.mqttClient, &testMsg{
		duplicate: false,
		qos:       1,
		retained:  false,
		topic:     "test",
		messageID: 1,
		payload:   []byte(`payload`),
		once:      sync.Once{},
		ack:       func() {},
	})
}

type testMsg struct {
	duplicate bool
	qos       byte
	retained  bool
	topic     string
	messageID uint16
	payload   []byte
	once      sync.Once
	ack       func()
}

func (t *testMsg) Duplicate() bool {
	return t.duplicate
}

func (t *testMsg) Qos() byte {
	return t.qos
}

func (t *testMsg) Retained() bool {
	return t.retained
}

func (t *testMsg) Topic() string {
	return t.topic
}

func (t *testMsg) MessageID() uint16 {
	return t.messageID
}

func (t *testMsg) Payload() []byte {
	return t.payload
}

func (t *testMsg) Ack() {
	t.once.Do(t.ack)
}