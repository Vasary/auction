package auction

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrNotActive            = errors.New("auction not active")
	ErrFinished             = errors.New("auction finished")
	ErrBidNotLower          = errors.New("bid must be lower than current price")
	ErrBidNotAligned        = errors.New("bid not aligned with step")
	ErrRateLimited          = errors.New("rate limit exceeded")
	ErrInvalidConfig        = errors.New("invalid auction configuration")
	ErrSessionAlreadyDead   = errors.New("session stopped")
	ErrNotFound             = errors.New("auction not found")
	ErrAlreadyExists        = errors.New("auction already exists")
	ErrCannotDeleteStarted  = errors.New("cannot delete started or finished auction")
	ErrCannotUpdateStarted  = errors.New("cannot update started or already loaded auction")
	ErrCannotUpdateFinished = errors.New("cannot update finished auction")
)

type Status string

const (
	StatusScheduled Status = "Scheduled"
	StatusActive    Status = "Active"
	StatusFinished  Status = "Finished"
)

type LatestBid struct {
	ID        int64
	CompanyID uuid.UUID
	PersonID  uuid.UUID
	BidAmount int64
	BidAt     time.Time
}

type Snapshot struct {
	TenderID     uuid.UUID  `json:"tenderId"`
	Status       Status     `json:"status"`
	StartPrice   int64      `json:"startPrice"`
	Step         int64      `json:"step"`
	CurrentPrice int64      `json:"currentPrice"`
	WinnerID     *uuid.UUID `json:"winnerId,omitempty"`
	LatestBid    LatestBid  `json:"latestBid"`
	StartAt      time.Time  `json:"startAt"`
	EndAt        time.Time  `json:"endAt"`
	UpdatedAt    time.Time  `json:"updatedAt"`
}

type EventType string

const (
	EventSnapshot     EventType = "snapshot"
	EventPriceUpdated EventType = "price_updated"
	EventBidRejected  EventType = "bid_rejected"
	EventFinished     EventType = "finished"
	EventStarted      EventType = "started"
)

type Event struct {
	Type     EventType `json:"type"`
	TenderID uuid.UUID `json:"tenderId"`
	At       time.Time `json:"at"`
	Payload  any       `json:"payload"`
}

type BidRejection struct {
	CompanyID    uuid.UUID  `json:"companyId"`
	Reason       string     `json:"reason"`
	CurrentPrice int64      `json:"currentPrice"`
	WinnerID     *uuid.UUID `json:"winnerId,omitempty"`
}

type BidResult struct {
	Accepted     bool       `json:"accepted"`
	Error        string     `json:"error,omitempty"`
	CurrentPrice int64      `json:"currentPrice,omitempty"`
	WinnerID     *uuid.UUID `json:"winnerId,omitempty"`
}

type Config struct {
	TenderID     uuid.UUID
	WinnerID     *uuid.UUID
	StartPrice   int64
	CurrentPrice int64
	Step         int64
	StartAt      time.Time
	EndAt        time.Time
	LatestBid    LatestBid

	RateLimitPerBidder time.Duration
	BroadcastBuffer    int
}

type EventPublisher interface {
	PublishAuctionStarted(tenderID uuid.UUID, snapshot Snapshot) error
	PublishAuctionFinished(tenderID uuid.UUID, snapshot Snapshot) error
}

type AuctionsRepository interface {
	Create(ctx context.Context, a *PersistedAuction) error
	GetByID(ctx context.Context, tenderID uuid.UUID) (*PersistedAuction, error)
	Update(ctx context.Context, a *PersistedAuction) error
	UpdateStatus(ctx context.Context, tenderID uuid.UUID, status Status) error
	Delete(ctx context.Context, tenderID uuid.UUID) error
	FindStartingBetween(ctx context.Context, from, to time.Time) ([]PersistedAuction, error)
	List(ctx context.Context) ([]PersistedAuction, error)

	CreateBidTx(
		ctx context.Context,
		tenderID uuid.UUID,
		companyID uuid.UUID,
		personID uuid.UUID,
		amount int64,
	) (int64, error)
}

type PersistedAuction struct {
	TenderID     uuid.UUID
	StartPrice   int64
	Step         int64
	StartAt      time.Time
	EndAt        time.Time
	CreatedBy    uuid.UUID
	Status       Status
	CurrentPrice int64
	WinnerID     *uuid.UUID
	WinnerBidID  *int64
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type Bid struct {
	ID        int64
	TenderID  uuid.UUID
	CompanyID uuid.UUID
	PersonID  uuid.UUID
	BidAmount int64
	CreatedAt time.Time
}

type BidRepository interface {
	GetByID(ctx context.Context, bidID int64) (Bid, error)
	ListBids(ctx context.Context, tenderID uuid.UUID) ([]Bid, error)
}

type ParticipantRepository interface {
	AddParticipant(ctx context.Context, tenderID, companyID uuid.UUID) error
	IsParticipant(ctx context.Context, tenderID, companyID uuid.UUID) (bool, error)
	ListParticipants(ctx context.Context, tenderID uuid.UUID) ([]uuid.UUID, error)
}
