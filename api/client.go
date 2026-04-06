package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// 结构体也搬过来，因为它属于 API 的数据协议
type ElectricityResponse struct {
	Success bool   `json:"success"`
	Data    string `json:"data"`
	Message string `json:"message"`
}

// 注意：函数名首字母大写（GetBalance），这样 main 包才能调用它
func GetBalance(building, room string) (string, error) {
	apiURL := "http://202.192.240.231/scp-api/electricity-recharge/getCurrentRemaining"

	formData := url.Values{
		"userTypeID": {"1"},
		"building":   {building},
		"room":       {room},
	}

	resp, err := http.PostForm(apiURL, formData)
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

	return result.Data, nil
}
