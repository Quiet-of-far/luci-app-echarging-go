package notifier

import "log"

type Notifier interface {
	Send(summary, body string) error
}

func SendAll(notifiers []Notifier, summary, body string) {
	for _, n := range notifiers {
		if err := n.Send(summary, body); err != nil {
			log.Printf("[notifier] 发送失败: %v", err)
		}
	}
}
