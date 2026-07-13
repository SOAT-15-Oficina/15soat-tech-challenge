package email

import (
	"context"
	"testing"

	"github.com/ESSantana/15soat-tech-challenge-step-1/internal/application/port"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubProvider struct {
	last Message
}

func (s *stubProvider) Send(_ context.Context, msg Message) error {
	s.last = msg
	return nil
}

func TestNewPortAdapter_DelegatesToProvider(t *testing.T) {
	provider := &stubProvider{}
	adapter := NewPortAdapter(provider)

	err := adapter.Send(context.Background(), port.EmailMessage{
		To:      []string{"cliente@example.com"},
		Subject: "Teste",
		Body:    "<p>Olá</p>",
		HTML:    true,
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"cliente@example.com"}, provider.last.To)
	assert.Equal(t, "Teste", provider.last.Subject)
	assert.True(t, provider.last.HTML)
}
