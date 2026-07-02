package queue

import (
	"context"
	"fmt"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/rs/zerolog/log"
)

type Client struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	url     string
}

type Event struct {
	Type      string
	Source    string
	Payload   []byte
	Timestamp time.Time
}

func NewClient(url string) *Client {
	return &Client{url: url}
}

func (c *Client) Connect() error {
	conn, err := amqp.Dial(c.url)
	if err != nil {
		return fmt.Errorf("dial rabbitmq: %w", err)
	}
	c.conn = conn

	ch, err := conn.Channel()
	if err != nil {
		return fmt.Errorf("open channel: %w", err)
	}
	c.channel = ch

	// Enable publisher confirms
	if err := ch.Confirm(false); err != nil {
		return fmt.Errorf("confirm mode: %w", err)
	}

	log.Info().Msg("connected to rabbitmq")
	return nil
}

func (c *Client) DeclareExchange(name, kind string) error {
	return c.channel.ExchangeDeclare(
		name,       // name
		kind,       // type: topic, direct, fanout, headers
		true,       // durable
		false,      // auto-delete
		false,      // internal
		false,      // no-wait
		nil,        // args
	)
}

func (c *Client) DeclareQueue(name string, args amqp.Table) (amqp.Queue, error) {
	return c.channel.QueueDeclare(
		name,       // name
		true,       // durable
		false,      // auto-delete
		false,      // exclusive
		false,      // no-wait
		args,       // args (e.g., dead-letter exchange)
	)
}

func (c *Client) BindQueue(queue, routingKey, exchange string) error {
	return c.channel.QueueBind(queue, routingKey, exchange, false, nil)
}

func (c *Client) Publish(ctx context.Context, exchange, routingKey string, event Event) error {
	return c.channel.PublishWithContext(ctx,
		exchange,
		routingKey,
		true,  // mandatory
		false, // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Timestamp:    event.Timestamp,
			Type:         event.Type,
			AppId:        event.Source,
			Body:         event.Payload,
		},
	)
}

func (c *Client) Consume(queue, consumer string, autoAck bool) (<-chan amqp.Delivery, error) {
	return c.channel.Consume(
		queue,
		consumer,
		autoAck,
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,
	)
}

func (c *Client) QOS(prefetchCount int) error {
	return c.channel.Qos(prefetchCount, 0, false)
}

func (c *Client) Close() {
	if c.channel != nil {
		c.channel.Close()
	}
	if c.conn != nil {
		c.conn.Close()
	}
}

// Standard exchanges used by the platform
const (
	ExchangeDomainEvents = "aetherius.domain.events"
	ExchangeDeadLetter   = "aetherius.dlx"

	RoutingKeyDeploymentRequested = "deployment.requested"
	RoutingKeyDeploymentScheduled = "deployment.scheduled"
	RoutingKeyDeploymentStarted   = "deployment.started"
	RoutingKeyDeploymentStopped   = "deployment.stopped"
	RoutingKeyNodeHeartbeat       = "node.heartbeat"
	RoutingKeyNodeOffline         = "node.offline"
	RoutingKeyNodeOnline          = "node.online"
	RoutingKeyBillingUsage        = "billing.usage.record"
	RoutingKeyBillingInvoice      = "billing.invoice.generated"
	RoutingKeyAlertTriggered      = "alert.triggered"
)

func DeclareStandardExchanges(c *Client) error {
	if err := c.DeclareExchange(ExchangeDomainEvents, "topic"); err != nil {
		return err
	}
	if err := c.DeclareExchange(ExchangeDeadLetter, "fanout"); err != nil {
		return err
	}
	return nil
}
