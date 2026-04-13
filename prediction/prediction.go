package prediction

import (
	"errors"
	"time"

	"luci-app-echarging-go/models"
)

// Calculate computes consumption rate and remaining time.
// records must be sorted by query_time descending (newest first).
func Calculate(records []models.ElectricityRecord, customDailyConsumption float64) (*models.PredictionResult, error) {
	if len(records) == 0 {
		return nil, errors.New("no records available")
	}

	current := records[0]
	now := effectiveTime(current)

	if customDailyConsumption > 0 {
		return buildResult(current.RemainingKWh, customDailyConsumption, now, len(records)), nil
	}

	normalized := dedupeRecords(records)
	if len(normalized) < 2 {
		return nil, errors.New("insufficient data for prediction (need at least 2 unique samples)")
	}

	dailyRate := computeRate(normalized)
	if dailyRate <= 0 {
		return nil, errors.New("unable to compute positive consumption rate (possible recharge)")
	}

	return buildResult(current.RemainingKWh, dailyRate, now, len(normalized)), nil
}

func dedupeRecords(records []models.ElectricityRecord) []models.ElectricityRecord {
	seen := make(map[time.Time]bool, len(records))
	result := make([]models.ElectricityRecord, 0, len(records))

	for _, record := range records {
		ts := effectiveTime(record).UTC()
		if seen[ts] {
			continue
		}
		seen[ts] = true
		result = append(result, record)
	}

	return result
}

func computeRate(records []models.ElectricityRecord) float64 {
	newest := records[0]
	oldest := records[len(records)-1]

	duration := effectiveTime(newest).Sub(effectiveTime(oldest))
	if duration <= 0 {
		return 0
	}

	consumption := oldest.RemainingKWh - newest.RemainingKWh
	if consumption > 0 {
		return consumption / duration.Hours() * 24.0
	}

	var start, end int
	for i := 1; i < len(records); i++ {
		if records[i].RemainingKWh >= records[i-1].RemainingKWh {
			end = i
			continue
		}
		break
	}

	if end == start {
		return 0
	}

	duration = effectiveTime(records[start]).Sub(effectiveTime(records[end]))
	if duration <= 0 {
		return 0
	}

	consumption = records[end].RemainingKWh - records[start].RemainingKWh
	if consumption <= 0 {
		return 0
	}

	return consumption / duration.Hours() * 24.0
}

func buildResult(remainingKWh, dailyRate float64, now time.Time, sampleCount int) *models.PredictionResult {
	remainingDays := remainingKWh / dailyRate
	remainingHours := remainingDays * 24

	return &models.PredictionResult{
		DailyConsumptionKWh: dailyRate,
		RemainingDays:       remainingDays,
		RemainingHours:      remainingHours,
		EstimatedEmptyTime:  now.Add(time.Duration(remainingHours * float64(time.Hour))),
		SampleCount:         sampleCount,
	}
}

func effectiveTime(record models.ElectricityRecord) time.Time {
	if record.MeterTime != nil {
		return *record.MeterTime
	}
	return record.QueryTime
}
