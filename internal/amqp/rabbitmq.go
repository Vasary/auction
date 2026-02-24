package amqp

import (
	"auction-core/internal/auction"
	"context"

	"go.uber.org/zap"
)

func New(
	ctx context.Context,
	log *zap.Logger,
	amqpURL string,
) (auction.EventPublisher, func(), error) {

	if amqpURL == "" {
		log.Info("amqp url not configured, events will not be published")
		return nil, func() {}, nil
	}

	client := NewClient(Config{
		URL:      amqpURL,
		Exchange: "auctions",
	}, log)

	if err := client.Start(ctx); err != nil {
		log.Warn("amqp connection failed, continuing without events",
			zap.Error(err),
		)
		return nil, func() {}, nil
	}

	log.Info("amqp client started")

	cleanup := func() {
		if err := client.Close(); err != nil {
			log.Error("amqp close error", zap.Error(err))
		} else {
			log.Info("amqp client closed")
		}
	}

	return NewAuctionEventPublisher(client), cleanup, nil
}
