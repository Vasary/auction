package auction

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type bidCmd struct {
	companyID uuid.UUID
	personID  uuid.UUID
	bidAmount int64
	replyCh   chan BidResult
}

type subscribeCmd struct {
	ch      chan Event
	replyCh chan subscribeResult
}

type unsubscribeCmd struct {
	subID string
}

type stopCmd struct {
	replyCh chan struct{}
}

type subscribeResult struct {
	subID   string
	snap    Snapshot
	stopped bool
}

type Session struct {
	cfg Config

	status       Status
	startPrice   int64
	step         int64
	currentPrice int64
	winnerID     *uuid.UUID
	updatedAt    time.Time

	lastBidAt map[uuid.UUID]time.Time
	latestBid LatestBid
	subs      map[string]chan Event

	inbox chan any
	done  chan struct{}

	once   sync.Once
	runner sync.Once
	subSeq uint64
	subCnt atomic.Int64

	auctionsRepository AuctionsRepository
	eventPublisher     EventPublisher

	logger *zap.Logger
}

func NewSession(cfg Config, auctionsRepository AuctionsRepository, eventPublisher EventPublisher, logger *zap.Logger) (*Session, error) {
	if cfg.TenderID == uuid.Nil || cfg.StartPrice <= 0 || cfg.Step <= 0 {
		return nil, errors.New("missing tender ID, non-positive start price, or non-positive step")
	}
	if !cfg.EndAt.After(cfg.StartAt) {
		return nil, errors.New("endAt must be after startAt")
	}
	if cfg.RateLimitPerBidder <= 0 {
		cfg.RateLimitPerBidder = 400 * time.Millisecond
	}
	if cfg.BroadcastBuffer <= 0 {
		cfg.BroadcastBuffer = 64
	}

	res := &Session{
		cfg:                cfg,
		status:             StatusScheduled,
		startPrice:         cfg.StartPrice,
		step:               cfg.Step,
		currentPrice:       cfg.CurrentPrice,
		winnerID:           cfg.WinnerID,
		latestBid:          cfg.LatestBid,
		updatedAt:          time.Now(),
		lastBidAt:          make(map[uuid.UUID]time.Time),
		subs:               make(map[string]chan Event),
		inbox:              make(chan any, 1024),
		done:               make(chan struct{}),
		auctionsRepository: auctionsRepository,
		eventPublisher:     eventPublisher,
		logger:             logger,
	}

	if res.currentPrice == 0 {
		res.currentPrice = res.startPrice
	}

	if res.winnerID != nil && !res.latestBid.BidAt.IsZero() {
		res.lastBidAt[*res.winnerID] = res.latestBid.BidAt
	}

	return res, nil
}

func (s *Session) Start() {
	s.runner.Do(func() {
		go s.run()
	})
}

func (s *Session) Done() <-chan struct{} { return s.done }

func (s *Session) Stop() {
	s.once.Do(func() {
		s.Start()
		ch := make(chan struct{}, 1)
		s.inbox <- stopCmd{replyCh: ch}
		<-ch
	})
}

func (s *Session) PlaceBid(companyID, personID uuid.UUID, bidAmount int64) BidResult {
	if companyID == uuid.Nil || personID == uuid.Nil {
		return BidResult{Accepted: false, Error: "missing companyID or personID"}
	}

	if bidAmount < 1 {
		return BidResult{Accepted: false, Error: "bid must be at least 1"}
	}

	s.Start()
	reply := make(chan BidResult, 1)

	select {
	case <-s.done:
		return BidResult{Accepted: false, Error: ErrSessionAlreadyDead.Error()}
	case s.inbox <- bidCmd{companyID: companyID, personID: personID, bidAmount: bidAmount, replyCh: reply}:
	}

	select {
	case <-s.done:
		return BidResult{Accepted: false, Error: ErrSessionAlreadyDead.Error()}
	case res := <-reply:
		return res
	}
}

func (s *Session) Subscribe() (<-chan Event, func()) {
	s.Start()
	ch := make(chan Event, s.cfg.BroadcastBuffer)
	reply := make(chan subscribeResult, 1)

	select {
	case <-s.done:
		close(ch)
		return ch, func() {}
	case s.inbox <- subscribeCmd{ch: ch, replyCh: reply}:
	}

	var res subscribeResult
	select {
	case <-s.done:
		close(ch)
		return ch, func() {}
	case res = <-reply:
	}
	unsub := func() {
		select {
		case <-s.done:
			return
		default:
		}
		select {
		case s.inbox <- unsubscribeCmd{subID: res.subID}:
		default:
		}
	}
	return ch, unsub
}

func (s *Session) Snapshot() Snapshot {
	return s.makeSnapshot(s.updatedAt)
}

