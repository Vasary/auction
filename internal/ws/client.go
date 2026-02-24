package ws

import (
	"context"
	"sync"
	"time"

	"auction-core/internal/auction"
	"auction-core/internal/metrics"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

type Client struct {
	conn            *websocket.Conn
	session         *auction.Session
	participantRepo auction.ParticipantRepository
	tenderID        uuid.UUID
	logger          *zap.Logger
	writeMu         sync.Mutex
}

func NewClient(
	conn *websocket.Conn,
	session *auction.Session,
	repo auction.ParticipantRepository,
	tenderID uuid.UUID,
	logger *zap.Logger,
) *Client {
	return &Client{
		conn:            conn,
		session:         session,
		participantRepo: repo,
		tenderID:        tenderID,
		logger:          logger,
	}
}

func (c *Client) Run() {
	c.logger.Info("ws client connected",
		zap.String("tender_id", c.tenderID.String()),
		zap.String("remote_addr", c.conn.RemoteAddr().String()),
	)

	defer func() {
		c.logger.Info("ws client disconnected",
			zap.String("tender_id", c.tenderID.String()),
			zap.String("remote_addr", c.conn.RemoteAddr().String()),
		)
		c.conn.Close()
	}()

	events, unsubscribe := c.session.Subscribe()
	defer unsubscribe()

	done := make(chan struct{})
	var once sync.Once

	finish := func() {
		once.Do(func() {
			close(done)
		})
	}

	go func() {
		c.writeLoop(events)
		finish()
	}()

	go func() {
		c.readLoop()
		finish()
	}()

	<-done
}

func (c *Client) writeLoop(events <-chan auction.Event) {
	for ev := range events {

		c.logger.Debug("broadcast to client",
			zap.String("tender_id", c.tenderID.String()),
			zap.String("event_type", string(ev.Type)),
		)

		if err := c.writeJSON(ev); err != nil {
			metrics.WSMessagesTotal.WithLabelValues("out", string(ev.Type), "error").Inc()
			c.logger.Warn("ws write error", zap.Error(err))
			return
		}
		metrics.WSMessagesTotal.WithLabelValues("out", string(ev.Type), "ok").Inc()
	}
}

type incomingMessage struct {
	Type      string    `json:"type"`
	Bid       int64     `json:"bid,omitempty"`
	CompanyID uuid.UUID `json:"companyId"`
	PersonID  uuid.UUID `json:"personId"`
}

func (c *Client) readLoop() {
	for {
		var msg incomingMessage

		if err := c.conn.ReadJSON(&msg); err != nil {
			metrics.WSMessagesTotal.WithLabelValues("in", "read_json", "error").Inc()
			c.logger.Warn("ws read error", zap.Error(err))
			return
		}

		switch msg.Type {
		case "place_bid":
			c.handlePlaceBid(msg)
		default:
			metrics.WSMessagesTotal.WithLabelValues("in", msg.Type, "unknown_type").Inc()
			c.logger.Warn("unknown ws message",
				zap.String("type", msg.Type),
			)
		}
	}
}

func (c *Client) handlePlaceBid(msg incomingMessage) {
	started := time.Now()

	if msg.CompanyID == uuid.Nil || msg.PersonID == uuid.Nil {
		metrics.ObserveWSMessage("place_bid", started, context.Canceled)
		metrics.AuctionBidsTotal.WithLabelValues("rejected", "invalid_identity").Inc()
		c.logger.Warn("bid rejected: missing company_id or person_id",
			zap.String("company_id", msg.CompanyID.String()),
			zap.String("person_id", msg.PersonID.String()),
		)
		_ = c.writeJSON(auction.BidResult{
			Accepted: false,
			Error:    "missing company_id or person_id",
		})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	allowed, err := c.participantRepo.IsParticipant(
		ctx,
		c.tenderID,
		msg.CompanyID,
	)
	if err != nil {
		metrics.ObserveWSMessage("place_bid", started, err)
		c.logger.Error("participant check failed", zap.Error(err))
		return
	}

	if !allowed {
		metrics.ObserveWSMessage("place_bid", started, context.Canceled)
		metrics.AuctionBidsTotal.WithLabelValues("rejected", "not_participant").Inc()
		c.logger.Warn("bid rejected: company not allowed",
			zap.String("company_id", msg.CompanyID.String()),
		)
		return
	}

	c.logger.Info("bid received",
		zap.String("tender_id", c.tenderID.String()),
		zap.String("company_id", msg.CompanyID.String()),
		zap.String("person_id", msg.PersonID.String()),
		zap.Int64("bid", msg.Bid),
	)

	result := c.session.PlaceBid(msg.CompanyID, msg.PersonID, msg.Bid)
	metrics.ObserveBidResult(result, msg.Bid)
	metrics.ObserveWSMessage("place_bid", started, nil)

	if err := c.writeJSON(result); err != nil {
		metrics.WSMessagesTotal.WithLabelValues("out", "bid_result", "error").Inc()
		return
	}
	metrics.WSMessagesTotal.WithLabelValues("out", "bid_result", "ok").Inc()
}

func (c *Client) writeJSON(v any) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	return c.conn.WriteJSON(v)
}
