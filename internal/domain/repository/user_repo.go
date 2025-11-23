package repository

import (
	"VpnBot/internal/domain/model"
	"database/sql"
	"errors"
	"log"
)

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Init() error {
	query := `
create table if not exists users (
    user_id integer primary key,
    username text not null,
    is_block boolean not null
)`
	_, err := r.db.Exec(query)
	return err
}

func (r *UserRepository) Insert(user string, uid int64) error {
	_, err := r.db.Exec(`
insert into users (user_id, username, is_block) values (?, ?, ?)`, uid, user, false)

	return err
}

func (r *UserRepository) Block(userID int64, block bool) error {
	_, err := r.db.Exec(`
update users
   set is_block= ?
 where user_id = ?`, block, userID)

	return err
}

func (r *UserRepository) CheckBlock(userID int64) (bool, error) {
	row := r.db.QueryRow(`
select is_block
  from users
 where user_id = ? 
`, userID)

	var bl bool
	err := row.Scan(&bl)
	if err != nil {
		return false, err
	}

	return bl, nil
}

func (r *UserRepository) UserExist(userID int64) (bool, error) {
	row := r.db.QueryRow(`
select 1
  from users
 where user_id = ? 
	`, userID)

	var dummy int
	err := row.Scan(&dummy)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

func (r *UserRepository) GetUsernameByUserID(userID int64) (string, error) {
	row := r.db.QueryRow(`
        select username
          from users
         where user_id = ?;
    `, userID)

	var username string
	err := row.Scan(&username)

	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", err
	}

	return username, nil
}

func (r *UserRepository) GetBlocked() ([]model.TgUserModel, error) {
	rows, err := r.db.Query(`
select user_id, username, is_block
  from users
 where is_block = true`)

	if err != nil {
		return nil, err
	}
	defer func(rows *sql.Rows) {
		err := rows.Close()
		if err != nil {
			log.Println(err)
		}
	}(rows)

	var users []model.TgUserModel
	for rows.Next() {
		var u model.TgUserModel
		err := rows.Scan(&u.Uid, &u.Username, &u.IsBlock)
		if err != nil {
			return nil, err
		}
		users = append(users, u)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return users, nil
}

func (r *UserRepository) GetActive() ([]model.TgUserModel, error) {
	rows, err := r.db.Query(`
select user_id, username, is_block
  from users
 where is_block = false`)

	if err != nil {
		return nil, err
	}
	defer func(rows *sql.Rows) {
		err := rows.Close()
		if err != nil {
			log.Println(err)
		}
	}(rows)

	var users []model.TgUserModel
	for rows.Next() {
		var u model.TgUserModel
		err := rows.Scan(&u.Uid, &u.Username, &u.IsBlock)
		if err != nil {
			return nil, err
		}
		users = append(users, u)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return users, nil
}
