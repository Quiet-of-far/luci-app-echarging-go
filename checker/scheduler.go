package checker

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	"luci-app-echarging-go/api"
	"luci-app-echarging-go/config"
	"luci-app-echarging-go/models"
	"luci-app-echarging-go/notifier"
	"luci-app-echarging-go/prediction"
	"luci-app-echarging-go/storage"
)

const (
	alertLowEnergy = "low_energy"
	alertDepletion = "depletion"
)

type Checker struct {
	cfg       *config.Config
	store     *storage.Store
	notifiers []notifier.Notifier
	lastAlert map[string]time.Time
}

func New(cfg *config.Config, store *storage.Store, notifiers []notifier.Notifier) *Checker {
	return &Checker{
		cfg:       cfg,
		store:     store,
		notifiers: notifiers,
		lastAlert: make(map[string]time.Time),
	}
}

func (c *Checker) QueryRoom(room config.Room) (*models.QueryResult, error) {
	current, err := api.GetCurrentStatus(room.Building, room.Room)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	record := models.ElectricityRecord{
		Building:     room.Building,
		Room:         room.Room,
		RemainingKWh: current.RemainingKWh,
		QueryTime:    now,
		MeterTime:    current.MeterTime,
	}

	if err := c.store.InsertRecord(record); err != nil {
		return nil, fmt.Errorf("存储记录失败: %w", err)
	}

	status, err := c.buildStatus(room)
	if err != nil {
		return nil, err
	}

	log.Printf("[checker] %s 当前剩余电量: %.2f 度", room.Label, status.RemainingKWh)
	return &models.QueryResult{Room: *status}, nil
}

func (c *Checker) CheckRoom(room config.Room) (*models.QueryResult, error) {
	result, err := c.QueryRoom(room)
	if err != nil {
		log.Printf("[checker] 查询 %s 失败: %v", room.Label, err)
		return nil, err
	}

	alerts := c.evaluateAlerts(result.Room)
	if len(alerts) > 0 && c.shouldSendAlert(room, alerts, result.Room.QueryTime) {
		summary, body := c.buildAlertMessage(result.Room, alerts)
		notifier.SendAll(c.notifiers, summary, body)
		c.markAlertsSent(room, alerts, result.Room.QueryTime)
		log.Printf("[checker] 已发送告警: %s (%s)", room.Label, strings.Join(alerts, ","))
	}

	result.Alerts = alerts
	return result, nil
}

func (c *Checker) QueryAll() []models.QueryResult {
	results := make([]models.QueryResult, 0, len(c.cfg.Rooms))
	for _, room := range c.cfg.Rooms {
		result, err := c.CheckRoom(room)
		if err != nil {
			results = append(results, models.QueryResult{
				Room: models.RoomStatus{
					Label:            room.Label,
					Building:         room.Building,
					Room:             room.Room,
					PredictionStatus: "",
					QueryStatus:      err.Error(),
				},
			})
			continue
		}
		results = append(results, *result)
	}
	return results
}

func (c *Checker) GetStatuses() []models.RoomStatus {
	statuses := make([]models.RoomStatus, 0, len(c.cfg.Rooms))
	for _, room := range c.cfg.Rooms {
		status, err := c.buildStatus(room)
		if err != nil {
			if err == sql.ErrNoRows {
				statuses = append(statuses, models.RoomStatus{
					Label:              room.Label,
					Building:           room.Building,
					Room:               room.Room,
					QueryStatus:        "暂无查询记录",
					QueryTime:          time.Time{},
					MeterTime:          nil,
					PredictedEmptyTime: nil,
				})
				continue
			}

			statuses = append(statuses, models.RoomStatus{
				Label:            room.Label,
				Building:         room.Building,
				Room:             room.Room,
				QueryStatus:      err.Error(),
				QueryTime:        time.Time{},
				MeterTime:        nil,
				PredictionStatus: "",
			})
			continue
		}
		statuses = append(statuses, *status)
	}
	return statuses
}

func (c *Checker) TestNotifier(channel string) error {
	summary := "[电量监控] 测试通知"
	body := "这是一条测试通知，用于确认通知渠道配置已生效。"

	switch channel {
	case "email":
		if !c.cfg.Email.Enabled {
			return fmt.Errorf("邮件通知未启用")
		}
		n := notifier.NewEmailNotifier(c.cfg.Email)
		if err := n.Send(summary, body); err != nil {
			return err
		}
	case "wxpusher":
		if !c.cfg.WxPusher.Enabled {
			return fmt.Errorf("微信通知未启用")
		}
		n := notifier.NewWxPusherNotifier(c.cfg.WxPusher)
		if err := n.Send(summary, body); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported channel: %s", channel)
	}

	return nil
}

