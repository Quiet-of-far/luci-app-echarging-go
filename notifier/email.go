package notifier

import (
	"fmt"
	"net/smtp"
	"strings"

	"github.com/jordan-wright/email"
	"luci-app-5echarging-go/config"
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
	if strings.TrimSpace(n.cfg.SMTPHost) == "" {
		return fmt.Errorf("missing smtp_host")
	}
	if strings.TrimSpace(n.cfg.Username) == "" {
		return fmt.Errorf("missing username")
	}
	if strings.TrimSpace(n.cfg.Password) == "" {
		return fmt.Errorf("missing password")
	}
	if strings.TrimSpace(n.cfg.From) == "" {
		return fmt.Errorf("missing from address")
	}
	if len(n.cfg.To) == 0 {
		return fmt.Errorf("missing recipient list")
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
