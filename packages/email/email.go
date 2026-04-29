package email

import (
	"context"
	"fmt"
)

type Message struct {
	From    string
	To      []string
	Subject string
	Body    string
	HTML    bool
}

type Provider interface {
	Send(ctx context.Context, msg Message) error
}

type Config struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
}

func New(name string, cfg Config) (Provider, error) {
	switch name {
	case "mailhog":
		return newMailhog(cfg), nil
	default:
		return nil, fmt.Errorf("email: unknown provider %q", name)
	}
}
