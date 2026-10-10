package repository

import (
	"database/sql"
	"errors"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

func openProxyTestRepository(t *testing.T, path string) (*sql.DB, *ProxyRepository) {
	t.Helper()
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	repo := NewProxyRepository(db)
	if err := repo.Init(); err != nil {
		t.Fatal(err)
	}
	return db, repo
}

func TestProxyRepositoryCRUDPersistsAndDoesNotReuseIDs(t *testing.T) {
	path := filepath.Join(t.TempDir(), "proxies.db")
	db, repo := openProxyTestRepository(t, path)
	first, err := repo.Create("Первый", "socks5://example.com:1080")
	if err != nil {
		t.Fatal(err)
	}
	second, err := repo.Create("O'Reilly; drop table proxies;", "https://example.com:443")
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.Update(first.ID, "Обновлён", "http://example.com:8080"); err != nil {
		t.Fatal(err)
	}
	// The existing SQLite file is reopened to verify restart persistence.
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	_, repo = openProxyTestRepository(t, path)
	got, err := repo.GetByID(first.ID)
	if err != nil || got.Name != "Обновлён" || got.URL != "http://example.com:8080" {
		t.Fatalf("updated proxy after reopen: %+v, %v", got, err)
	}
	if count, err := repo.Count(); err != nil || count != 2 {
		t.Fatalf("count = %d, err = %v", count, err)
	}
	if err := repo.Delete(second.ID); err != nil {
		t.Fatal(err)
	}
	third, err := repo.Create("Третий", "socks5h://example.com:1080")
	if err != nil {
		t.Fatal(err)
	}
	if third.ID <= second.ID {
		t.Fatalf("deleted ID was reused: second=%d, third=%d", second.ID, third.ID)
	}
	page, err := repo.ListPage(0, 10)
	if err != nil || len(page) != 2 || page[0].ID != first.ID || page[1].ID != third.ID {
		t.Fatalf("unexpected stable page: %+v, %v", page, err)
	}
	if _, err := repo.GetByID(second.ID); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("get deleted proxy: %v", err)
	}
	if err := repo.Update(second.ID, "Missing", "socks5://example.com:1080"); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("update missing proxy: %v", err)
	}
	if err := repo.Delete(second.ID); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("delete missing proxy: %v", err)
	}
}

func TestProxyRepositoryPagination(t *testing.T) {
	_, repo := openProxyTestRepository(t, ":memory:")
	for i := 0; i < 55; i++ {
		if _, err := repo.Create("Proxy", "socks5://example.com:1080"); err != nil {
			t.Fatal(err)
		}
	}
	page, err := repo.ListPage(2, 3)
	if err != nil || len(page) != 3 || page[0].ID != 3 || page[2].ID != 5 {
		t.Fatalf("offset page: %+v, %v", page, err)
	}
	page, err = repo.ListPage(-1, 0)
	if err != nil || len(page) != 10 || page[0].ID != 1 {
		t.Fatalf("default page: %+v, %v", page, err)
	}
	page, err = repo.ListPage(0, 1000)
	if err != nil || len(page) != 50 {
		t.Fatalf("capped page: len=%d, %v", len(page), err)
	}
	page, err = repo.ListPage(100, 5)
	if err != nil || len(page) != 0 {
		t.Fatalf("empty page: %+v, %v", page, err)
	}
}
