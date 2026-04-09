package config

import (
	"encoding/json"
	"os"
)

type Room struct {
	Building string `json:"building"`
	Room     string `json:"room"`
	Label    string `json:"label"`
}

type ScheduleConfig struct {
	IntervalMinutes int   `json:"interval_minutes"`
	CheckHours      []int `json:"check_hours"`
}

type EmailConfig struct {
	Enabled  bool     `json:"enabled"`
	SMTPHost string   `json:"smtp_host"`
	SMTPPort int      `json:"smtp_port"`
	Username string   `json:"username"`
	Password string   `json:"password"`
	From     string   `json:"from"`
	To       []string `json:"to"`
}

type WxPusherConfig struct {
	Enabled  bool     `json:"enabled"`
	AppToken string   `json:"app_token"`
	UIDs     []string `json:"uids"`
	TopicIDs []int    `json:"topic_ids,omitempty"`
}

type PredictionConfig struct {
	SampleCount     int     `json:"sample_count"`
	CustomDailyRate float64 `json:"custom_daily_rate"`
}

type Config struct {
	Rooms      []Room           `json:"rooms"`
	Schedule   ScheduleConfig   `json:"schedule"`
	Threshold  float64          `json:"threshold"`
	Email      EmailConfig      `json:"email"`
	WxPusher   WxPusherConfig   `json:"wxpusher"`
	Prediction PredictionConfig `json:"prediction"`
	DBPath     string           `json:"db_path"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	cfg.applyDefaults()
	return &cfg, nil
}

func (c *Config) applyDefaults() {
	if c.Threshold == 0 {
		c.Threshold = 15.0
	}
	if c.DBPath == "" {
		c.DBPath = "echarging.db"
	}
	if c.Prediction.SampleCount == 0 {
		c.Prediction.SampleCount = 10
	}
	if c.Schedule.IntervalMinutes == 0 && len(c.Schedule.CheckHours) == 0 {
		c.Schedule.IntervalMinutes = 60
	}
}
