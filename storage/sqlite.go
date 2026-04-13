package storage

import (
	"database/sql"
	"fmt"
	"time"

	"luci-app-echarging-go/models"

	_ "modernc.org/sqlite"
)

type Store struct {
	db *sql.DB
}

func New(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}

	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) migrate() error {
	if _, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS balance_records (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			building TEXT NOT NULL,
			room TEXT NOT NULL,
			balance REAL NOT NULL,
			query_time DATETIME NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_room_time ON balance_records(building, room, query_time);
	`); err != nil {
		return err
	}

	exists, err := s.hasColumn("balance_records", "meter_time")
	if err != nil {
		return err
	}
	if !exists {
		if _, err := s.db.Exec(`ALTER TABLE balance_records ADD COLUMN meter_time DATETIME`); err != nil {
			return err
		}
	}

	return nil
}

func (s *Store) hasColumn(table, column string) (bool, error) {
	rows, err := s.db.Query(fmt.Sprintf("PRAGMA table_info(%s)", table))
	if err != nil {
		return false, err
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull int
		var dflt sql.NullString
		var pk int
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			return false, err
		}
		if name == column {
			return true, nil
		}
	}

	return false, rows.Err()
}

func (s *Store) InsertRecord(r models.ElectricityRecord) error {
	var meterTime any
	if r.MeterTime != nil {
		meterTime = r.MeterTime.UTC().Format(time.RFC3339)
	}

	_, err := s.db.Exec(
		"INSERT INTO balance_records (building, room, balance, query_time, meter_time) VALUES (?, ?, ?, ?, ?)",
		r.Building,
		r.Room,
		r.RemainingKWh,
		r.QueryTime.UTC().Format(time.RFC3339),
		meterTime,
	)
	return err
}

func (s *Store) GetRecentRecords(building, room string, limit int) ([]models.ElectricityRecord, error) {
	rows, err := s.db.Query(
		"SELECT id, building, room, balance, query_time, meter_time FROM balance_records WHERE building = ? AND room = ? ORDER BY query_time DESC LIMIT ?",
		building,
		room,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	records := make([]models.ElectricityRecord, 0, limit)
	for rows.Next() {
		record, err := scanRecord(rows)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}

	return records, rows.Err()
}

func (s *Store) GetLatestRecord(building, room string) (*models.ElectricityRecord, error) {
	rows, err := s.db.Query(
		"SELECT id, building, room, balance, query_time, meter_time FROM balance_records WHERE building = ? AND room = ? ORDER BY query_time DESC LIMIT 1",
		building,
		room,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, sql.ErrNoRows
	}

	record, err := scanRecord(rows)
	if err != nil {
		return nil, err
	}
	return &record, nil
}

func scanRecord(scanner interface{ Scan(dest ...any) error }) (models.ElectricityRecord, error) {
	var record models.ElectricityRecord
	var queryTime string
	var meterTime sql.NullString

	if err := scanner.Scan(
		&record.ID,
		&record.Building,
		&record.Room,
		&record.RemainingKWh,
		&queryTime,
		&meterTime,
	); err != nil {
		return models.ElectricityRecord{}, err
	}

	parsedQuery, err := parseSQLiteTime(queryTime)
	if err != nil {
		return models.ElectricityRecord{}, err
	}
	record.QueryTime = parsedQuery

	if meterTime.Valid && meterTime.String != "" {
		parsedMeter, err := parseSQLiteTime(meterTime.String)
		if err != nil {
			return models.ElectricityRecord{}, err
		}
		record.MeterTime = &parsedMeter
	}

	return record, nil
}

func parseSQLiteTime(raw string) (time.Time, error) {
	layouts := []string{
		time.RFC3339,
		"2006-01-02T15:04:05Z",
		"2006-01-02 15:04:05",
	}

	for _, layout := range layouts {
		if parsed, err := time.Parse(layout, raw); err == nil {
			return parsed, nil
		}
	}

	return time.Time{}, fmt.Errorf("unsupported time format: %s", raw)
}

func (s *Store) Close() error {
	return s.db.Close()
}
