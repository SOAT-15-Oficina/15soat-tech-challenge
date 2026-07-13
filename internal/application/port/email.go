package port

import "github.com/ESSantana/15soat-tech-challenge-step-1/packages/email"

// EmailSender is the application outbound port for email delivery.
// Implemented by packages/email (MailHog in development).
type EmailSender = email.Provider

// EmailMessage is the email payload used by the port.
type EmailMessage = email.Message
