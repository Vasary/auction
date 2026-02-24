package amqp

import (
	"auction-core/internal/auction"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockPublisher struct {
	mock.Mock
}

func (m *MockPublisher) Publish(routingKey string, payload interface{}) error {
	args := m.Called(routingKey, payload)
	return args.Error(0)
}

func TestAuctionEventPublisher_PublishAuctionStarted(t *testing.T) {
	mockPub := new(MockPublisher)
	publisher := NewAuctionEventPublisher(mockPub)

	tenderID := uuid.MustParse("00000000-0000-0000-0000-000000000701")
	snapshot := auction.Snapshot{
		Status:       auction.StatusActive,
		StartPrice:   100,
		Step:         10,
		CurrentPrice: 100,
		StartAt:      time.Now(),
		EndAt:        time.Now().Add(time.Hour),
	}

	mockPub.On("Publish", "auction.started", mock.MatchedBy(func(event AuctionStartedEvent) bool {
		return event.TenderID == tenderID && event.Status == snapshot.Status
	})).Return(nil)

	err := publisher.PublishAuctionStarted(tenderID, snapshot)

	assert.NoError(t, err)
	mockPub.AssertExpectations(t)
}

func TestAuctionEventPublisher_PublishAuctionFinished(t *testing.T) {
	mockPub := new(MockPublisher)
	publisher := NewAuctionEventPublisher(mockPub)

	tenderID := uuid.MustParse("00000000-0000-0000-0000-000000000702")
	winnerID := uuid.MustParse("00000000-0000-0000-0000-000000000703")
	snapshot := auction.Snapshot{
		Status:       auction.StatusFinished,
		StartPrice:   100,
		Step:         10,
		CurrentPrice: 150,
		WinnerID:     &winnerID,
		LatestBid: auction.LatestBid{
			BidAmount: 150,
			PersonID:  uuid.MustParse("00000000-0000-0000-0000-000000000704"),
			BidAt:     time.Now(),
		},
		StartAt: time.Now().Add(-time.Hour),
		EndAt:   time.Now(),
	}

	mockPub.On("Publish", "auction.finished", mock.MatchedBy(func(event AuctionFinishedEvent) bool {
		return event.TenderID == tenderID &&
			event.WinnerCompanyID == snapshot.WinnerID &&
			event.WinningBid.BidAmount == snapshot.LatestBid.BidAmount
	})).Return(nil)

	err := publisher.PublishAuctionFinished(tenderID, snapshot)

	assert.NoError(t, err)
	mockPub.AssertExpectations(t)
}
