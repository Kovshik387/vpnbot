package repository

import (
	"VpnBot/internal/domain/model"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"
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

func (r *UserRepository) EnsureSchema() error {
	log.Println("Проверяем и обновляем схему таблицы users...")

	if err := r.Init(); err != nil {
		return fmt.Errorf("не удалось создать таблицу: %w", err)
	}

	migrations := []struct {
		name string
		sql  string
	}{
		{
			name: "price",
			sql:  "alter table users add column price real not null default 150.0",
		},
		{
			name: "is_free",
			sql:  "alter table users add column is_free boolean not null default false",
		},
		{
			name: "payment_date",
			sql:  `alter table users add column payment_date timestamp default '2024-11-21 00:00:00'`,
		},
	}

	for _, migration := range migrations {
		if err := r.safeAddColumn(migration.name, migration.sql); err != nil {
			return fmt.Errorf("не удалось добавить поле %s: %w", migration.name, err)
		}
	}

	log.Println("Схема таблицы users успешно обновлена")
	return nil
}

func (r *UserRepository) safeAddColumn(columnName, alterSQL string) error {
	exists, err := r.columnExists(columnName)
	if err != nil {
		return err
	}

	if exists {
		log.Printf("Поле %s уже существует, пропускаем", columnName)
		return nil
	}

	log.Printf("Добавляем поле %s...", columnName)

	_, err = r.db.Exec(alterSQL)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate column name") {
			log.Printf("Поле %s уже существует (обнаружено по ошибке)", columnName)
			return nil
		}
		return err
	}

	log.Printf("Поле %s успешно добавлено", columnName)
	return nil
}

func (r *UserRepository) columnExists(columnName string) (bool, error) {
	var exists bool
	query := `
		select count(*) > 0 
		  from pragma_table_info('users') 
		 where name = ?
	`
	err := r.db.QueryRow(query, columnName).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists, nil
}

func (r *UserRepository) Insert(user string, uid int64) error {
	_, err := r.db.Exec(`
		insert into users (user_id, username, is_block, price, is_free, payment_date) 
		values (?, ?, ?, ?, ?, ?)`,
		uid, user, false, 0.0, false, time.Now())
	return err
}

func (r *UserRepository) UpdatePrice(username string, price float64) error {
	_, err := r.db.Exec(`
		update users 
		   set price = ?
		 where username = ?`,
		price, username)
	return err
}

func (r *UserRepository) UpdateTypePayment(username string, isFree bool) error {
	_, err := r.db.Exec(`
		update users 
		   set is_free = ?
		 where username = ?`,
		isFree, username)
	return err
}

func (r *UserRepository) UpdatePaymentDate(username string, date time.Time) error {
	_, err := r.db.Exec(`
update users set payment_date = ? where username = ?`, date, username)

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

func (r *UserRepository) GetPayment() ([]model.TgUserModel, error) {
	rows, err := r.db.Query(`
select user_id, username, is_block, price, is_free, payment_date from users
 where payment_date is not null and is_block = false and is_free = false
  and cast(strftime('%d', substr(payment_date, 1, 19)) as integer)
      = cast(strftime('%d', 'now') as integer)
 order by payment_date desc`)

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
		var paymentDate sql.NullTime

		err := rows.Scan(&u.Uid, &u.Username, &u.IsBlock, &u.Price, &u.IsFree, &paymentDate)
		if err != nil {
			return nil, err
		}

		if paymentDate.Valid {
			u.PaymentDate = &paymentDate.Time
		}
		users = append(users, u)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return users, nil
}

func (r *UserRepository) OverrideDate(date time.Time, dateOverride time.Time) error {
	_, err := r.db.Exec(`update users set payment_date = ? where payment_date = ?`, date, dateOverride)

	return err
}
