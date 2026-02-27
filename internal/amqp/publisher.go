package amqp

import (
	"auction-core/internal/auction"
	"time"

	"github.com/google/uuid"
)

type Publisher interface {
	Publish(routingKey string, payload interface{}) error
}

type AuctionEventPublisher struct {
	publisher Publisher
}

func NewAuctionEventPublisher(publisher Publisher) *AuctionEventPublisher {
	return &AuctionEventPublisher{
		publisher: publisher,
	}
}

type AuctionStartedEvent struct {
	TenderID     uuid.UUID      `json:"tenderId"`
	Status       auction.Status `json:"status"`
	StartPrice   int64          `json:"startPrice"`
	Step         int64          `json:"step"`
	CurrentPrice int64          `json:"currentPrice"`
	StartAt      time.Time      `json:"startAt"`
	EndAt        time.Time      `json:"endAt"`
	At           time.Time      `json:"at"`
}

type WinningBid struct {
	ID        int64     `json:"id"`
	BidAmount int64     `json:"bidAmount"`
	PersonID  uuid.UUID `json:"personId"`
	BidAt     time.Time `json:"bidAt"`
}

type AuctionFinishedEvent struct {
	TenderID        uuid.UUID      `json:"tenderId"`
	Status          auction.Status `json:"status"`
	StartPrice      int64          `json:"startPrice"`
	Step            int64          `json:"step"`
	CurrentPrice    int64          `json:"currentPrice"`
	WinnerCompanyID *uuid.UUID     `json:"winnerCompanyId,omitempty"`
	WinningBid      WinningBid     `json:"winningBid"`
	StartAt         time.Time      `json:"startAt"`
	EndAt           time.Time      `json:"endAt"`
	At              time.Time      `json:"at"`
}

func (p *AuctionEventPublisher) PublishAuctionStarted(tenderID uuid.UUID, snapshot auction.Snapshot) error {
	event := AuctionStartedEvent{
		TenderID:     tenderID,
		Status:       snapshot.Status,
		StartPrice:   snapshot.StartPrice,
		Step:         snapshot.Step,
		CurrentPrice: snapshot.CurrentPrice,
		StartAt:      snapshot.StartAt,
		EndAt:        snapshot.EndAt,
		At:           time.Now(),
	}
	return p.publisher.Publish("auction.started", event)
}

func (p *AuctionEventPublisher) PublishAuctionFinished(tenderID uuid.UUID, snapshot auction.Snapshot) error {
	event := AuctionFinishedEvent{
		TenderID:        tenderID,
		Status:          snapshot.Status,
		StartPrice:      snapshot.StartPrice,
		Step:            snapshot.Step,
		CurrentPrice:    snapshot.CurrentPrice,
		WinnerCompanyID: snapshot.WinnerID,
		WinningBid: WinningBid{
			ID:        snapshot.LatestBid.ID,
			BidAmount: snapshot.LatestBid.BidAmount,
			PersonID:  snapshot.LatestBid.PersonID,
			BidAt:     snapshot.LatestBid.BidAt,
		},
		StartAt: snapshot.StartAt,
		EndAt:   snapshot.EndAt,
		At:      time.Now(),
	}
	return p.publisher.Publish("auction.finished", event)
}
