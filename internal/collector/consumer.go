package collector

import (
	"encoding/json"
	"log/slog"

	"github.com/nats-io/nats.go"
)

type ClickEvent struct {
	Code      string `json:"code"`
	IP        string `json:"ip"`
	UserAgent string `json:"user_agent"`
	Referer   string `json:"referer"`
	Timestamp int64  `json:"timestamp"`
}

type Consumer struct {
	js     nats.JetStreamContext
	writer *Writer
	log    *slog.Logger
	sub    *nats.Subscription
}

func NewConsumer(js nats.JetStreamContext, writer *Writer, log *slog.Logger) *Consumer {
	return &Consumer{js: js, writer: writer, log: log}
}

func (c *Consumer) Start() error {
	sub, err := c.js.Subscribe("clicks.created", func(msg *nats.Msg) {
		var event ClickEvent
		if err := json.Unmarshal(msg.Data, &event); err != nil {
			c.log.Error("failed to unmarshal click event", "error", err)
			_ = msg.Nak()
			return
		}
		c.writer.Add(event)
		_ = msg.Ack()
	}, nats.Durable("collector"), nats.ManualAck())
	if err != nil {
		return err
	}

	c.sub = sub
	c.log.Info("subscribed to clicks.created (JetStream durable)")
	return nil
}

func (c *Consumer) Stop() error {
	if c.sub != nil {
		return c.sub.Drain()
	}
	return nil
}
