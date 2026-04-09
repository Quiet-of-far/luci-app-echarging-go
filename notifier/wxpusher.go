package notifier

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"luci-app-echarging-go/config"
)

const wxPusherAPI = "https://wxpusher.zjiecode.com/api/send/message"

type wxPusherRequest struct {
	AppToken    string   `json:"appToken"`
	Content     string   `json:"content"`
	Summary     string   `json:"summary,omitempty"`
	ContentType int      `json:"contentType"`
	TopicIDs    []int    `json:"topicIds,omitempty"`
	UIDs        []string `json:"uids,omitempty"`
}

type wxPusherResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

type WxPusherNotifier struct {
	cfg config.WxPusherConfig
}

func NewWxPusherNotifier(cfg config.WxPusherConfig) *WxPusherNotifier {
	return &WxPusherNotifier{cfg: cfg}
}

func (n *WxPusherNotifier) Send(summary, body string) error {
	if !n.cfg.Enabled {
		return nil
	}

	reqBody := wxPusherRequest{
		AppToken:    n.cfg.AppToken,
		Content:     body,
		Summary:     summary,
		ContentType: 1,
		UIDs:        n.cfg.UIDs,
		TopicIDs:    n.cfg.TopicIDs,
	}

	data, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}

	resp, err := http.Post(wxPusherAPI, "application/json", bytes.NewReader(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var result wxPusherResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return err
	}

	if result.Code != 1000 {
		return fmt.Errorf("wxpusher error (code %d): %s", result.Code, result.Msg)
	}

	return nil
}
