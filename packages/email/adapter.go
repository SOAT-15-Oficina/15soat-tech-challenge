package email

import (
	"context"

	"github.com/ESSantana/15soat-tech-challenge-step-1/internal/application/port"
)

type portAdapter struct {
	provider Provider
}

func NewPortAdapter(provider Provider) port.EmailSender {
	return &portAdapter{provider: provider}
}

func (a *portAdapter) Send(ctx context.Context, msg port.EmailMessage) error {
	return a.provider.Send(ctx, Message{
		From:    msg.From,
		To:      msg.To,
		Subject: msg.Subject,
		Body:    msg.Body,
		HTML:    msg.HTML,
	})
}
