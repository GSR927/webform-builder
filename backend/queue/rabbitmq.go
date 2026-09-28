package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Publisher struct {
	conn *amqp.Connection
	ch   *amqp.Channel
	mu   sync.Mutex
}

const (
	ExchangeName   = "submissions.direct"
	QueueName      = "submissions.durable"
	DeadLetterName = "submissions.dlq"
)

func NewPublisher(amqpURL string) (*Publisher, error) {
	conn, err := amqp.Dial(amqpURL)
	if err != nil {
		return nil, fmt.Errorf("dial rabbitmq: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("open channel: %w", err)
	}

	if err := ch.Confirm(false); err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return nil, fmt.Errorf("enable confirm mode: %w", err)
	}

	if err := declareTopology(ch); err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return nil, err
	}

	return &Publisher{conn: conn, ch: ch}, nil
}

func declareTopology(ch *amqp.Channel) error {
	if err := ch.ExchangeDeclare(
		ExchangeName, "direct", true, false, false, false, nil,
	); err != nil {
		return err
	}

	if _, err := ch.QueueDeclare(
		DeadLetterName, true, false, false, false, nil,
	); err != nil {
		return err
	}

	mainArgs := amqp.Table{
		"x-dead-letter-exchange":    "",
		"x-dead-letter-routing-key": DeadLetterName,
	}

	if _, err := ch.QueueDeclare(
		QueueName, true, false, false, false, mainArgs,
	); err != nil {
		return err
	}

	return ch.QueueBind(
		QueueName, QueueName, ExchangeName, false, nil,
	)
}

func (p *Publisher) PublishSubmission(ctx context.Context, submission any) error {
	body, err := json.Marshal(submission)
	if err != nil {
		return fmt.Errorf("marshal submission: %w", err)
	}

	// One AMQP channel is shared by HTTP requests. Serialize publishing on
	// that channel; each call gets its own deferred confirmation.
	p.mu.Lock()
	defer p.mu.Unlock()

	if err := ctx.Err(); err != nil {
		return err
	}

	confirmation, err := p.ch.PublishWithDeferredConfirmWithContext(
		ctx,
		ExchangeName,
		QueueName,
		true,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Body:         body,
			Timestamp:    time.Now(),
		},
	)
	if err != nil {
		return fmt.Errorf("publish submission: %w", err)
	}
	if confirmation == nil {
		return fmt.Errorf("publisher confirmation unavailable")
	}

	acked, err := confirmation.WaitContext(ctx)
	if err != nil {
		return fmt.Errorf("wait for publisher confirmation: %w", err)
	}
	if !acked {
		return fmt.Errorf("broker rejected submission")
	}

	return nil
}

func (p *Publisher) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.ch != nil {
		_ = p.ch.Close()
	}
	if p.conn != nil {
		_ = p.conn.Close()
	}
}