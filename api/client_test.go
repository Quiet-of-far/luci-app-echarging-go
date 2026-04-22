package api

import (
	"testing"
	"time"
)

func TestParseMeterTimeUsesShanghaiTimezone(t *testing.T) {
	parsed, err := parseMeterTime("2026-04-22 08:30:00")
	if err != nil {
		t.Fatalf("parseMeterTime returned error: %v", err)
	}

	want := time.Date(2026, 4, 22, 0, 30, 0, 0, time.UTC)
	if !parsed.UTC().Equal(want) {
		t.Fatalf("parsed UTC time = %s, want %s", parsed.UTC(), want)
	}
}
