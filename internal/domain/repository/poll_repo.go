package repository

import (
	"VpnBot/internal/domain/model"
	"database/sql"
	"errors"
	"time"
)

type PollRepository struct {
	db *sql.DB
}

func NewPollRepository(db *sql.DB) *PollRepository {
	return &PollRepository{db: db}
}

func (r *PollRepository) Init() error {
	pollsTable := `
create table if not exists polls (
	poll_id        text primary key,
	question       text not null,
	is_anonymous   boolean not null,
	allows_multiple boolean not null,
	created_at     timestamp not null
);`

	optionsTable := `
create table if not exists poll_options (
	poll_id      text not null,
	option_index integer not null,
	text         text not null,
	votes        integer not null default 0,
	primary key (poll_id, option_index),
	foreign key (poll_id) references polls(poll_id) on delete cascade
);`

	if _, err := r.db.Exec(pollsTable); err != nil {
		return err
	}
	if _, err := r.db.Exec(optionsTable); err != nil {
		return err
	}

	return nil
}

func (r *PollRepository) UpsertPollResults(
	pollID string,
	question string,
	isAnonymous bool,
	allowsMultiple bool,
	options []model.PollOptionResult,
) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if tx != nil {
			_ = tx.Rollback()
		}
	}()

	_, err = tx.Exec(`
insert into polls(poll_id, question, is_anonymous, allows_multiple, created_at)
values (?, ?, ?, ?, ?)
	on conflict(poll_id) do update set
		question        = excluded.question,
		is_anonymous    = excluded.is_anonymous,
		allows_multiple = excluded.allows_multiple;
`, pollID, question, isAnonymous, allowsMultiple, time.Now())
	if err != nil {
		return err
	}

	for _, opt := range options {
		_, err = tx.Exec(`
insert into poll_options(poll_id, option_index, text, votes)
values (?, ?, ?, ?)
	on conflict(poll_id, option_index) do update set
		text  = excluded.text,
		votes = excluded.votes;
`, pollID, opt.OptionIndex, opt.Text, opt.Votes)
		if err != nil {
			return err
		}
	}

	if err = tx.Commit(); err != nil {
		return err
	}
	tx = nil

	return nil
}

func (r *PollRepository) GetPollResults(pollID string) (*model.PollResult, error) {
	row := r.db.QueryRow(`
select poll_id, question, is_anonymous, allows_multiple, created_at
  from polls
 where poll_id = ?`, pollID)

	var pr model.PollResult
	err := row.Scan(&pr.PollID, &pr.Question, &pr.IsAnonymous, &pr.AllowsMultiple, &pr.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	rows, err := r.db.Query(`
select option_index, text, votes
  from poll_options
 where poll_id = ?
 order by option_index`, pollID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var options []model.PollOptionResult
	for rows.Next() {
		var o model.PollOptionResult
		if err := rows.Scan(&o.OptionIndex, &o.Text, &o.Votes); err != nil {
			return nil, err
		}
		options = append(options, o)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	pr.Options = options
	return &pr, nil
}

func (r *PollRepository) ListPolls() ([]model.PollResult, error) {
	rows, err := r.db.Query(`
select poll_id, question, is_anonymous, allows_multiple, created_at
  from polls
 order by created_at desc`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []model.PollResult

	for rows.Next() {
		var p model.PollResult
		err := rows.Scan(&p.PollID, &p.Question, &p.IsAnonymous, &p.AllowsMultiple, &p.CreatedAt)
		if err != nil {
			return nil, err
		}

		result = append(result, p)
	}

	return result, rows.Err()
}
