package repository

import (
	"VpnBot/internal/domain/model"
	"database/sql"
)

type ProxyRepository struct {
	db *sql.DB
}

func NewProxyRepository(db *sql.DB) *ProxyRepository {
	return &ProxyRepository{db: db}
}

func (r *ProxyRepository) Init() error {
	_, err := r.db.Exec(`
create table if not exists proxies (
    id integer primary key autoincrement
  , name text not null
  , url text not null
)`)
	return err
}

func (r *ProxyRepository) Create(name, proxyURL string) (model.Proxy, error) {
	result, err := r.db.Exec(`insert into proxies (name, url) values (?, ?)`, name, proxyURL)
	if err != nil {
		return model.Proxy{}, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return model.Proxy{}, err
	}
	return model.Proxy{ID: id, Name: name, URL: proxyURL}, nil
}

func (r *ProxyRepository) Update(id int64, name, proxyURL string) error {
	result, err := r.db.Exec(`update proxies set name = ?, url = ? where id = ?`, name, proxyURL, id)
	return proxyMutationResult(result, err)
}

func (r *ProxyRepository) Delete(id int64) error {
	result, err := r.db.Exec(`delete from proxies where id = ?`, id)
	return proxyMutationResult(result, err)
}

func proxyMutationResult(result sql.Result, err error) error {
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *ProxyRepository) GetByID(id int64) (model.Proxy, error) {
	var proxy model.Proxy
	err := r.db.QueryRow(`select id, name, url from proxies where id = ?`, id).
		Scan(&proxy.ID, &proxy.Name, &proxy.URL)
	return proxy, err
}

func (r *ProxyRepository) ListPage(offset, limit int) ([]model.Proxy, error) {
	if offset < 0 {
		offset = 0
	}
	if limit <= 0 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}
	rows, err := r.db.Query(`select id, name, url from proxies order by id asc limit ? offset ?`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	proxies := make([]model.Proxy, 0)
	for rows.Next() {
		var proxy model.Proxy
		if err := rows.Scan(&proxy.ID, &proxy.Name, &proxy.URL); err != nil {
			return nil, err
		}
		proxies = append(proxies, proxy)
	}
	return proxies, rows.Err()
}

func (r *ProxyRepository) Count() (int64, error) {
	var count int64
	err := r.db.QueryRow(`select count(*) from proxies`).Scan(&count)
	return count, err
}
