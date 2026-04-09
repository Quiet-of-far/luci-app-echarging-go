package checker

import (
	"fmt"
	"log"
	"strconv"
	"time"

	"luci-app-echarging-go/api"
	"luci-app-echarging-go/config"
	"luci-app-echarging-go/models"
	"luci-app-echarging-go/notifier"
	"luci-app-echarging-go/prediction"
	"luci-app-echarging-go/storage"
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

func (c *Checker) CheckRoom(room config.Room) {
	balanceStr, err := api.GetBalance(room.Building, room.Room)
	if err != nil {
		log.Printf("[checker] 查询 %s 失败: %v", room.Label, err)
		return
	}

	balance, err := strconv.ParseFloat(balanceStr, 64)
	if err != nil {
		log.Printf("[checker] 解析余额失败 (%s=%q): %v", room.Label, balanceStr, err)
		return
	}

	now := time.Now()
	record := models.BalanceRecord{
		Building:  room.Building,
		Room:      room.Room,
		Balance:   balance,
		QueryTime: now,
	}
	if err := c.store.InsertRecord(record); err != nil {
		log.Printf("[checker] 存储记录失败 (%s): %v", room.Label, err)
	}

	// Run prediction
	var predStr string
	records, err := c.store.GetRecentRecords(room.Building, room.Room, c.cfg.Prediction.SampleCount)
	if err != nil {
		log.Printf("[checker] 读取历史记录失败 (%s): %v", room.Label, err)
	} else {
		pred, err := prediction.Calculate(records, c.cfg.Prediction.CustomDailyRate)
		if err != nil {
			predStr = fmt.Sprintf("预测不可用: %v", err)
		} else {
			predStr = fmt.Sprintf("日均消耗: %.2f 元/天\n预计剩余: %.0f天%.0f小时\n预计耗尽: %s",
				pred.DailyRate,
				pred.RemainingDays,
				float64(int(pred.RemainingHours)%24),
				pred.EstimatedEmpty.Format("2006-01-02 15:04"))
		}
	}

	log.Printf("[checker] %s 当前余额: %.2f 元", room.Label, balance)

	// Alert if below threshold
	if balance < c.cfg.Threshold {
		key := room.Building + ":" + room.Room
		if last, ok := c.lastAlert[key]; ok && time.Since(last) < 24*time.Hour {
			return
		}

		summary := fmt.Sprintf("[电费预警] %s", room.Label)
		body := fmt.Sprintf("当前余额: %.2f 元\n%s\n请及时充值!", balance, predStr)

		notifier.SendAll(c.notifiers, summary, body)
		c.lastAlert[key] = now
		log.Printf("[checker] 已发送低余额告警: %s", room.Label)
	}
}

func (c *Checker) CheckAll() {
	for _, room := range c.cfg.Rooms {
		c.CheckRoom(room)
	}
}

func (c *Checker) Run(stop <-chan struct{}) {
	// Run immediately on start
	c.CheckAll()

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
			c.CheckAll()
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

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	lastRunHour := -1
	lastRunDay := -1

	for {
		select {
		case t := <-ticker.C:
			hour := t.Hour()
			day := t.YearDay()
			if hourSet[hour] && !(day == lastRunDay && hour == lastRunHour) {
				c.CheckAll()
				lastRunHour = hour
				lastRunDay = day
			}
		case <-stop:
			return
		}
	}
}
