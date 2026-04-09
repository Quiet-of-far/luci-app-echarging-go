package prediction

import (
	"errors"
	"time"

	"luci-app-echarging-go/models"
)

// Calculate computes consumption rate and remaining time.
// records must be sorted by query_time descending (newest first).
func Calculate(records []models.BalanceRecord, customDailyRate float64) (*models.PredictionResult, error) {
	if len(records) == 0 {
		return nil, errors.New("no records available")
	}

	currentBalance := records[0].Balance
	now := records[0].QueryTime

	if customDailyRate > 0 {
		return buildResult(currentBalance, customDailyRate, now, len(records)), nil
	}

	if len(records) < 2 {
		return nil, errors.New("insufficient data for prediction (need at least 2 records)")
	}

	dailyRate := computeRate(records)
	if dailyRate <= 0 {
		return nil, errors.New("unable to compute positive consumption rate (possible recharge)")
	}

	return buildResult(currentBalance, dailyRate, now, len(records)), nil
}

func computeRate(records []models.BalanceRecord) float64 {
	// Try using full range first (newest at index 0, oldest at end)
	newest := records[0]
	oldest := records[len(records)-1]

	duration := newest.QueryTime.Sub(oldest.QueryTime)
	if duration <= 0 {
		return 0
	}

	consumption := oldest.Balance - newest.Balance
	if consumption > 0 {
		return consumption / duration.Hours() * 24.0
	}

	// Full range has negative consumption (recharge happened).
	// Find longest monotonically decreasing run from newest backward.
	var start, end int
	for i := 1; i < len(records); i++ {
		if records[i].Balance >= records[i-1].Balance {
			// records[i] is older and has higher or equal balance — still decreasing over time
			end = i
		} else {
			break
		}
	}

	if end == start {
		return 0
	}

	duration = records[start].QueryTime.Sub(records[end].QueryTime)
	if duration <= 0 {
		return 0
	}

	consumption = records[end].Balance - records[start].Balance
	if consumption <= 0 {
		return 0
	}

	return consumption / duration.Hours() * 24.0
}

func buildResult(balance, dailyRate float64, now time.Time, sampleCount int) *models.PredictionResult {
	remainingDays := balance / dailyRate
	remainingHours := remainingDays * 24

	return &models.PredictionResult{
		DailyRate:      dailyRate,
		RemainingDays:  remainingDays,
		RemainingHours: remainingHours,
		EstimatedEmpty: now.Add(time.Duration(remainingHours * float64(time.Hour))),
		SampleCount:    sampleCount,
	}
}
