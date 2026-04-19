package repository

import "database/sql"

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
)`)
	return err
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
