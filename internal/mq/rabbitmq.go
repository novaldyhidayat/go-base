package mq

import (
	"context"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"

	"go-base/internal/config"
)

// Client abstracts message queue interactions.
type Client interface {
	Publish(ctx context.Context, routingKey string, body []byte) error
	Subscribe(queue string, handler func(amqp.Delivery)) error
	Close() error
}

type rabbitClient struct {
	conn     *amqp.Connection
	channel  *amqp.Channel
	cfg      config.RabbitConfig
	queue    amqp.Queue
	exchange string
}

// NewRabbitMQ establishes a RabbitMQ connection and declares exchange/queue.
func NewRabbitMQ(cfg config.RabbitConfig) (Client, error) {
	conn, err := amqp.Dial(cfg.URI)
	if err != nil {
		return nil, fmt.Errorf("dial rabbitmq: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("create channel: %w", err)
	}

	if err := ch.ExchangeDeclare(
		cfg.ExchangeName,
		cfg.ExchangeType,
		cfg.Durable,
		false,
		false,
		false,
		nil,
	); err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return nil, fmt.Errorf("declare exchange: %w", err)
	}

	queue, err := ch.QueueDeclare(
		cfg.QueueName,
		cfg.Durable,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return nil, fmt.Errorf("declare queue: %w", err)
	}

	if err := ch.QueueBind(
		queue.Name,
		cfg.RoutingKey,
		cfg.ExchangeName,
		false,
		nil,
	); err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return nil, fmt.Errorf("bind queue: %w", err)
	}

	return &rabbitClient{
		conn:     conn,
		channel:  ch,
		cfg:      cfg,
		queue:    queue,
		exchange: cfg.ExchangeName,
	}, nil
}

func (r *rabbitClient) Publish(ctx context.Context, routingKey string, body []byte) error {
	return r.channel.PublishWithContext(ctx, r.exchange, routingKey, false, false, amqp.Publishing{
		ContentType: "application/json",
		Body:        body,
	})
}

func (r *rabbitClient) Subscribe(queue string, handler func(amqp.Delivery)) error {
	msgs, err := r.channel.Consume(queue, "", true, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("consume queue: %w", err)
	}

	go func() {
		for msg := range msgs {
			handler(msg)
		}
	}()

	return nil
}

func (r *rabbitClient) Close() error {
	if err := r.channel.Close(); err != nil {
		return err
	}
	return r.conn.Close()
}
