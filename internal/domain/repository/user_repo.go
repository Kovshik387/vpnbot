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
    user_id integer primary key
  , username text not null
  , is_block boolean not null
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
			sql:  "alter table users add column price real not null default 250.0",
		},
		{
			name: "is_free",
			sql:  "alter table users add column is_free boolean not null default false",
		},
		{
			name: "payment_date",
			sql:  `alter table users add column payment_date timestamp default '2024-11-21 00:00:00'`,
		},
		{
			name: "payment_reminder_stage",
			sql:  `alter table users add column payment_reminder_stage integer not null default 0`,
		},
		{
			name: "awaiting_payment_screenshot",
			sql:  `alter table users add column awaiting_payment_screenshot integer not null default 0`,
		},
		{
			name: "payment_access_revoked",
			sql:  `alter table users add column payment_access_revoked boolean not null default 0`,
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
   select count(*) > 0 as column_exists
     from pragma_table_info('users') pt
    where pt.name = ?
`
	err := r.db.QueryRow(query, columnName).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists, nil
}

func (r *UserRepository) Insert(user string, uid int64, paymentDate time.Time) error {
	_, err := r.db.Exec(`
insert into users (user_id, username, is_block, price, is_free, payment_date)
values (?, ?, ?, ?, ?, ?)`,
		uid, user, false, 250.0, false, paymentDate)
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
update users
   set payment_date = ?
     , payment_reminder_stage = 0
     , payment_access_revoked = 0
 where username = ?`, date, username)

	return err
}

func (r *UserRepository) Block(userID int64, block bool) error {
	_, err := r.db.Exec(`
update users
   set is_block = ?
 where user_id = ?`, block, userID)

	return err
}

