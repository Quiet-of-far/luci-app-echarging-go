package prediction

import (
	"errors"
	"math"
	"testing"
	"time"

	"luci-app-echarging-go/models"
)

func TestCalculateNormalConsumption(t *testing.T) {
	records := []models.ElectricityRecord{
		recordAt("2026-04-03T00:00:00Z", 90),
		recordAt("2026-04-01T00:00:00Z", 100),
	}

	result := mustCalculate(t, records)

	assertClose(t, result.DailyConsumptionKWh, 5)
	if result.SampleCount != 2 {
		t.Fatalf("SampleCount = %d, want 2", result.SampleCount)
	}
}

func TestCalculateAveragesSegmentsAcrossRechargeWhenAvailable(t *testing.T) {
	records := []models.ElectricityRecord{
		recordAt("2026-04-04T00:00:00Z", 115),
		recordAt("2026-04-03T00:00:00Z", 120),
		recordAt("2026-04-02T00:00:00Z", 40),
		recordAt("2026-04-01T00:00:00Z", 45),
	}

	result := mustCalculate(t, records)

	assertClose(t, result.DailyConsumptionKWh, 5)
	if result.SampleCount != 4 {
		t.Fatalf("SampleCount = %d, want 4", result.SampleCount)
	}
}

func TestCalculateFallsBackToPreRechargeSegment(t *testing.T) {
	records := []models.ElectricityRecord{
		recordAt("2026-04-04T00:00:00Z", 120),
		recordAt("2026-04-03T00:00:00Z", 40),
		recordAt("2026-04-02T00:00:00Z", 45),
	}

	result := mustCalculate(t, records)

	assertClose(t, result.DailyConsumptionKWh, 5)
	assertClose(t, result.RemainingDays, 24)
	if result.SampleCount != 2 {
		t.Fatalf("SampleCount = %d, want 2", result.SampleCount)
	}
}

func TestCalculateUsesWeightedSegmentsAcrossMultipleRecharges(t *testing.T) {
	records := []models.ElectricityRecord{
		recordAt("2026-04-06T00:00:00Z", 200),
		recordAt("2026-04-05T00:00:00Z", 80),
		recordAt("2026-04-04T00:00:00Z", 90),
		recordAt("2026-04-03T00:00:00Z", 100),
		recordAt("2026-04-02T00:00:00Z", 20),
		recordAt("2026-04-01T00:00:00Z", 25),
	}

	result := mustCalculate(t, records)

	assertClose(t, result.DailyConsumptionKWh, 25.0/3.0)
	if result.SampleCount != 5 {
		t.Fatalf("SampleCount = %d, want 5", result.SampleCount)
	}
}

func TestCalculateDedupesByEffectiveTime(t *testing.T) {
	meterTime := mustParseTime(t, "2026-04-02T00:00:00Z")
	records := []models.ElectricityRecord{
		{
			RemainingKWh: 90,
			QueryTime:    mustParseTime(t, "2026-04-02T01:00:00Z"),
			MeterTime:    &meterTime,
		},
		{
			RemainingKWh: 91,
			QueryTime:    mustParseTime(t, "2026-04-02T00:30:00Z"),
			MeterTime:    &meterTime,
		},
		recordAt("2026-04-01T00:00:00Z", 100),
	}

	result := mustCalculate(t, records)

	assertClose(t, result.DailyConsumptionKWh, 10)
	if result.SampleCount != 2 {
		t.Fatalf("SampleCount = %d, want 2", result.SampleCount)
	}
}

func TestCalculateSortsByEffectiveTime(t *testing.T) {
	olderMeterTime := mustParseTime(t, "2026-04-01T00:00:00Z")
	newerMeterTime := mustParseTime(t, "2026-04-03T00:00:00Z")
	records := []models.ElectricityRecord{
		{
			RemainingKWh: 100,
			QueryTime:    mustParseTime(t, "2026-04-05T00:00:00Z"),
			MeterTime:    &olderMeterTime,
		},
		{
			RemainingKWh: 90,
			QueryTime:    mustParseTime(t, "2026-04-04T00:00:00Z"),
			MeterTime:    &newerMeterTime,
		},
	}

	result := mustCalculate(t, records)

	assertClose(t, result.DailyConsumptionKWh, 5)
	assertClose(t, result.RemainingDays, 18)
	wantEmptyTime := newerMeterTime.Add(18 * 24 * time.Hour)
	if !result.EstimatedEmptyTime.Equal(wantEmptyTime) {
		t.Fatalf("EstimatedEmptyTime = %s, want %s", result.EstimatedEmptyTime, wantEmptyTime)
	}
}

func TestCalculateReportsNoConsumptionObservedForFlatRechargeSegments(t *testing.T) {
	records := []models.ElectricityRecord{
		recordAt("2026-04-23T20:13:02Z", 223.33),
		recordAt("2026-04-23T12:13:02Z", 223.33),
		recordAt("2026-04-21T15:52:58Z", 14.33),
		recordAt("2026-04-21T15:52:58Z", 14.33),
	}

	_, err := Calculate(records, 0)
	if !errors.Is(err, ErrNoConsumptionObserved) {
		t.Fatalf("Calculate error = %v, want %v", err, ErrNoConsumptionObserved)
	}
}

func mustCalculate(t *testing.T, records []models.ElectricityRecord) *models.PredictionResult {
	t.Helper()
	result, err := Calculate(records, 0)
	if err != nil {
		t.Fatalf("Calculate returned error: %v", err)
	}
	return result
}

func recordAt(raw string, remaining float64) models.ElectricityRecord {
	ts, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		panic(err)
	}
	return models.ElectricityRecord{
		RemainingKWh: remaining,
		QueryTime:    ts,
		MeterTime:    &ts,
	}
}

func mustParseTime(t *testing.T, raw string) time.Time {
	t.Helper()
	ts, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		t.Fatalf("parse %q: %v", raw, err)
	}
	return ts
}

func assertClose(t *testing.T, got, want float64) {
	t.Helper()
	if math.Abs(got-want) > 0.000001 {
		t.Fatalf("got %f, want %f", got, want)
	}
}
