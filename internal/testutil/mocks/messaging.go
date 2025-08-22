package mocks

import (
	"context"

	"github.com/stretchr/testify/mock"
)

// RuleUpdate represents a rule update message
type RuleUpdate struct {
	TenantID      string `json:"tenantId"`
	TargetScope   string `json:"targetScope"`
	RulepackID    string `json:"rulepackId"`
	Version       int    `json:"version"`
	ContentSHA256 string `json:"checksum"`
}

// MockPublisher mocks the message publisher interface
type MockPublisher struct {
	mock.Mock
}

func (m *MockPublisher) PublishRuleUpdate(ctx context.Context, update interface{}) error {
	args := m.Called(ctx, update)
	return args.Error(0)
}

func (m *MockPublisher) Publish(ctx context.Context, topic string, message interface{}) error {
	args := m.Called(ctx, topic, message)
	return args.Error(0)
}

func (m *MockPublisher) Close() error {
	args := m.Called()
	return args.Error(0)
}

// MockSubscriber mocks the message subscriber interface
type MockSubscriber struct {
	mock.Mock
}

func (m *MockSubscriber) Subscribe(ctx context.Context, topic string, handler func([]byte) error) error {
	args := m.Called(ctx, topic, handler)
	return args.Error(0)
}

func (m *MockSubscriber) Close() error {
	args := m.Called()
	return args.Error(0)
}