func (s *Session) Status() Status {
	return s.status
}

func (s *Session) ConnectionsCount() int {
	return int(s.subCnt.Load())
}

func (s *Session) run() {
	defer close(s.done)

	s.status = s.initialStatus(time.Now())
	startTimer := s.startTimer()
	endTimer := time.NewTimer(nonNegativeDuration(time.Until(s.cfg.EndAt)))
	if s.status == StatusFinished {
		s.finishAndPersist()
		return
	}

	for {
		select {
		case <-startTimer.C:
			s.handleStartTimer()

		case <-endTimer.C:
			s.markFinished()
			s.finishAndPersist()
			return

		case msg := <-s.inbox:
			if s.handleCommand(msg) {
				return
			}
		}
	}
}

func (s *Session) initialStatus(now time.Time) Status {
	switch {
	case now.Before(s.cfg.StartAt):
		return StatusScheduled
	case now.Before(s.cfg.EndAt):
		return StatusActive
	default:
		return StatusFinished
	}
}

func (s *Session) startTimer() *time.Timer {
	if s.status == StatusScheduled {
		return time.NewTimer(nonNegativeDuration(time.Until(s.cfg.StartAt)))
	}

	timer := time.NewTimer(0)
	<-timer.C
	return timer
}

func nonNegativeDuration(d time.Duration) time.Duration {
	if d < 0 {
		return 0
	}
	return d
}

func (s *Session) handleStartTimer() {
	if s.status == StatusScheduled && time.Now().Before(s.cfg.EndAt) {
		s.startAndPersist()
	}
}

func (s *Session) handleCommand(msg any) bool {
	switch m := msg.(type) {
	case bidCmd:
		s.handleBid(m)
	case subscribeCmd:
		s.handleSubscribe(m)
	case unsubscribeCmd:
		s.handleUnsubscribe(m)
	case stopCmd:
		s.handleStop(m)
		return true
	}
	return false
}

func (s *Session) handleStop(cmd stopCmd) {
	if s.status == StatusActive {
		s.markFinished()
		s.finishAndPersist()
	} else {
		s.closeAllSubs()
	}
	cmd.replyCh <- struct{}{}
}

func (s *Session) markFinished() {
	if s.status != StatusFinished {
		s.status = StatusFinished
		s.updatedAt = time.Now()
	}
}

func (s *Session) startAndPersist() {
	s.status = StatusActive
	s.updatedAt = time.Now()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := s.auctionsRepository.UpdateStatus(ctx, s.cfg.TenderID, StatusActive); err != nil {
		s.logger.Error("failed to update auction status to active",
			zap.String("tender_id", s.cfg.TenderID.String()),
			zap.Error(err),
		)
	}

	snap := s.makeSnapshot(s.updatedAt)
	s.broadcast(Event{
		Type:     EventStarted,
		TenderID: s.cfg.TenderID,
		At:       s.updatedAt,
		Payload:  snap,
	})

	if s.eventPublisher != nil {
		if err := s.eventPublisher.PublishAuctionStarted(s.cfg.TenderID, snap); err != nil {
			s.logger.Warn("failed to publish auction started event",
				zap.String("tender_id", s.cfg.TenderID.String()),
				zap.Error(err),
			)
		}
	}
}

func (s *Session) finishAndPersist() {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := s.auctionsRepository.UpdateStatus(ctx, s.cfg.TenderID, StatusFinished); err != nil {
		s.logger.Error("failed to update auction status to finished",
			zap.String("tender_id", s.cfg.TenderID.String()),
			zap.Error(err),
		)
	}

	snapshot := s.makeSnapshot(s.updatedAt)
	s.broadcast(Event{
		Type:     EventFinished,
		TenderID: s.cfg.TenderID,
		At:       s.updatedAt,
		Payload:  snapshot,
	})

	if s.eventPublisher != nil {
		if err := s.eventPublisher.PublishAuctionFinished(s.cfg.TenderID, snapshot); err != nil {
			s.logger.Warn("failed to publish auction finished event",
				zap.String("tender_id", s.cfg.TenderID.String()),
				zap.Error(err),
			)
		}
	}

	s.closeAllSubs()
}

