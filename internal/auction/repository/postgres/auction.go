package postgres

import (
	"context"
	"errors"
	"time"

	"auction-core/internal/auction"
	"auction-core/internal/metrics"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepository struct {
	db *pgxpool.Pool
}

func NewAuctionPostgresRepository(db *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) CreateBidTx(
	ctx context.Context,
	tenderID uuid.UUID,
	companyID uuid.UUID,
	personID uuid.UUID,
	amount int64,
) (err error) {
	started := time.Now()
	defer func() {
		metrics.ObserveDBQuery("create_bid_tx", started, err)
	}()

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var bidID int64
	err = tx.QueryRow(ctx, `
		INSERT INTO bids (tender_id, company_id, person_id, bid_amount)
		VALUES ($1,$2,$3,$4)
		RETURNING id
	`, tenderID, companyID, personID, amount).Scan(&bidID)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `
		UPDATE auctions
		SET current_price = $1,
		    winner_id = $2,
		    winner_bid_id = $3,
		    updated_at = NOW()
		WHERE tender_id = $4
	`, amount, companyID, bidID, tenderID)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (r *PostgresRepository) Create(ctx context.Context, a *auction.PersistedAuction) error {
	started := time.Now()
	query := `
		INSERT INTO auctions (
			tender_id,
			start_price,
			step,
			start_at,
			end_at,
			created_by,
			status,
			current_price,
			winner_id,
			winner_bid_id,
			created_at,
			updated_at
		)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,NOW(),NOW())
	`

	_, err := r.db.Exec(ctx, query,
		a.TenderID,
		a.StartPrice,
		a.Step,
		a.StartAt,
		a.EndAt,
		a.CreatedBy,
		string(a.Status),
		a.StartPrice,
		a.WinnerID,
		a.WinnerBidID,
	)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			metrics.ObserveDBQuery("create_auction", started, auction.ErrAlreadyExists)
			return auction.ErrAlreadyExists
		}
	}

	metrics.ObserveDBQuery("create_auction", started, err)
	return err
}

func (r *PostgresRepository) GetByID(
	ctx context.Context,
	tenderID uuid.UUID,
) (_ *auction.PersistedAuction, err error) {
	started := time.Now()
	defer func() {
		metrics.ObserveDBQuery("get_auction_by_id", started, err)
	}()

	query := `
		SELECT tender_id,
		       start_price,
		       step,
		       start_at,
		       end_at,
		       created_by,
		       status,
		       current_price,
		       winner_id,
		       winner_bid_id,
		       created_at,
		       updated_at
		FROM auctions
		WHERE tender_id = $1
	`

	row := r.db.QueryRow(ctx, query, tenderID)

	var a auction.PersistedAuction
	var status string

	if err := row.Scan(
		&a.TenderID,
		&a.StartPrice,
		&a.Step,
		&a.StartAt,
		&a.EndAt,
		&a.CreatedBy,
		&status,
		&a.CurrentPrice,
		&a.WinnerID,
		&a.WinnerBidID,
		&a.CreatedAt,
		&a.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, auction.ErrNotFound
		}

		return nil, err
	}

	a.Status = auction.Status(status)
	return &a, nil
}

func (r *PostgresRepository) Update(ctx context.Context, a *auction.PersistedAuction) error {
	started := time.Now()
	query := `
		UPDATE auctions
		SET start_price = $1,
		    step = $2,
		    start_at = $3,
		    end_at = $4,
		    current_price = $5,
		    updated_at = NOW()
		WHERE tender_id = $6
	`

	tag, err := r.db.Exec(ctx, query,
		a.StartPrice,
		a.Step,
		a.StartAt,
		a.EndAt,
		a.CurrentPrice,
		a.TenderID,
	)
	if err != nil {
		metrics.ObserveDBQuery("update_auction", started, err)
		return err
	}
	if tag.RowsAffected() == 0 {
		metrics.ObserveDBQuery("update_auction", started, auction.ErrNotFound)
		return auction.ErrNotFound
	}
	metrics.ObserveDBQuery("update_auction", started, nil)
	return nil
}

func (r *PostgresRepository) UpdateStatus(
	ctx context.Context,
	tenderID uuid.UUID,
	status auction.Status,
) (err error) {
	started := time.Now()
	defer func() {
		metrics.ObserveDBQuery("update_auction_status", started, err)
	}()

	query := `
		UPDATE auctions
		SET status = $1,
		    updated_at = NOW()
		WHERE tender_id = $2
	`

	_, err = r.db.Exec(ctx, query, string(status), tenderID)
	return err
}

func (r *PostgresRepository) Delete(ctx context.Context, tenderID uuid.UUID) error {
	started := time.Now()
	query := `DELETE FROM auctions WHERE tender_id = $1`
	tag, err := r.db.Exec(ctx, query, tenderID)
	if err != nil {
		metrics.ObserveDBQuery("delete_auction", started, err)
		return err
	}
	if tag.RowsAffected() == 0 {
		metrics.ObserveDBQuery("delete_auction", started, auction.ErrNotFound)
		return auction.ErrNotFound
	}
	metrics.ObserveDBQuery("delete_auction", started, nil)
	return nil
}

func (r *PostgresRepository) FindStartingBetween(
	ctx context.Context,
	from time.Time,
	to time.Time,
) (_ []auction.PersistedAuction, err error) {
	started := time.Now()
	defer func() {
		metrics.ObserveDBQuery("find_starting_between", started, err)
	}()

	query := `
		SELECT tender_id,
		       start_price,
		       step,
		       start_at,
		       end_at,
		       created_by,
		       status,
		       current_price,
		       winner_id,
		       winner_bid_id,
		       created_at,
		       updated_at
		FROM auctions
		WHERE start_at BETWEEN $1 AND $2
		AND status IN ('Scheduled','Active')
	`

	rows, err := r.db.Query(ctx, query, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []auction.PersistedAuction

	for rows.Next() {
		var a auction.PersistedAuction
		var status string

		if err := rows.Scan(
			&a.TenderID,
			&a.StartPrice,
			&a.Step,
			&a.StartAt,
			&a.EndAt,
			&a.CreatedBy,
			&status,
			&a.CurrentPrice,
			&a.WinnerID,
			&a.WinnerBidID,
			&a.CreatedAt,
			&a.UpdatedAt,
		); err != nil {
			return nil, err
		}

		a.Status = auction.Status(status)
		result = append(result, a)
	}

	err = rows.Err()
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (r *PostgresRepository) List(ctx context.Context) (_ []auction.PersistedAuction, err error) {
	started := time.Now()
	defer func() {
		metrics.ObserveDBQuery("list_auctions", started, err)
	}()

	rows, err := r.db.Query(ctx, `
		SELECT 
			tender_id, start_price, step, start_at, end_at, created_by, 
			status, current_price, winner_id, winner_bid_id, created_at, updated_at
		FROM auctions
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var auctions []auction.PersistedAuction
	for rows.Next() {
		var a auction.PersistedAuction
		err := rows.Scan(
			&a.TenderID,
			&a.StartPrice,
			&a.Step,
			&a.StartAt,
			&a.EndAt,
			&a.CreatedBy,
			&a.Status,
			&a.CurrentPrice,
			&a.WinnerID,
			&a.WinnerBidID,
			&a.CreatedAt,
			&a.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		auctions = append(auctions, a)
	}

	err = rows.Err()
	if err != nil {
		return nil, err
	}
	return auctions, nil
}
