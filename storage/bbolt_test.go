package storage

import (
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"luci-app-5echarging-go/models"
)

func TestStoreLatestAndRecentRecords(t *testing.T) {
	store := newTestStore(t)
	base := time.Date(2026, 5, 2, 10, 0, 0, 0, time.UTC)

	for i := 0; i < 3; i++ {
		if err := store.InsertRecord(record("44", "1207", float64(10+i), base.Add(time.Duration(i)*time.Hour), nil)); err != nil {
			t.Fatalf("insert record %d: %v", i, err)
		}
	}

	latest, err := store.GetLatestRecord("44", "1207")
	if err != nil {
		t.Fatalf("latest record: %v", err)
	}
	if latest.RemainingKWh != 12 {
		t.Fatalf("latest remaining = %v, want 12", latest.RemainingKWh)
	}

	recent, err := store.GetRecentRecords("44", "1207", 2)
	if err != nil {
		t.Fatalf("recent records: %v", err)
	}
	if len(recent) != 2 {
		t.Fatalf("recent len = %d, want 2", len(recent))
	}
	if recent[0].RemainingKWh != 12 || recent[1].RemainingKWh != 11 {
		t.Fatalf("recent order = %v, %v; want newest first", recent[0].RemainingKWh, recent[1].RemainingKWh)
	}
}

func TestStoreInsertRecordIfChanged(t *testing.T) {
	store := newTestStore(t)
	meterTime := time.Date(2026, 5, 2, 9, 30, 0, 0, time.UTC)
	first := record("44", "1207", 15, time.Date(2026, 5, 2, 10, 0, 0, 0, time.UTC), &meterTime)

	inserted, err := store.InsertRecordIfChanged(first, 10)
	if err != nil {
		t.Fatalf("insert first: %v", err)
	}
	if !inserted {
		t.Fatal("first insert returned false")
	}

	duplicate := first
	duplicate.QueryTime = first.QueryTime.Add(time.Hour)
	inserted, err = store.InsertRecordIfChanged(duplicate, 10)
	if err != nil {
		t.Fatalf("insert duplicate: %v", err)
	}
	if inserted {
		t.Fatal("duplicate insert returned true")
	}

	changed := duplicate
	changed.RemainingKWh = 14.5
	inserted, err = store.InsertRecordIfChanged(changed, 10)
	if err != nil {
		t.Fatalf("insert changed: %v", err)
	}
	if !inserted {
		t.Fatal("changed insert returned false")
	}

	recent, err := store.GetRecentRecords("44", "1207", 10)
	if err != nil {
		t.Fatalf("recent records: %v", err)
	}
	if len(recent) != 2 {
		t.Fatalf("record count = %d, want 2", len(recent))
	}
}

func TestStorePrunesOldRecords(t *testing.T) {
	store := newTestStore(t)
	base := time.Date(2026, 5, 2, 10, 0, 0, 0, time.UTC)

	for i := 0; i < 5; i++ {
		inserted, err := store.InsertRecordIfChanged(record("44", "1207", float64(i), base.Add(time.Duration(i)*time.Hour), nil), 3)
		if err != nil {
			t.Fatalf("insert %d: %v", i, err)
		}
		if !inserted {
			t.Fatalf("insert %d returned false", i)
		}
	}

	recent, err := store.GetRecentRecords("44", "1207", 10)
	if err != nil {
		t.Fatalf("recent records: %v", err)
	}
	if len(recent) != 3 {
		t.Fatalf("record count = %d, want 3", len(recent))
	}
	if recent[0].RemainingKWh != 4 || recent[2].RemainingKWh != 2 {
		t.Fatalf("remaining records = %#v, want 4..2", recent)
	}
}

func TestStoreSeparatesRoomsAndNoRows(t *testing.T) {
	store := newTestStore(t)
	base := time.Date(2026, 5, 2, 10, 0, 0, 0, time.UTC)

	if _, err := store.GetLatestRecord("44", "1207"); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("empty latest err = %v, want sql.ErrNoRows", err)
	}

	if err := store.InsertRecord(record("44", "1207", 10, base, nil)); err != nil {
		t.Fatalf("insert first room: %v", err)
	}
	if err := store.InsertRecord(record("45", "1207", 20, base, nil)); err != nil {
		t.Fatalf("insert second room: %v", err)
	}

	firstRoom, err := store.GetRecentRecords("44", "1207", 10)
	if err != nil {
		t.Fatalf("first room records: %v", err)
	}
	if len(firstRoom) != 1 || firstRoom[0].RemainingKWh != 10 {
		t.Fatalf("first room records = %#v", firstRoom)
	}
}

func newTestStore(t *testing.T) *Store {
	t.Helper()

	store, err := New(filepath.Join(t.TempDir(), "5echarging.bbolt"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Fatalf("close store: %v", err)
		}
	})
	return store
}

func record(building, room string, remaining float64, queryTime time.Time, meterTime *time.Time) models.ElectricityRecord {
	return models.ElectricityRecord{
		Building:     building,
		Room:         room,
		RemainingKWh: remaining,
		QueryTime:    queryTime,
		MeterTime:    meterTime,
	}
}
