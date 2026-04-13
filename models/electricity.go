package models

import "time"

type ElectricityRecord struct {
	ID           int64      `json:"id"`
	Building     string     `json:"building"`
	Room         string     `json:"room"`
	RemainingKWh float64    `json:"remaining_kwh"`
	QueryTime    time.Time  `json:"query_time"`
	MeterTime    *time.Time `json:"meter_time,omitempty"`
}

type PredictionResult struct {
	DailyConsumptionKWh float64   `json:"daily_consumption_kwh"`
	RemainingDays       float64   `json:"remaining_days"`
	RemainingHours      float64   `json:"remaining_hours"`
	EstimatedEmptyTime  time.Time `json:"predicted_empty_time"`
	SampleCount         int       `json:"sample_count"`
}

type RoomStatus struct {
	Label               string     `json:"label"`
	Building            string     `json:"building"`
	Room                string     `json:"room"`
	RemainingKWh        float64    `json:"remaining_kwh"`
	MeterTime           *time.Time `json:"meter_time,omitempty"`
	QueryTime           time.Time  `json:"query_time"`
	PredictedEmptyTime  *time.Time `json:"predicted_empty_time,omitempty"`
	DailyConsumptionKWh *float64   `json:"daily_consumption_kwh,omitempty"`
	PredictionStatus    string     `json:"prediction_status"`
	QueryStatus         string     `json:"query_status,omitempty"`
}

type QueryResult struct {
	Room   RoomStatus `json:"room"`
	Alerts []string   `json:"alerts,omitempty"`
}
