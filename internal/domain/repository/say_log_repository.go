package repository

import (
	"database/sql"
	"fmt"
	"strings"
)

type SayLogEntry struct {
	ID        int64
	CreatedAt string
	Kind      string
	Body      string
}

type SayLogRepository struct {
	db *sql.DB
}

func NewSayLogRepository(db *sql.DB) *SayLogRepository {
	return &SayLogRepository{db: db}
}

func (r *SayLogRepository) Init() error {
	_, err := r.db.Exec(`
create table if not exists say_logs (
    id integer primary key autoincrement
  , created_at text not null default (datetime('now'))
  , kind text not null
  , body text not null
)`)
	return err
}

func (r *SayLogRepository) Insert(kind, body string) error {
	kind = strings.TrimSpace(kind)
	body = strings.TrimSpace(body)
	if body == "" {
		body = "—"
	}
	if len(body) > 200_000 {
		body = body[:200_000] + "…"
	}
	_, err := r.db.Exec(`
insert into say_logs (kind, body)
values (?, ?)`, kind, body)
	if err != nil {
		return fmt.Errorf("say_logs insert: %w", err)
	}
	return nil
}

// ListDesc возвращает записи от новых к старым.
func (r *SayLogRepository) ListDesc(offset, limit int) ([]SayLogEntry, error) {
	if limit <= 0 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	rows, err := r.db.Query(`
   select sl.id as id
        , sl.created_at as created_at
        , sl.kind as kind
        , sl.body as body
     from say_logs sl
 order by sl.id desc
    limit ?
   offset ?`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var out []SayLogEntry
	for rows.Next() {
		var e SayLogEntry
		if err := rows.Scan(&e.ID, &e.CreatedAt, &e.Kind, &e.Body); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (r *SayLogRepository) GetByID(id int64) (SayLogEntry, bool) {
	row := r.db.QueryRow(`
   select sl.id as id
        , sl.created_at as created_at
        , sl.kind as kind
        , sl.body as body
     from say_logs sl
    where sl.id = ?`, id)
	var e SayLogEntry
	if err := row.Scan(&e.ID, &e.CreatedAt, &e.Kind, &e.Body); err != nil {
		return SayLogEntry{}, false
	}
	return e, true
}

func (r *SayLogRepository) Count() (int64, error) {
	var n int64
	err := r.db.QueryRow(`
   select count(*) as cnt
     from say_logs sl`).Scan(&n)
	return n, err
}
