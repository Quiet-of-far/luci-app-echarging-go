package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

var httpClient = &http.Client{Timeout: 10 * time.Second}

type electricityResponse struct {
	Success bool                  `json:"success"`
	Data    electricityResponseV2 `json:"data"`
	Message string                `json:"message"`
}

type electricityResponseV2 struct {
	RoomName string          `json:"roomName"`
	Resamp   json.RawMessage `json:"resamp"`
	Updatedt string          `json:"updatedt"`
}

type CurrentStatus struct {
	RoomName     string
	RemainingKWh float64
	MeterTime    *time.Time
}

func GetCurrentStatus(building, room string) (*CurrentStatus, error) {
	apiURL := "http://202.192.240.231/scp-api/electricity-recharge/getCurrentRemaining_v2"

	formData := url.Values{
		"userTypeID": {"1"},
		"building":   {building},
		"room":       {room},
	}

	resp, err := httpClient.PostForm(apiURL, formData)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result electricityResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}
	if !result.Success {
		return nil, fmt.Errorf("API 报错: %s", result.Message)
	}

	remaining, err := parseFloat(result.Data.Resamp)
	if err != nil {
		return nil, fmt.Errorf("解析剩余电量失败: %w", err)
	}

	var meterTime *time.Time
	if strings.TrimSpace(result.Data.Updatedt) != "" {
		parsed, err := time.ParseInLocation("2006-01-02 15:04:05", result.Data.Updatedt, time.Local)
		if err != nil {
			return nil, fmt.Errorf("解析抄表时间失败: %w", err)
		}
		meterTime = &parsed
	}

	return &CurrentStatus{
		RoomName:     result.Data.RoomName,
		RemainingKWh: remaining,
		MeterTime:    meterTime,
	}, nil
}

func parseFloat(raw json.RawMessage) (float64, error) {
	var asNumber float64
	if err := json.Unmarshal(raw, &asNumber); err == nil {
		return asNumber, nil
	}

	var asString string
	if err := json.Unmarshal(raw, &asString); err != nil {
		return 0, fmt.Errorf("unsupported numeric payload: %s", string(raw))
	}

	value, err := strconv.ParseFloat(strings.TrimSpace(asString), 64)
	if err != nil {
		return 0, err
	}
	return value, nil
}
