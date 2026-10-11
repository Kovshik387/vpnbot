package repository

import (
	"VpnBot/internal/domain/model"
	"database/sql"
	"errors"
	"fmt"
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
    poll_id text primary key
  , question text not null
  , is_anonymous boolean not null
  , allows_multiple boolean not null
  , created_at timestamp not null
  , total_voter_count integer not null default -1
  , is_closed boolean not null default false
  , chat_id integer not null default 0
  , message_id integer not null default 0
  , source_poll_id text not null default ''
);`

	optionsTable := `
create table if not exists poll_options (
    poll_id text not null
  , option_index integer not null
  , text text not null
  , votes integer not null default 0
  , primary key (poll_id, option_index)
  , foreign key (poll_id) references polls (poll_id) on delete cascade
);`

	if _, err := r.db.Exec(pollsTable); err != nil {
		return err
	}
	if _, err := r.db.Exec(optionsTable); err != nil {
		return err
	}
	columns := []struct{ name, definition string }{
		{"total_voter_count", "integer not null default -1"},
		{"is_closed", "boolean not null default false"},
		{"chat_id", "integer not null default 0"},
		{"message_id", "integer not null default 0"},
		{"source_poll_id", "text not null default ''"},
	}
	for _, column := range columns {
		var exists bool
		if err := r.db.QueryRow(`select count(*) > 0 from pragma_table_info('polls') where name = ?`, column.name).Scan(&exists); err != nil {
			return err
		}
		if !exists {
			if _, err := r.db.Exec(fmt.Sprintf("alter table polls add column %s %s", column.name, column.definition)); err != nil {
				return err
			}
		}
	}
	if _, err := r.db.Exec(`create unique index if not exists polls_source_id on polls(source_poll_id) where source_poll_id != ''`); err != nil {
		return err
	}

	return nil
}

func (r *PollRepository) UpsertPollResults(p *model.PollResult) error {
	return r.saveResults(p, true)
}

// Unknown polls are ignored: only a published bot poll has reliable updates.
func (r *PollRepository) UpdatePollResults(p *model.PollResult) error {
	return r.saveResults(p, false)
}

func (r *PollRepository) saveResults(p *model.PollResult, insert bool) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if tx != nil {
			_ = tx.Rollback()
		}
	}()

	if insert {
		_, err = tx.Exec(`
insert into polls (poll_id, question, is_anonymous, allows_multiple, created_at,
                   total_voter_count, is_closed, chat_id, message_id, source_poll_id)
values (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
on conflict (poll_id) do update set question = excluded.question,
    is_anonymous = excluded.is_anonymous, allows_multiple = excluded.allows_multiple,
    total_voter_count = excluded.total_voter_count, is_closed = excluded.is_closed`,
			p.PollID, p.Question, p.IsAnonymous, p.AllowsMultiple, time.Now(),
			p.TotalVoterCount, p.IsClosed, p.ChatID, p.MessageID, p.SourcePollID)
	} else {
		var result sql.Result
		result, err = tx.Exec(`update polls set question = ?, is_anonymous = ?, allows_multiple = ?,
    total_voter_count = ?, is_closed = ? where poll_id = ?`,
			p.Question, p.IsAnonymous, p.AllowsMultiple, p.TotalVoterCount, p.IsClosed, p.PollID)
		if err == nil {
			var affected int64
			affected, err = result.RowsAffected()
			if err == nil && affected == 0 {
				return nil
			}
		}
	}
	if err != nil {
		return err
	}

	// Replace a complete snapshot, including retracted votes and removed options.
	if _, err := tx.Exec(`delete from poll_options where poll_id = ?`, p.PollID); err != nil {
		return err
	}
	for _, opt := range p.Options {
		_, err = tx.Exec(`
insert into poll_options (poll_id, option_index, text, votes)
values (?, ?, ?, ?)
on conflict (poll_id, option_index) do update set text = excluded.text
     , votes = excluded.votes`,
			p.PollID, opt.OptionIndex, opt.Text, opt.Votes)
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
   select pl.poll_id as poll_id
        , pl.question as question
        , pl.is_anonymous as is_anonymous
        , pl.allows_multiple as allows_multiple
        , pl.created_at as created_at
        , pl.total_voter_count
        , pl.is_closed
        , pl.chat_id
        , pl.message_id
        , pl.source_poll_id
     from polls pl
    where pl.poll_id = ?`, pollID)

	var pr model.PollResult
	err := row.Scan(&pr.PollID, &pr.Question, &pr.IsAnonymous, &pr.AllowsMultiple, &pr.CreatedAt,
		&pr.TotalVoterCount, &pr.IsClosed, &pr.ChatID, &pr.MessageID, &pr.SourcePollID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	rows, err := r.db.Query(`
   select po.option_index as option_index
        , po.text as text
        , po.votes as votes
     from poll_options po
    where po.poll_id = ?
 order by po.option_index`, pollID)
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

func (r *PollRepository) FindPublishedPoll(sourceID string) (*model.PollResult, error) {
	var id string
	err := r.db.QueryRow(`select poll_id from polls
where message_id > 0 and (poll_id = ? or source_poll_id = ?) limit 1`, sourceID, sourceID).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return r.GetPollResults(id)
}

func (r *PollRepository) ListPolls(offset, limit int) ([]model.PollResult, error) {
	rows, err := r.db.Query(`
   select pl.poll_id as poll_id
        , pl.question as question
        , pl.is_anonymous as is_anonymous
        , pl.allows_multiple as allows_multiple
        , pl.created_at as created_at
        , pl.total_voter_count
        , pl.is_closed
        , pl.chat_id
        , pl.message_id
        , pl.source_poll_id
     from polls pl
 order by pl.created_at desc, pl.poll_id desc limit ? offset ?`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []model.PollResult

	for rows.Next() {
		var p model.PollResult
		err := rows.Scan(&p.PollID, &p.Question, &p.IsAnonymous, &p.AllowsMultiple, &p.CreatedAt,
			&p.TotalVoterCount, &p.IsClosed, &p.ChatID, &p.MessageID, &p.SourcePollID)
		if err != nil {
			return nil, err
		}

		result = append(result, p)
	}

	return result, rows.Err()
}
