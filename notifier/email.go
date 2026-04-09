package notifier

import (
	"fmt"
	"net/smtp"

	"github.com/jordan-wright/email"
	"luci-app-echarging-go/config"
)

type EmailNotifier struct {
	cfg config.EmailConfig
}

func NewEmailNotifier(cfg config.EmailConfig) *EmailNotifier {
	return &EmailNotifier{cfg: cfg}
}

func (n *EmailNotifier) Send(summary, body string) error {
	if !n.cfg.Enabled {
		return nil
	}

	e := email.NewEmail()
	e.From = n.cfg.From
	e.To = n.cfg.To
	e.Subject = summary
	e.Text = []byte(body)

	addr := fmt.Sprintf("%s:%d", n.cfg.SMTPHost, n.cfg.SMTPPort)
	auth := smtp.PlainAuth("", n.cfg.Username, n.cfg.Password, n.cfg.SMTPHost)
	return e.Send(addr, auth)
}
