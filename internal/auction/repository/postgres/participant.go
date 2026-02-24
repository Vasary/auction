package postgres

import (
	"context"
	"time"

	"auction-core/internal/metrics"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresParticipantRepository struct {
	db *pgxpool.Pool
}

func NewPostgresParticipantRepository(db *pgxpool.Pool) *PostgresParticipantRepository {
	return &PostgresParticipantRepository{db: db}
}

func (r *PostgresParticipantRepository) AddParticipant(
	ctx context.Context,
	tenderID uuid.UUID,
	companyID uuid.UUID,
) (err error) {
	started := time.Now()
	defer func() {
		metrics.ObserveDBQuery("add_participant", started, err)
	}()

	_, err = r.db.Exec(ctx, `INSERT INTO auction_participants (tender_id, company_id) VALUES ($1,$2) ON CONFLICT DO NOTHING`, tenderID, companyID)

	return err
}

func (r *PostgresParticipantRepository) IsParticipant(
	ctx context.Context,
	tenderID uuid.UUID,
	companyID uuid.UUID,
) (_ bool, err error) {
	started := time.Now()
	defer func() {
		metrics.ObserveDBQuery("is_participant", started, err)
	}()

	var exists bool

	err = r.db.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM auction_participants WHERE tender_id = $1 AND company_id = $2)`, tenderID, companyID).Scan(&exists)

	return exists, err
}

func (r *PostgresParticipantRepository) ListParticipants(
	ctx context.Context,
	tenderID uuid.UUID,
) (_ []uuid.UUID, err error) {
	started := time.Now()
	defer func() {
		metrics.ObserveDBQuery("list_participants", started, err)
	}()

	rows, err := r.db.Query(ctx, `SELECT company_id FROM auction_participants WHERE tender_id = $1`, tenderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var participants []uuid.UUID
	for rows.Next() {
		var companyID uuid.UUID
		if err := rows.Scan(&companyID); err != nil {
			return nil, err
		}
		participants = append(participants, companyID)
	}

	err = rows.Err()
	if err != nil {
		return nil, err
	}

	return participants, nil
}
