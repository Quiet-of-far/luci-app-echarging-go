package storage

import (
	"database/sql"
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
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS balance_records (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			building TEXT NOT NULL,
			room TEXT NOT NULL,
			balance REAL NOT NULL,
			query_time DATETIME NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_room_time ON balance_records(building, room, query_time);
	`)
	return err
}

func (s *Store) InsertRecord(r models.BalanceRecord) error {
	_, err := s.db.Exec(
		"INSERT INTO balance_records (building, room, balance, query_time) VALUES (?, ?, ?, ?)",
		r.Building, r.Room, r.Balance, r.QueryTime,
	)
	return err
}

func (s *Store) GetRecentRecords(building, room string, limit int) ([]models.BalanceRecord, error) {
	rows, err := s.db.Query(
		"SELECT id, building, room, balance, query_time FROM balance_records WHERE building = ? AND room = ? ORDER BY query_time DESC LIMIT ?",
		building, room, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []models.BalanceRecord
	for rows.Next() {
		var r models.BalanceRecord
		var qt string
		if err := rows.Scan(&r.ID, &r.Building, &r.Room, &r.Balance, &qt); err != nil {
			return nil, err
		}
		r.QueryTime, err = time.Parse("2006-01-02T15:04:05Z", qt)
		if err != nil {
			r.QueryTime, err = time.Parse("2006-01-02 15:04:05", qt)
			if err != nil {
				return nil, err
			}
		}
		records = append(records, r)
	}
	return records, rows.Err()
}

func (s *Store) Close() error {
	return s.db.Close()
}
