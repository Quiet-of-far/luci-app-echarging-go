package models

import "time"

type BalanceRecord struct {
	ID        int64     `json:"id"`
	Building  string    `json:"building"`
	Room      string    `json:"room"`
	Balance   float64   `json:"balance"`
	QueryTime time.Time `json:"query_time"`
}

type PredictionResult struct {
	DailyRate      float64   `json:"daily_rate"`
	RemainingDays  float64   `json:"remaining_days"`
	RemainingHours float64   `json:"remaining_hours"`
	EstimatedEmpty time.Time `json:"estimated_empty"`
	SampleCount    int       `json:"sample_count"`
}
