package repository

import (
	"VpnBot/internal/domain/model"
	"database/sql"
	"errors"
	"time"
)

type CooldownRepository struct {
	db *sql.DB
}

func NewCooldownRepository(db *sql.DB) *CooldownRepository {
	return &CooldownRepository{db: db}
}

func (r *CooldownRepository) Init() error {
	query := `
create table if not exists cooldowns (
    user_id integer primary key
  , expires_at timestamp not null
);`
	_, err := r.db.Exec(query)
	return err
}

func (r *CooldownRepository) SetCooldown(userID int64, duration time.Duration) error {
	expiresAt := time.Now().Add(duration)
	_, err := r.db.Exec(`
insert into cooldowns (user_id, expires_at)
values (?, ?)
on conflict (user_id) do update set expires_at = excluded.expires_at`,
		userID, expiresAt)
	return err
}

func (r *CooldownRepository) GetCooldown(userID int64) (*model.Cooldown, error) {
	row := r.db.QueryRow(`
   select cd.user_id as user_id
        , cd.expires_at as expires_at
     from cooldowns cd
    where cd.user_id = ?`, userID)

	var cd model.Cooldown
	err := row.Scan(&cd.UserID, &cd.ExpiresAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &cd, nil
}

func (r *CooldownRepository) ClearExpired() error {
	_, err := r.db.Exec(`
delete
  from cooldowns as cd
 where cd.expires_at < ?`, time.Now())
	return err
}
