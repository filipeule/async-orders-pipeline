package queue

import (
	"context"
	"encoding/json"
	"sync"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	QueueLeadReceived   = "lead.received"
	QueueDLQDecrypt     = "lead.dead.decrypt_failed"
	QueueDLQSchema      = "lead.dead.schema_failed"
	QueueDLQConsumer    = "lead.dead.consumer_failed"
	QueueDistSMS        = "dist.sms"
	QueueDistEmail      = "dist.email"
	QueueDistCallCenter = "dist.callcenter"
	QueueDistWhatsApp   = "dist.whatsapp"
	QueueDLQSMS         = "dist.dead.sms"
)

var allQueues = []string{
	QueueLeadReceived, QueueDLQDecrypt, QueueDLQSchema, QueueDLQConsumer,
	QueueDistSMS, QueueDistEmail, QueueDistCallCenter, QueueDistWhatsApp, QueueDLQSMS,
}

type Connection struct {
	conn *amqp.Connection
}

func NewConnection(url string) (*Connection, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, err
	}

	c := &Connection{conn: conn}

	return c, c.declareQueues()
}

func (c *Connection) declareQueues() error {
	ch, err := c.conn.Channel()
	if err != nil {
		return err
	}
	defer ch.Close()

	for _, q := range allQueues {
		if _, err := ch.QueueDeclare(q, true, false, false, false, nil); err != nil {
			return err
		}
	}

	return nil
}

func (c *Connection) NewPublisher() (*Publisher, error) {
	ch, err := c.conn.Channel()
	if err != nil {
		return nil, err
	}

	return &Publisher{ch: ch}, nil
}

// NewConsumer returns a delivery channel; each consumer gets its own AMQP channel.
func (c *Connection) NewConsumer(queue string) (<-chan amqp.Delivery, error) {
	ch, err := c.conn.Channel()
	if err != nil {
		return nil, err
	}

	if err := ch.Qos(1, 0, false); err != nil {
		return nil, err
	}

	return ch.Consume(queue, "", false, false, false, false, nil)
}

func (c *Connection) Close() {
	c.conn.Close()
}

// Publisher is safe for concurrent use from multiple goroutines.
type Publisher struct {
	mu sync.Mutex
	ch *amqp.Channel
}

func (p *Publisher) Publish(ctx context.Context, queue string, body any) error {
	b, err := json.Marshal(body)
	if err != nil {
		return err
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	return p.ch.PublishWithContext(ctx, "", queue, false, false, amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		Body:         b,
	})
}