func (s *Session) handleBid(cmd bidCmd) {
	if s.status == StatusFinished {
		cmd.replyCh <- BidResult{Accepted: false, Error: ErrFinished.Error()}
		return
	}
	if s.status != StatusActive {
		cmd.replyCh <- BidResult{Accepted: false, Error: ErrNotActive.Error()}
		return
	}

	now := time.Now()

	if last, ok := s.lastBidAt[cmd.companyID]; ok {
		if now.Sub(last) < s.cfg.RateLimitPerBidder {
			cmd.replyCh <- BidResult{Accepted: false, Error: ErrRateLimited.Error()}
			return
		}
	}

	if cmd.bidAmount >= s.currentPrice {
		s.logger.Warn("bid rejected: not lower than current price",
			zap.Int64("bid", cmd.bidAmount),
			zap.Int64("current", s.currentPrice),
		)
		cmd.replyCh <- BidResult{Accepted: false, Error: ErrBidNotLower.Error()}
		return
	}

	diff := s.currentPrice - cmd.bidAmount
	if diff < s.step || diff%s.step != 0 {
		s.logger.Warn("bid rejected: not aligned with step",
			zap.Int64("bid", cmd.bidAmount),
			zap.Int64("current", s.currentPrice),
			zap.Int64("step", s.step),
		)
		cmd.replyCh <- BidResult{Accepted: false, Error: ErrBidNotAligned.Error()}
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	bidID, err := s.auctionsRepository.CreateBidTx(ctx, s.cfg.TenderID, cmd.companyID, cmd.personID, cmd.bidAmount)
	if err != nil {
		s.logger.Error("failed to create bid", zap.Error(err))
		cmd.replyCh <- BidResult{Accepted: false, Error: "failed to persist bid"}
		return
	}

	s.lastBidAt[cmd.companyID] = now
	s.currentPrice = cmd.bidAmount
	winnerID := cmd.companyID
	s.winnerID = &winnerID
	s.updatedAt = now
	s.latestBid = LatestBid{
		ID:        bidID,
		CompanyID: cmd.companyID,
		PersonID:  cmd.personID,
		BidAmount: cmd.bidAmount,
		BidAt:     now,
	}

	cmd.replyCh <- BidResult{
		Accepted:     true,
		CurrentPrice: s.currentPrice,
		WinnerID:     s.winnerID,
	}

	s.broadcast(Event{
		Type:     EventPriceUpdated,
		TenderID: s.cfg.TenderID,
		At:       now,
		Payload:  s.makeSnapshot(now),
	})
}

func (s *Session) handleSubscribe(cmd subscribeCmd) {
	s.subSeq++
	subID := s.makeSubID(s.cfg.TenderID, s.subSeq)

	s.subs[subID] = cmd.ch
	s.subCnt.Store(int64(len(s.subs)))

	now := time.Now()
	snap := s.makeSnapshot(now)

	select {
	case cmd.ch <- Event{Type: EventSnapshot, TenderID: s.cfg.TenderID, At: now, Payload: snap}:
	default:
	}

	cmd.replyCh <- subscribeResult{subID: subID, snap: snap}

	s.logger.Info("client subscribed",
		zap.String("tender_id", s.cfg.TenderID.String()),
		zap.Int("subscribers", len(s.subs)),
	)
}

func (s *Session) handleUnsubscribe(cmd unsubscribeCmd) {
	if ch, ok := s.subs[cmd.subID]; ok {
		delete(s.subs, cmd.subID)
		close(ch)
		s.subCnt.Store(int64(len(s.subs)))

		s.logger.Info("client unsubscribed",
			zap.String("tender_id", s.cfg.TenderID.String()),
			zap.Int("subscribers", len(s.subs)),
		)
	}
}

func (s *Session) broadcast(ev Event) {
	s.logger.Debug("broadcast event",
		zap.String("tender_id", s.cfg.TenderID.String()),
		zap.String("event_type", string(ev.Type)),
		zap.Int("subscribers", len(s.subs)),
	)

	for _, ch := range s.subs {
		select {
		case ch <- ev:
		default:
		}
	}
}

func (s *Session) makeSnapshot(at time.Time) Snapshot {
	return Snapshot{
		TenderID:     s.cfg.TenderID,
		Status:       s.status,
		StartPrice:   s.startPrice,
		Step:         s.step,
		CurrentPrice: s.currentPrice,
		WinnerID:     s.winnerID,
		StartAt:      s.cfg.StartAt,
		EndAt:        s.cfg.EndAt,
		UpdatedAt:    at,
		LatestBid: LatestBid{
			ID:        s.latestBid.ID,
			CompanyID: s.latestBid.CompanyID,
			PersonID:  s.latestBid.PersonID,
			BidAmount: s.latestBid.BidAmount,
			BidAt:     s.latestBid.BidAt,
		},
	}
}

func (s *Session) closeAllSubs() {
	for id, ch := range s.subs {
		delete(s.subs, id)
		close(ch)
	}
	s.subCnt.Store(0)
}

func (s *Session) makeSubID(tenderID uuid.UUID, seq uint64) string {
	return tenderID.String() + "#" + s.gen(seq)
}

func (s *Session) gen(v uint64) string {
	if v == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = byte('0' + (v % 10))
		v /= 10
	}
	return string(buf[i:])
}