func (r *UserRepository) CheckBlock(userID int64) (bool, error) {
	row := r.db.QueryRow(`
   select us.is_block as is_block
     from users us
    where us.user_id = ?
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
   select 1 as one
     from users us
    where us.user_id = ?
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
   select us.username as username
     from users us
    where us.user_id = ?;
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

func (r *UserRepository) GetPriceByUserID(userID int64) (float64, error) {
	row := r.db.QueryRow(`
   select us.price as price
     from users us
    where us.user_id = ?`, userID)

	var price float64
	err := row.Scan(&price)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}

	return price, nil
}

func (r *UserRepository) GetPriceByUsername(username string) (float64, error) {
	row := r.db.QueryRow(`
   select us.price as price
     from users us
    where us.username = ?`, username)

	var price float64
	err := row.Scan(&price)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}

	return price, nil
}

func (r *UserRepository) GetBlocked() ([]model.TgUserModel, error) {
	rows, err := r.db.Query(`
   select us.user_id as user_id
        , us.username as username
        , us.is_block as is_block
     from users us
    where us.is_block = true /* blocked */`)

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
   select us.user_id as user_id
        , us.username as username
        , us.is_block as is_block
     from users us
    where us.is_block = false /* active */`)

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
   select us.user_id as user_id
        , us.username as username
        , us.is_block as is_block
        , us.price as price
        , us.is_free as is_free
        , us.payment_date as payment_date
     from users us
    where us.payment_date is not null
      and us.is_block = false /* active */
      and us.is_free = false /* paid tier */
      and date(substr(us.payment_date, 1, 19)) = date('now')
 order by us.payment_date desc`)

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
	oldDateStr := date.Format("2006-01-02")

	_, err := r.db.Exec(
		`
update users
   set payment_date = ?
     , payment_reminder_stage = 0
     , payment_access_revoked = 0
 where date(payment_date) = ?`, dateOverride, oldDateStr)

	return err
}

func (r *UserRepository) AddCompensationDays(daysCount int) error {
	_, err := r.db.Exec(`
update users
   set payment_date = datetime(
           coalesce(payment_date, '2024-11-21 00:00:00'), '+' || ? || ' days')
     , payment_reminder_stage = 0
 where is_free = 0 /* paid tier */
`, daysCount)

	return err
}

// ListPaidUsersForBilling — платные активные (не заблокированы админом), с датой оплаты.
func (r *UserRepository) ListPaidUsersForBilling() ([]model.TgUserModel, error) {
	rows, err := r.db.Query(`
   select us.user_id as user_id
        , us.username as username
        , us.is_block as is_block
        , us.price as price
        , us.is_free as is_free
        , us.payment_date as payment_date
        , coalesce(us.payment_reminder_stage, 0) as payment_reminder_stage
        , coalesce(us.payment_access_revoked, 0) as payment_access_revoked
        , coalesce(us.awaiting_payment_screenshot, 0) as awaiting_payment_screenshot
     from users us
    where us.is_free = 0
      and us.is_block = 0
      and us.payment_date is not null`)
	if err != nil {
		return nil, err
	}
	defer func(rows *sql.Rows) {
		_ = rows.Close()
	}(rows)

	var users []model.TgUserModel
	for rows.Next() {
		var u model.TgUserModel
		var paymentDate sql.NullTime
		var awaiting int
		err := rows.Scan(&u.Uid, &u.Username, &u.IsBlock, &u.Price, &u.IsFree, &paymentDate,
			&u.PaymentReminderStage, &u.PaymentAccessRevoked, &awaiting)
		if err != nil {
			return nil, err
		}
		if paymentDate.Valid {
			u.PaymentDate = &paymentDate.Time
		}
		u.AwaitingPaymentScreenshot = awaiting != 0
		users = append(users, u)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return users, nil
}

func (r *UserRepository) CheckPaymentRevoked(userID int64) (bool, error) {
	row := r.db.QueryRow(`
   select coalesce(us.payment_access_revoked, 0) as payment_access_revoked
     from users us
    where us.user_id = ?`, userID)
	var v bool
	err := row.Scan(&v)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return v, nil
}

func (r *UserRepository) SetPaymentAccessRevoked(userID int64, revoked bool) error {
	_, err := r.db.Exec(`
update users
   set payment_access_revoked = ?
 where user_id = ?`, revoked, userID)
	return err
}

func (r *UserRepository) SetAwaitingPaymentScreenshot(userID int64, v bool) error {
	val := 0
	if v {
		val = 1
	}
	_, err := r.db.Exec(`
update users
   set awaiting_payment_screenshot = ?
 where user_id = ?`, val, userID)
	return err
}

func (r *UserRepository) GetAwaitingPaymentScreenshot(userID int64) (bool, error) {
	row := r.db.QueryRow(`
   select coalesce(us.awaiting_payment_screenshot, 0) as awaiting_payment_screenshot
     from users us
    where us.user_id = ?`, userID)
	var n int
	err := row.Scan(&n)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return n != 0, nil
}

func (r *UserRepository) SetPaymentReminderStage(userID int64, stage int) error {
	_, err := r.db.Exec(`
update users
   set payment_reminder_stage = ?
 where user_id = ?`, stage, userID)
	return err
}

func (r *UserRepository) IsPaidSubscription(userID int64) (bool, error) {
	row := r.db.QueryRow(`
   select coalesce(us.is_free, 1) as is_free
     from users us
    where us.user_id = ?`, userID)
	var isFree bool
	err := row.Scan(&isFree)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return !isFree, nil
}

func (r *UserRepository) ApplyConfirmedPaymentExtension(userID int64, newPaymentDate time.Time) error {
	_, err := r.db.Exec(`
update users
   set payment_date = ?
     , payment_reminder_stage = 0
     , payment_access_revoked = 0
     , awaiting_payment_screenshot = 0
 where user_id = ?`, newPaymentDate, userID)
	return err
}

func (r *UserRepository) GetPaymentDateByUserID(userID int64) (*time.Time, error) {
	row := r.db.QueryRow(`
   select us.payment_date as payment_date
     from users us
    where us.user_id = ?`, userID)
	var t sql.NullTime
	err := row.Scan(&t)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if !t.Valid {
		return nil, nil
	}
	return &t.Time, nil
}

func (r *UserRepository) GetPaymentDateByUsername(username string) (*time.Time, error) {
	row := r.db.QueryRow(`
   select us.payment_date as payment_date
     from users us
    where us.username = ?`, username)
	var t sql.NullTime
	err := row.Scan(&t)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if !t.Valid {
		return nil, nil
	}
	return &t.Time, nil
}

// NormalizePaymentDatesToCurrentMonth приводит payment_date к текущему году/месяцу,
// сохраняя день месяца (и безопасно ограничивая его длиной месяца).
func (r *UserRepository) NormalizePaymentDatesToCurrentMonth(now time.Time) (int64, error) {
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	rows, err := r.db.Query(`
   select us.user_id as user_id
        , us.payment_date as payment_date
     from users us
    where us.is_free = 0
      and us.payment_date is not null`)
	if err != nil {
		return 0, err
	}
	defer func() { _ = rows.Close() }()

	type rec struct {
		userID int64
		date   time.Time
	}
	var users []rec
	for rows.Next() {
		var userID int64
		var dt sql.NullTime
		if err := rows.Scan(&userID, &dt); err != nil {
			return 0, err
		}
		if !dt.Valid {
			continue
		}
		users = append(users, rec{userID: userID, date: dt.Time})
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}

	tx, err := r.db.Begin()
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback() }()

	var affected int64
	for _, u := range users {
		day := u.date.Day()
		target := clampDateByDay(today.Year(), today.Month(), day, today.Location())
		// Если дата уже в прошлом относительно сегодняшнего дня, переносим на следующий месяц,
		// чтобы пользователь не платил второй раз в этом же цикле.
		if target.Before(today) {
			nextMonth := today.AddDate(0, 1, 0)
			target = clampDateByDay(nextMonth.Year(), nextMonth.Month(), day, today.Location())
		}

		if _, err := tx.Exec(`
update users
   set payment_date = ?
     , payment_reminder_stage = 0
     , payment_access_revoked = 0
 where user_id = ?`, target, u.userID); err != nil {
			return 0, err
		}
		affected++
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return affected, nil
}

func clampDateByDay(year int, month time.Month, day int, loc *time.Location) time.Time {
	lastDay := time.Date(year, month+1, 0, 0, 0, 0, 0, loc).Day()
	if day > lastDay {
		day = lastDay
	}
	if day < 1 {
		day = 1
	}
	return time.Date(year, month, day, 0, 0, 0, 0, loc)
}
