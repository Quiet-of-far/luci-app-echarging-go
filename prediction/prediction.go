package prediction

import (
	"errors"
	"sort"
	"time"

	"luci-app-echarging-go/models"
)

// Calculate computes consumption rate and remaining time.
func Calculate(records []models.ElectricityRecord, customDailyConsumption float64) (*models.PredictionResult, error) {
	if len(records) == 0 {
		return nil, errors.New("no records available")
	}

	normalized := normalizeRecords(records)
	if len(normalized) == 0 {
		return nil, errors.New("no records available")
	}

	current := normalized[0]
	now := effectiveTime(current)

	if customDailyConsumption > 0 {
		return buildResult(current.RemainingKWh, customDailyConsumption, now, len(normalized)), nil
	}

	dailyRate, sampleCount := computeRate(normalized)
	if sampleCount < 2 {
		return nil, errors.New("insufficient data for prediction (need at least 2 samples in a consumption segment)")
	}
	if dailyRate <= 0 {
		return nil, errors.New("unable to compute positive consumption rate")
	}

	return buildResult(current.RemainingKWh, dailyRate, now, sampleCount), nil
}

func normalizeRecords(records []models.ElectricityRecord) []models.ElectricityRecord {
	sorted := append([]models.ElectricityRecord(nil), records...)
	sort.SliceStable(sorted, func(i, j int) bool {
		left := effectiveTime(sorted[i])
		right := effectiveTime(sorted[j])
		if left.Equal(right) {
			return sorted[i].QueryTime.After(sorted[j].QueryTime)
		}
		return left.After(right)
	})

	seen := make(map[time.Time]bool, len(records))
	result := make([]models.ElectricityRecord, 0, len(records))

	for _, record := range sorted {
		ts := effectiveTime(record).UTC()
		if seen[ts] {
			continue
		}
		seen[ts] = true
		result = append(result, record)
	}

	return result
}

func computeRate(records []models.ElectricityRecord) (float64, int) {
	segments := make([][]models.ElectricityRecord, 0, 1)
	segment := make([]models.ElectricityRecord, 0, len(records))
	segment = append(segment, records[0])

	for i := 1; i < len(records); i++ {
		older := records[i]
		newer := records[i-1]
		if older.RemainingKWh < newer.RemainingKWh {
			segments = append(segments, segment)
			segment = []models.ElectricityRecord{older}
			continue
		}
		segment = append(segment, older)
	}
	segments = append(segments, segment)

	bestSampleCount := 0
	for _, segment := range segments {
		if len(segment) < 2 {
			continue
		}
		if bestSampleCount == 0 {
			bestSampleCount = len(segment)
		}
		if rate := segmentRate(segment); rate > 0 {
			return rate, len(segment)
		}
	}

	return 0, bestSampleCount
}

func segmentRate(records []models.ElectricityRecord) float64 {
	newest := records[0]
	oldest := records[len(records)-1]

	duration := effectiveTime(newest).Sub(effectiveTime(oldest))
	if duration <= 0 {
		return 0
	}

	consumption := oldest.RemainingKWh - newest.RemainingKWh
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
