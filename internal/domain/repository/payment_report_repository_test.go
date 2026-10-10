package repository

import (
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

func TestPaymentReportInitPreservesLegacyPayments(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(`create table payment_confirms (id integer primary key autoincrement, user_id integer not null, username text not null, amount real not null, confirmed_at text not null default (datetime('now')))`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`insert into payment_confirms (user_id, username, amount, confirmed_at) values (101, 'alice', 250, '2026-04-10 00:00:00')`); err != nil {
		t.Fatal(err)
	}
	repo := NewPaymentReportRepository(db)
	for range 2 {
		if err := repo.Init(); err != nil {
			t.Fatal(err)
		}
	}
	entries, err := repo.Monthly("2026-04")
	if err != nil || len(entries) != 1 || entries[0].Username != "alice" {
		t.Fatalf("legacy payments not preserved: entries=%v err=%v", entries, err)
	}
	if confirmed, err := repo.HasConfirmedProof(101, 901); err != nil || confirmed {
		t.Fatalf("legacy payment should not claim a new proof: confirmed=%v err=%v", confirmed, err)
	}
}