func (c *Checker) buildStatus(room config.Room) (*models.RoomStatus, error) {
	latest, err := c.store.GetLatestRecord(room.Building, room.Room)
	if err != nil {
		return nil, err
	}

	status := &models.RoomStatus{
		Label:            room.Label,
		Building:         room.Building,
		Room:             room.Room,
		RemainingKWh:     latest.RemainingKWh,
		MeterTime:        latest.MeterTime,
		QueryTime:        latest.QueryTime,
		PredictionStatus: "预测不可用",
	}

	records, err := c.store.GetRecentRecords(room.Building, room.Room, c.cfg.Prediction.SampleCount)
	if err != nil {
		return nil, fmt.Errorf("读取历史记录失败: %w", err)
	}

	pred, err := prediction.Calculate(records, c.cfg.Prediction.CustomDailyConsumption)
	if err != nil {
		status.PredictionStatus = fmt.Sprintf("预测不可用: %v", err)
		return status, nil
	}

	status.PredictionStatus = "ok"
	status.PredictedEmptyTime = &pred.EstimatedEmptyTime
	status.DailyConsumptionKWh = &pred.DailyConsumptionKWh
	return status, nil
}

func (c *Checker) evaluateAlerts(status models.RoomStatus) []string {
	var alerts []string

	if status.QueryStatus != "" {
		return alerts
	}

	if status.RemainingKWh <= c.cfg.LowEnergyThreshold {
		alerts = append(alerts, alertLowEnergy)
	}

	if status.PredictedEmptyTime != nil && c.cfg.DepletionAlertDays > 0 {
		remaining := status.PredictedEmptyTime.Sub(status.QueryTime)
		if remaining <= time.Duration(c.cfg.DepletionAlertDays)*24*time.Hour {
			alerts = append(alerts, alertDepletion)
		}
	}

	return alerts
}

func (c *Checker) shouldSendAlert(room config.Room, alerts []string, now time.Time) bool {
	for _, alert := range alerts {
		key := alertKey(room, alert)
		if last, ok := c.lastAlert[key]; ok && now.Sub(last) < 24*time.Hour {
			continue
		}
		return true
	}
	return false
}

func (c *Checker) markAlertsSent(room config.Room, alerts []string, now time.Time) {
	for _, alert := range alerts {
		key := alertKey(room, alert)
		c.lastAlert[key] = now
	}
}

func alertKey(room config.Room, alert string) string {
	return room.Building + ":" + room.Room + ":" + alert
}

func (c *Checker) buildAlertMessage(status models.RoomStatus, alerts []string) (string, string) {
	summary := fmt.Sprintf("[电量预警] %s", status.Label)
	reasons := make([]string, 0, len(alerts))
	for _, alert := range alerts {
		switch alert {
		case alertLowEnergy:
			reasons = append(reasons, fmt.Sprintf("剩余电量低于阈值 %.2f 度", c.cfg.LowEnergyThreshold))
		case alertDepletion:
			reasons = append(reasons, fmt.Sprintf("预计将在 %d 天内耗尽", c.cfg.DepletionAlertDays))
		}
	}

	lines := []string{
		fmt.Sprintf("宿舍: %s", status.Label),
		fmt.Sprintf("剩余电量: %.2f 度", status.RemainingKWh),
		fmt.Sprintf("告警原因: %s", strings.Join(reasons, "；")),
	}
	if status.MeterTime != nil {
		lines = append(lines, fmt.Sprintf("抄表时间: %s", status.MeterTime.Format("2006-01-02 15:04:05")))
	}
	if status.PredictedEmptyTime != nil {
		lines = append(lines, fmt.Sprintf("预计耗尽时间: %s", status.PredictedEmptyTime.Format("2006-01-02")))
	}
	if status.DailyConsumptionKWh != nil {
		lines = append(lines, fmt.Sprintf("日均消耗: %.2f 度/天", *status.DailyConsumptionKWh))
	}
	if status.PredictionStatus != "ok" && status.PredictionStatus != "" {
		lines = append(lines, status.PredictionStatus)
	}

	return summary, strings.Join(lines, "\n")
}

func (c *Checker) Run(stop <-chan struct{}) {
	c.QueryAll()

	if len(c.cfg.Schedule.CheckHours) > 0 {
		c.runCheckHoursMode(stop)
	} else {
		c.runIntervalMode(stop)
	}
}

func (c *Checker) runIntervalMode(stop <-chan struct{}) {
	ticker := time.NewTicker(time.Duration(c.cfg.Schedule.IntervalMinutes) * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.QueryAll()
		case <-stop:
			return
		}
	}
}

func (c *Checker) runCheckHoursMode(stop <-chan struct{}) {
	hourSet := make(map[int]bool)
	for _, h := range c.cfg.Schedule.CheckHours {
		hourSet[h] = true
	}

	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	lastRunHour := -1
	lastRunDay := -1

	for {
		select {
		case t := <-ticker.C:
			hour := t.Hour()
			day := t.YearDay()
			if hourSet[hour] && !(day == lastRunDay && hour == lastRunHour) {
				c.QueryAll()
				lastRunHour = hour
				lastRunDay = day
			}
		case <-stop:
			return
		}
	}
}
