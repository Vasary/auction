package amqp

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"auction-core/internal/metrics"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

type Config struct {
	URL      string
	Exchange string
}

type Client struct {
	cfg    Config
	logger *zap.Logger

	mu      sync.RWMutex
	conn    *amqp.Connection
	channel *amqp.Channel

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	notifyConnClose chan *amqp.Error
	notifyChanClose chan *amqp.Error
	isConnected     bool
}

func NewClient(cfg Config, logger *zap.Logger) *Client {
	ctx, cancel := context.WithCancel(context.Background())
	return &Client{
		cfg:    cfg,
		logger: logger,
		ctx:    ctx,
		cancel: cancel,
	}
}

func (c *Client) Start(ctx context.Context) error {
	if err := c.connect(); err != nil {
		return err
	}

	c.wg.Add(1)
	go c.reconnectLoop(ctx)

	return nil
}

func (c *Client) connect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.isConnected && c.conn != nil && !c.conn.IsClosed() {
		return nil
	}

	conn, err := amqp.Dial(c.cfg.URL)
	if err != nil {
		return err
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return err
	}

	if err := ch.ExchangeDeclare(
		c.cfg.Exchange,
		"topic",
		true,
		false,
		false,
		false,
		nil,
	); err != nil {
		ch.Close()
		conn.Close()
		return err
	}

	c.conn = conn
	c.channel = ch
	c.isConnected = true
	metrics.RecordAMQPConnectionState(true)
	c.notifyConnClose = make(chan *amqp.Error, 1)
	c.notifyChanClose = make(chan *amqp.Error, 1)
	c.conn.NotifyClose(c.notifyConnClose)
	c.channel.NotifyClose(c.notifyChanClose)

	c.logger.Info("amqp connected",
		zap.String("exchange", c.cfg.Exchange),
	)

	return nil
}

func (c *Client) reconnectLoop(ctx context.Context) {
	defer c.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case <-c.ctx.Done():
			return
		case err := <-c.notifyConnClose:
			c.handleDisconnect(err)
		case err := <-c.notifyChanClose:
			c.handleDisconnect(err)
		}

		if !c.isConnected {
			select {
			case <-ctx.Done():
				return
			case <-c.ctx.Done():
				return
			case <-time.After(5 * time.Second):
				if err := c.connect(); err != nil {
					metrics.AMQPReconnectTotal.WithLabelValues("error").Inc()
					c.logger.Error("amqp reconnect failed", zap.Error(err))
					continue
				}
				metrics.AMQPReconnectTotal.WithLabelValues("ok").Inc()
				c.logger.Info("amqp reconnected")
			}
		}
	}
}

func (c *Client) handleDisconnect(err error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.isConnected = false
	metrics.RecordAMQPConnectionState(false)
	if err != nil {
		c.logger.Warn("amqp connection lost", zap.Error(err))
	}
}

func (c *Client) Publish(routingKey string, payload interface{}) error {
	started := time.Now()

	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.isConnected || c.channel == nil {
		err := ErrNotConnected
		metrics.ObserveAMQPPublish(routingKey, started, err)
		return err
	}

	body, err := json.Marshal(payload)
	if err != nil {
		metrics.ObserveAMQPPublish(routingKey, started, err)
		return err
	}

	err = c.channel.PublishWithContext(
		c.ctx,
		c.cfg.Exchange,
		routingKey,
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Body:         body,
			Timestamp:    time.Now(),
		},
	)
	metrics.ObserveAMQPPublish(routingKey, started, err)
	return err
}

func (c *Client) Close() error {
	c.cancel()
	c.wg.Wait()

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.channel != nil {
		c.channel.Close()
	}
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
