package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"luci-app-echarging-go/checker"
	"luci-app-echarging-go/config"
	"luci-app-echarging-go/notifier"
	"luci-app-echarging-go/storage"
)

func main() {
	configPath := flag.String("config", "/var/etc/echarging.json", "配置文件路径")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	store, err := storage.New(cfg.DBPath)
	if err != nil {
		log.Fatalf("打开数据库失败: %v", err)
	}
	defer store.Close()

	var notifiers []notifier.Notifier
	if cfg.Email.Enabled {
		notifiers = append(notifiers, notifier.NewEmailNotifier(cfg.Email))
	}
	if cfg.WxPusher.Enabled {
		notifiers = append(notifiers, notifier.NewWxPusherNotifier(cfg.WxPusher))
	}

	c := checker.New(cfg, store, notifiers)

	stop := make(chan struct{})
	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
		<-sig
		log.Println("收到退出信号，正在关闭...")
		close(stop)
	}()

	log.Println("电费监控已启动")
	c.Run(stop)
	log.Println("已退出")
}
