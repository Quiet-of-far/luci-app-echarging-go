package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

var httpClient = &http.Client{Timeout: 10 * time.Second}

type ElectricityResponse struct {
	Success bool        `json:"success"`
	Data    json.Number `json:"data"`
	Message string      `json:"message"`
}

func GetBalance(building, room string) (string, error) {
	apiURL := "http://202.192.240.231/scp-api/electricity-recharge/getCurrentRemaining"

	formData := url.Values{
		"userTypeID": {"1"},
		"building":   {building},
		"room":       {room},
	}

	resp, err := httpClient.PostForm(apiURL, formData)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var result ElectricityResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	if !result.Success {
		return "", fmt.Errorf("API 报错: %s", result.Message)
	}

	return result.Data.String(), nil
}
