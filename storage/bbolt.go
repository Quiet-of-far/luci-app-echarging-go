package storage

import (
	"database/sql"
	"encoding/binary"
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"time"

	"luci-app-5echarging-go/models"

	"go.etcd.io/bbolt"
)

const balanceEpsilon = 1e-6

var roomsBucket = []byte("rooms")

type Store struct {
	db *bbolt.DB
}

func New(dbPath string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, err
	}

	db, err := bbolt.Open(dbPath, 0600, nil)
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
	return s.db.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(roomsBucket)
		return err
	})
}

func (s *Store) InsertRecord(r models.ElectricityRecord) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		bucket, err := roomBucket(tx, r.Building, r.Room, true)
		if err != nil {
			return err
		}
		return insertRecord(bucket, r)
	})
}

func (s *Store) InsertRecordIfChanged(r models.ElectricityRecord, maxRecordsPerRoom int) (bool, error) {
	inserted := false

	err := s.db.Update(func(tx *bbolt.Tx) error {
		bucket, err := roomBucket(tx, r.Building, r.Room, true)
		if err != nil {
			return err
		}

		latest, err := latestRecord(bucket)
		if err != nil && err != sql.ErrNoRows {
			return err
		}
		if err == nil && sameReading(*latest, r) {
			return nil
		}

		if err := insertRecord(bucket, r); err != nil {
			return err
		}
		inserted = true

		if maxRecordsPerRoom > 0 {
			return pruneOldRecords(bucket, maxRecordsPerRoom)
		}
		return nil
	})
	return inserted, err
}

func insertRecord(bucket *bbolt.Bucket, r models.ElectricityRecord) error {
	id, err := bucket.NextSequence()
	if err != nil {
		return err
	}
	r.ID = int64(id)

	payload, err := json.Marshal(r)
	if err != nil {
		return err
	}

	return bucket.Put(sequenceKey(id), payload)
}

func pruneOldRecords(bucket *bbolt.Bucket, maxRecords int) error {
	records := 0
	cursor := bucket.Cursor()
	for key, _ := cursor.First(); key != nil; key, _ = cursor.Next() {
		records++
	}

	for records > maxRecords {
		key, _ := cursor.First()
		if key == nil {
			return nil
		}
		if err := cursor.Delete(); err != nil {
			return err
		}
		records--
	}
	return nil
}

func (s *Store) GetRecentRecords(building, room string, limit int) ([]models.ElectricityRecord, error) {
	if limit <= 0 {
		return []models.ElectricityRecord{}, nil
	}

	records := make([]models.ElectricityRecord, 0, limit)
	err := s.db.View(func(tx *bbolt.Tx) error {
		bucket, err := roomBucket(tx, building, room, false)
		if err != nil {
			return err
		}
		if bucket == nil {
			return nil
		}

		cursor := bucket.Cursor()
		for key, value := cursor.Last(); key != nil && len(records) < limit; key, value = cursor.Prev() {
			record, err := decodeRecord(value)
			if err != nil {
				return err
			}
			records = append(records, record)
		}
		return nil
	})
	return records, err
}

func (s *Store) GetLatestRecord(building, room string) (*models.ElectricityRecord, error) {
	var record *models.ElectricityRecord

	err := s.db.View(func(tx *bbolt.Tx) error {
		bucket, err := roomBucket(tx, building, room, false)
		if err != nil {
			return err
		}
		if bucket == nil {
			return sql.ErrNoRows
		}

		record, err = latestRecord(bucket)
		return err
	})
	return record, err
}

func latestRecord(bucket *bbolt.Bucket) (*models.ElectricityRecord, error) {
	key, value := bucket.Cursor().Last()
	if key == nil {
		return nil, sql.ErrNoRows
	}

	record, err := decodeRecord(value)
	if err != nil {
		return nil, err
	}
	return &record, nil
}

func roomBucket(tx *bbolt.Tx, building, room string, create bool) (*bbolt.Bucket, error) {
	rooms := tx.Bucket(roomsBucket)
	if rooms == nil {
		if !create {
			return nil, nil
		}

		var err error
		rooms, err = tx.CreateBucketIfNotExists(roomsBucket)
		if err != nil {
			return nil, err
		}
	}

	key := roomBucketKey(building, room)
	if create {
		return rooms.CreateBucketIfNotExists(key)
	}
	return rooms.Bucket(key), nil
}

func roomBucketKey(building, room string) []byte {
	key := make([]byte, 0, len(building)+1+len(room))
	key = append(key, building...)
	key = append(key, 0)
	key = append(key, room...)
	return key
}

func sequenceKey(id uint64) []byte {
	key := make([]byte, 8)
	binary.BigEndian.PutUint64(key, id)
	return key
}

func decodeRecord(value []byte) (models.ElectricityRecord, error) {
	var record models.ElectricityRecord
	if err := json.Unmarshal(value, &record); err != nil {
		return models.ElectricityRecord{}, err
	}
	return record, nil
}

func sameReading(left, right models.ElectricityRecord) bool {
	if math.Abs(left.RemainingKWh-right.RemainingKWh) > balanceEpsilon {
		return false
	}
	return sameMeterTime(left.MeterTime, right.MeterTime)
}

func sameMeterTime(left, right *time.Time) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return left.UTC().Equal(right.UTC())
}

func (s *Store) Close() error {
	return s.db.Close()
}
