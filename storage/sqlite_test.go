package storage

import (
	"path/filepath"
	"testing"
	"time"

	"luci-app-echarging-go/models"
)

func TestInsertRecordIfChangedDedupesLatestReading(t *testing.T) {
	store := newTestStore(t)

	meterTime := mustTime(t, "2026-04-24T10:00:00Z")
	first := testRecord("44", "1207", 100, "2026-04-24T10:05:00Z", &meterTime)

	inserted, err := store.InsertRecordIfChanged(first, 500)
	if err != nil {
		t.Fatalf("first insert: %v", err)
	}
	if !inserted {
		t.Fatal("first insert inserted = false, want true")
	}

	duplicate := testRecord("44", "1207", 100, "2026-04-24T10:10:00Z", &meterTime)
	inserted, err = store.InsertRecordIfChanged(duplicate, 500)
	if err != nil {
		t.Fatalf("duplicate insert: %v", err)
	}
	if inserted {
		t.Fatal("duplicate insert inserted = true, want false")
	}
	if got := countRecords(t, store, "44", "1207"); got != 1 {
		t.Fatalf("record count = %d, want 1", got)
	}
}

func TestInsertRecordIfChangedInsertsWhenMeterTimeChanges(t *testing.T) {
	store := newTestStore(t)

	firstMeter := mustTime(t, "2026-04-24T10:00:00Z")
	secondMeter := mustTime(t, "2026-04-24T11:00:00Z")

	mustInsertChanged(t, store, testRecord("44", "1207", 100, "2026-04-24T10:05:00Z", &firstMeter), 500)
	inserted, err := store.InsertRecordIfChanged(testRecord("44", "1207", 100, "2026-04-24T11:05:00Z", &secondMeter), 500)
	if err != nil {
		t.Fatalf("insert changed meter time: %v", err)
	}
	if !inserted {
		t.Fatal("changed meter time inserted = false, want true")
	}
	if got := countRecords(t, store, "44", "1207"); got != 2 {
		t.Fatalf("record count = %d, want 2", got)
	}
}

func TestInsertRecordIfChangedInsertsWhenBalanceChanges(t *testing.T) {
	store := newTestStore(t)

	meterTime := mustTime(t, "2026-04-24T10:00:00Z")

	mustInsertChanged(t, store, testRecord("44", "1207", 100, "2026-04-24T10:05:00Z", &meterTime), 500)
	inserted, err := store.InsertRecordIfChanged(testRecord("44", "1207", 99.5, "2026-04-24T10:10:00Z", &meterTime), 500)
	if err != nil {
		t.Fatalf("insert changed balance: %v", err)
	}
	if !inserted {
		t.Fatal("changed balance inserted = false, want true")
	}
	if got := countRecords(t, store, "44", "1207"); got != 2 {
		t.Fatalf("record count = %d, want 2", got)
	}
}

func TestInsertRecordIfChangedScopesDedupingByRoom(t *testing.T) {
	store := newTestStore(t)

	meterTime := mustTime(t, "2026-04-24T10:00:00Z")

	mustInsertChanged(t, store, testRecord("44", "1207", 100, "2026-04-24T10:05:00Z", &meterTime), 500)
	inserted, err := store.InsertRecordIfChanged(testRecord("44", "1303", 100, "2026-04-24T10:05:00Z", &meterTime), 500)
	if err != nil {
		t.Fatalf("insert other room: %v", err)
	}
	if !inserted {
		t.Fatal("other room inserted = false, want true")
	}
	if got := countRecords(t, store, "44", "1207"); got != 1 {
		t.Fatalf("1207 record count = %d, want 1", got)
	}
	if got := countRecords(t, store, "44", "1303"); got != 1 {
		t.Fatalf("1303 record count = %d, want 1", got)
	}
}

func TestInsertRecordIfChangedPrunesPerRoom(t *testing.T) {
	store := newTestStore(t)

	mustInsertChanged(t, store, testRecord("44", "1207", 100, "2026-04-24T10:00:00Z", nil), 2)
	mustInsertChanged(t, store, testRecord("44", "1207", 99, "2026-04-24T11:00:00Z", nil), 2)
	mustInsertChanged(t, store, testRecord("44", "1303", 80, "2026-04-24T11:00:00Z", nil), 2)
	mustInsertChanged(t, store, testRecord("44", "1207", 98, "2026-04-24T12:00:00Z", nil), 2)

	if got := countRecords(t, store, "44", "1207"); got != 2 {
		t.Fatalf("1207 record count = %d, want 2", got)
	}
	if got := countRecords(t, store, "44", "1303"); got != 1 {
		t.Fatalf("1303 record count = %d, want 1", got)
	}

	latestRecords, err := store.GetRecentRecords("44", "1207", 10)
	if err != nil {
		t.Fatalf("GetRecentRecords: %v", err)
	}
	if len(latestRecords) != 2 {
		t.Fatalf("recent count = %d, want 2", len(latestRecords))
	}
	if latestRecords[0].RemainingKWh != 98 || latestRecords[1].RemainingKWh != 99 {
		t.Fatalf("remaining balances = %.2f, %.2f; want 98.00, 99.00", latestRecords[0].RemainingKWh, latestRecords[1].RemainingKWh)
	}
}

func TestInsertRecordIfChangedDoesNotPruneWhenLimitDisabled(t *testing.T) {
	store := newTestStore(t)

	mustInsertChanged(t, store, testRecord("44", "1207", 100, "2026-04-24T10:00:00Z", nil), 0)
	mustInsertChanged(t, store, testRecord("44", "1207", 99, "2026-04-24T11:00:00Z", nil), 0)
	mustInsertChanged(t, store, testRecord("44", "1207", 98, "2026-04-24T12:00:00Z", nil), 0)

	if got := countRecords(t, store, "44", "1207"); got != 3 {
		t.Fatalf("record count = %d, want 3", got)
	}
}

func newTestStore(t *testing.T) *Store {
	t.Helper()
	store, err := New(filepath.Join(t.TempDir(), "echarging.db"))
	if err != nil {
		t.Fatalf("New store: %v", err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Fatalf("Close store: %v", err)
		}
	})
	return store
}

func mustInsertChanged(t *testing.T, store *Store, record models.ElectricityRecord, maxRecordsPerRoom int) {
	t.Helper()
	inserted, err := store.InsertRecordIfChanged(record, maxRecordsPerRoom)
	if err != nil {
		t.Fatalf("InsertRecordIfChanged: %v", err)
	}
	if !inserted {
		t.Fatal("InsertRecordIfChanged inserted = false, want true")
	}
}

func countRecords(t *testing.T, store *Store, building, room string) int {
	t.Helper()
	var count int
	if err := store.db.QueryRow("SELECT COUNT(*) FROM balance_records WHERE building = ? AND room = ?", building, room).Scan(&count); err != nil {
		t.Fatalf("count records: %v", err)
	}
	return count
}

func testRecord(building, room string, remaining float64, queryTime string, meterTime *time.Time) models.ElectricityRecord {
	return models.ElectricityRecord{
		Building:     building,
		Room:         room,
		RemainingKWh: remaining,
		QueryTime:    mustParseRFC3339(queryTime),
		MeterTime:    meterTime,
	}
}

func mustTime(t *testing.T, raw string) time.Time {
	t.Helper()
	return mustParseRFC3339(raw)
}

func mustParseRFC3339(raw string) time.Time {
	parsed, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		panic(err)
	}
	return parsed
}
