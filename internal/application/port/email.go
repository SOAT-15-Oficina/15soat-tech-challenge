package port

import "context"

type EmailMessage struct {
	From    string
	To      []string
	Subject string
	Body    string
	HTML    bool
}

type EmailSender interface {
	Send(ctx context.Context, msg EmailMessage) error
}
