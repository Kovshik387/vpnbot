package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

type PaymentConfirmEntry struct {
	ID          int64
	UserID      int64
	Username    string
	Amount      float64
	ConfirmedAt string
}

type PaymentReportRepository struct {
	db *sql.DB
}

func NewPaymentReportRepository(db *sql.DB) *PaymentReportRepository {
	return &PaymentReportRepository{db: db}
}

func (r *PaymentReportRepository) Init() error {
	_, err := r.db.Exec(`
create table if not exists payment_confirms (
    id integer primary key autoincrement
  , user_id integer not null
  , username text not null
  , amount real not null
  , confirmed_at text not null default (datetime('now'))
  , proof_message_id integer
)`)
	if err != nil {
		return err
	}
	var hasProofColumn bool
	if err := r.db.QueryRow(`select count(*) > 0 from pragma_table_info('payment_confirms') where name = 'proof_message_id'`).Scan(&hasProofColumn); err != nil {
		return err
	}
	if !hasProofColumn {
		if _, err := r.db.Exec(`alter table payment_confirms add column proof_message_id integer`); err != nil {
			return err
		}
	}
	_, err = r.db.Exec(`create unique index if not exists payment_confirms_proof on payment_confirms (user_id, proof_message_id)`)
	return err
}

func (r *PaymentReportRepository) HasConfirmedProof(userID int64, proofMessageID int) (bool, error) {
	var confirmed bool
	err := r.db.QueryRow(`select exists(select 1 from payment_confirms where user_id = ? and proof_message_id = ?)`, userID, proofMessageID).Scan(&confirmed)
	return confirmed, err
}

// ApplyConfirmedExtension записывает срок и журнал оплаты одной транзакцией.
// Ненулевой ID сообщения с чеком защищает от повторного подтверждения одной оплаты.
func (r *PaymentReportRepository) ApplyConfirmedExtension(userID int64, newPaymentDate time.Time, proofMessageID int) (bool, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return false, err
	}
	defer func() { _ = tx.Rollback() }()

	if proofMessageID > 0 {
		var confirmed bool
		if err := tx.QueryRow(`select exists(select 1 from payment_confirms where user_id = ? and proof_message_id = ?)`, userID, proofMessageID).Scan(&confirmed); err != nil {
			return false, err
		}
		if confirmed {
			return false, nil
		}
	}
	var username string
	var amount float64
	if err := tx.QueryRow(`select username, price from users where user_id = ?`, userID).Scan(&username, &amount); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, fmt.Errorf("пользователь %d не найден", userID)
		}
		return false, err
	}
	if _, err := tx.Exec(`update users set payment_date = ?, payment_reminder_stage = 0, payment_access_revoked = 0, awaiting_payment_screenshot = 0 where user_id = ?`, newPaymentDate, userID); err != nil {
		return false, err
	}
	var proofID any
	if proofMessageID > 0 {
		proofID = proofMessageID
	}
	if _, err := tx.Exec(`insert into payment_confirms (user_id, username, amount, proof_message_id) values (?, ?, ?, ?)`, userID, username, amount, proofID); err != nil {
		return false, err
	}
	if err := tx.Commit(); err != nil {
		return false, err
	}
	return true, nil
}

func (r *PaymentReportRepository) Add(userID int64, username string, amount float64) error {
	_, err := r.db.Exec(`
insert into payment_confirms (user_id, username, amount)
values (?, ?, ?)`, userID, username, amount)
	return err
}

func (r *PaymentReportRepository) Monthly(month string) ([]PaymentConfirmEntry, error) {
	rows, err := r.db.Query(`
   select pc.id as id
        , pc.user_id as user_id
        , pc.username as username
        , pc.amount as amount
        , pc.confirmed_at as confirmed_at
     from payment_confirms pc
    where strftime('%Y-%m', pc.confirmed_at) = ?
 order by pc.confirmed_at desc`, month)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var out []PaymentConfirmEntry
	for rows.Next() {
		var e PaymentConfirmEntry
		if err := rows.Scan(&e.ID, &e.UserID, &e.Username, &e.Amount, &e.ConfirmedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}
