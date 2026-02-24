package postgres

import (
	"context"
	"time"

	"auction-core/internal/auction"
	"auction-core/internal/metrics"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresBidRepository struct {
	db *pgxpool.Pool
}

func NewPostgresBidRepository(db *pgxpool.Pool) *PostgresBidRepository {
	return &PostgresBidRepository{db: db}
}

func (r *PostgresBidRepository) GetByID(ctx context.Context, bidID int64) (_ auction.Bid, err error) {
	started := time.Now()
	defer func() {
		metrics.ObserveDBQuery("get_bid_by_id", started, err)
	}()

	var b auction.Bid

	err = r.db.QueryRow(ctx, `SELECT id, tender_id, company_id, person_id, bid_amount, created_at FROM bids WHERE id = $1`, bidID).Scan(
		&b.ID,
		&b.TenderID,
		&b.CompanyID,
		&b.PersonID,
		&b.BidAmount,
		&b.CreatedAt,
	)

	if err != nil {
		return auction.Bid{}, err
	}

	return b, nil
}

func (r *PostgresBidRepository) ListBids(ctx context.Context, tenderID uuid.UUID) (_ []auction.Bid, err error) {
	started := time.Now()
	defer func() {
		metrics.ObserveDBQuery("list_bids", started, err)
	}()

	rows, err := r.db.Query(ctx, `SELECT id, tender_id, company_id, person_id, bid_amount, created_at FROM bids WHERE tender_id = $1 ORDER BY created_at ASC`, tenderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var bids []auction.Bid

	for rows.Next() {
		var b auction.Bid
		if err := rows.Scan(
			&b.ID,
			&b.TenderID,
			&b.CompanyID,
			&b.PersonID,
			&b.BidAmount,
			&b.CreatedAt,
		); err != nil {
			return nil, err
		}
		bids = append(bids, b)
	}

	err = rows.Err()
	if err != nil {
		return nil, err
	}
	return bids, nil
}
