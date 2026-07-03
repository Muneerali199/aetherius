package service

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/rs/zerolog/log"

	"github.com/aetherius/platform/services/notification/internal/model"
)

type NotificationService struct {
	url      string
	conn     *amqp.Connection
	channel  *amqp.Channel
	consumer string
}

func NewNotificationService(url string) *NotificationService {
	suffix := make([]byte, 4)
	rand.Read(suffix)
	return &NotificationService{
		url:      url,
		consumer: "notification." + hex.EncodeToString(suffix),
	}
}

func (s *NotificationService) Start() error {
	conn, err := amqp.Dial(s.url)
	if err != nil {
		return fmt.Errorf("dial rabbitmq: %w", err)
	}
	s.conn = conn

	ch, err := conn.Channel()
	if err != nil {
		return fmt.Errorf("open channel: %w", err)
	}
	s.channel = ch

	if err := ch.ExchangeDeclare(
		"events",
		"topic",
		true,
		false,
		false,
		false,
		nil,
	); err != nil {
		return fmt.Errorf("declare exchange: %w", err)
	}

	queue, err := ch.QueueDeclare(
		"",
		false,
		false,
		true,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("declare queue: %w", err)
	}

	routingKeys := []string{"deployment.*", "node.*", "billing.*", "marketplace.*"}
	for _, key := range routingKeys {
		if err := ch.QueueBind(queue.Name, key, "events", false, nil); err != nil {
			return fmt.Errorf("bind queue %s -> %s: %w", queue.Name, key, err)
		}
	}

	msgs, err := ch.Consume(
		queue.Name,
		s.consumer,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("consume: %w", err)
	}

	log.Info().Str("consumer", s.consumer).Msg("notification consumer started")

	go func() {
		for d := range msgs {
			s.process(d.Body)
		}
	}()

	return nil
}

func (s *NotificationService) process(body []byte) {
	var notif model.Notification
	if err := json.Unmarshal(body, &notif); err != nil {
		log.Error().Err(err).Msg("failed to unmarshal notification")
		return
	}

	fmt.Printf("[%s] %s: %s\n", time.Now().Format(time.RFC3339), notif.Type, notif.Message)

	switch {
	case isPrefix(string(notif.Type), "deployment"):
		s.handleDeploymentEvent(notif)
	case isPrefix(string(notif.Type), "node"):
		s.handleNodeEvent(notif)
	case isPrefix(string(notif.Type), "billing"):
		s.handleBillingEvent(notif)
	case isPrefix(string(notif.Type), "marketplace"):
		s.handleMarketplaceEvent(notif)
	}
}

func isPrefix(s, prefix string) bool {
	if len(s) < len(prefix) {
		return false
	}
	return s[:len(prefix)] == prefix
}

func (s *NotificationService) handleDeploymentEvent(n model.Notification) {
	fmt.Printf("EMAIL: to user@%s subject: %s\n", n.UserID, n.Title)
}

func (s *NotificationService) handleNodeEvent(n model.Notification) {
	fmt.Printf("EMAIL: to user@%s subject: %s\n", n.UserID, n.Title)
}

func (s *NotificationService) handleBillingEvent(n model.Notification) {
	log.Info().Str("type", string(n.Type)).Str("user_id", n.UserID).Msg("billing event received")
}

func (s *NotificationService) handleMarketplaceEvent(n model.Notification) {
	log.Info().Str("type", string(n.Type)).Str("user_id", n.UserID).Msg("marketplace event received")
}

func (s *NotificationService) Shutdown() {
	if s.channel != nil {
		s.channel.Close()
	}
	if s.conn != nil {
		s.conn.Close()
	}
}
