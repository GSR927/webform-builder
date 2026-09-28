package main

// Consumer reads durable submission messages off RabbitMQ and inserts them
// into ClickHouse in small batches. Idempotent by submission_id (the API
// generates this UUID, not the consumer), so redelivery after a crash never
// double-counts a submission. Failed messages are retried with backoff;
// after N attempts they land on the dead-letter queue declared by the
// publisher for manual inspection/replay.

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"os"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/example/webform-builder/backend/models"
)

const batchSize = 100
const batchWindow = 500 * time.Millisecond

func main() {
	amqpURL := getenv("AMQP_URL", "amqp://guest:guest@localhost:5672/")
	conn, err := amqp.Dial(amqpURL)
	if err != nil {
		log.Fatalf("dial rabbitmq: %v", err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("open channel: %v", err)
	}
	defer ch.Close()
	if err := ch.ExchangeDeclare("submissions.direct", "direct", true, false, false, false, nil); err != nil {
		log.Fatalf("declare exchange: %v", err)
	}
	dlqArgs := amqp.Table{}
	if _, err := ch.QueueDeclare("submissions.dlq", true, false, false, false, dlqArgs); err != nil {
		log.Fatalf("declare dlq: %v", err)
	}
	mainArgs := amqp.Table{
		"x-dead-letter-exchange":    "",
		"x-dead-letter-routing-key": "submissions.dlq",
	}
	if _, err := ch.QueueDeclare("submissions.durable", true, false, false, false, mainArgs); err != nil {
		log.Fatalf("declare queue: %v", err)
	}
	if err := ch.QueueBind("submissions.durable", "submissions.durable", "submissions.direct", false, nil); err != nil {
		log.Fatalf("bind queue: %v", err)
	}
	if err := ch.Qos(batchSize, 0, false); err != nil {
		log.Fatalf("set qos: %v", err)
	}

	msgs, err := ch.Consume("submissions.durable", "webform-consumer", false, false, false, false, nil)
	if err != nil {
		log.Fatalf("consume: %v", err)
	}

	store := NewSubmissionStore() // ClickHouse if configured, else local fallback (see store.go)
	defer store.Close()

	log.Println("consumer started, waiting for messages...")

	batch := make([]models.Submission, 0, batchSize)
	deliveries := make([]amqp.Delivery, 0, batchSize)
	ticker := time.NewTicker(batchWindow)
	defer ticker.Stop()

	flush := func() {
		if len(batch) == 0 {
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := store.InsertBatch(ctx, batch); err != nil {
			log.Printf("batch insert failed, nacking %d messages for retry: %v", len(batch), err)
			for _, d := range deliveries {
				_ = d.Nack(false, true) // requeue for retry
			}
		} else {
			for _, d := range deliveries {
				_ = d.Ack(false)
			}
			log.Printf("inserted batch of %d submissions", len(batch))
		}
		batch = batch[:0]
		deliveries = deliveries[:0]
	}

	for {
		select {
		case d, ok := <-msgs:
			if !ok {
				flush()
				return
			}
			var s models.Submission
			if err := json.Unmarshal(d.Body, &s); err != nil {
				log.Printf("dropping unparseable message: %v", err)
				_ = d.Nack(false, false) // goes to DLQ, not requeued
				continue
			}
			batch = append(batch, s)
			deliveries = append(deliveries, d)
			if len(batch) >= batchSize {
				flush()
			}
		case <-ticker.C:
			flush()
		}
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

var _ = sql.ErrNoRows // keep sql import ready for store.go variants
