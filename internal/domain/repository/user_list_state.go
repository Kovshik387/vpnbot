package repository

import (
	"database/sql"
	"encoding/json"
	"errors"
)

// UserListState keeps searches and the exact usernames behind the current buttons.
type UserListState struct {
	Token      string   `json:"token"`
	Query      string   `json:"query"`
	ActiveOnly bool     `json:"active_only"`
	Offset     int      `json:"offset"`
	Usernames  []string `json:"usernames"`
}

func (r *PanelRepository) initUserListState() error {
	_, err := r.db.Exec(`create table if not exists ui_user_list (
    user_id integer primary key not null,
    state text not null
)`)
	return err
}

func (r *PanelRepository) SetUserListState(userID int64, state UserListState) error {
	encoded, err := json.Marshal(state)
	if err != nil {
		return err
	}
	_, err = r.db.Exec(`insert into ui_user_list (user_id, state) values (?, ?)
on conflict (user_id) do update set state = excluded.state`, userID, string(encoded))
	return err
}

func (r *PanelRepository) GetUserListState(userID int64) (UserListState, error) {
	var encoded string
	err := r.db.QueryRow(`select state from ui_user_list where user_id = ?`, userID).Scan(&encoded)
	if errors.Is(err, sql.ErrNoRows) {
		return UserListState{}, nil
	}
	if err != nil {
		return UserListState{}, err
	}
	var state UserListState
	err = json.Unmarshal([]byte(encoded), &state)
	return state, err
}
