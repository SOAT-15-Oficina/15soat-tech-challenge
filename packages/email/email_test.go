package email

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNew_Mailhog(t *testing.T) {
	cfg := Config{Host: "localhost", Port: 1025, From: "test@test.com"}
	provider, err := New("mailhog", cfg)
	assert.NoError(t, err)
	assert.NotNil(t, provider)
}

func TestNew_UnknownProvider(t *testing.T) {
	cfg := Config{}
	provider, err := New("unknown", cfg)
	assert.Error(t, err)
	assert.Nil(t, provider)
}
