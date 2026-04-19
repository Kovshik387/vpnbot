package repository

import (
	"database/sql"
	"fmt"
)

// PanelRepository хранит id одного «экранного» сообщения бота на пользователя (приватный чат).
type PanelRepository struct {
	db *sql.DB
}

func NewPanelRepository(db *sql.DB) *PanelRepository {
	return &PanelRepository{db: db}
}

func (r *PanelRepository) Init() error {
	_, err := r.db.Exec(`
create table if not exists ui_panel (
    user_id integer primary key not null
  , chat_id integer not null
  , message_id integer not null
)`)
	return err
}

func (r *PanelRepository) Set(userID, chatID int64, messageID int) error {
	_, err := r.db.Exec(`
insert into ui_panel (user_id, chat_id, message_id)
values (?, ?, ?)
on conflict (user_id) do update set chat_id = excluded.chat_id
     , message_id = excluded.message_id`,
		userID, chatID, messageID)
	if err != nil {
		return fmt.Errorf("ui_panel set: %w", err)
	}
	return nil
}

func (r *PanelRepository) Get(userID int64) (chatID int64, messageID int, ok bool) {
	row := r.db.QueryRow(`
   select up.chat_id as chat_id
        , up.message_id as message_id
     from ui_panel up
    where up.user_id = ?`, userID)
	var mid int
	if err := row.Scan(&chatID, &mid); err != nil {
		return 0, 0, false
	}
	return chatID, mid, true
}

func (r *PanelRepository) Clear(userID int64) {
	_, _ = r.db.Exec(`
delete
  from ui_panel as up
 where up.user_id = ?`, userID)
}
