package main

import (
	"encoding/json"
	"flag"
	"fmt"
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

	c := checker.New(cfg, store, buildNotifiers(cfg))
	command := "run"
	var args []string
	if flag.NArg() > 0 {
		command = flag.Arg(0)
		args = flag.Args()[1:]
	}

	switch command {
	case "run":
		runDaemon(c)
	case "query-now":
		writeJSON(c.QueryAll())
	case "status":
		writeJSON(c.GetStatuses())
	case "test-notify":
		channel, err := parseNotifyChannel(args)
		if err != nil {
			log.Fatal(err)
		}
		if err := c.TestNotifier(channel); err != nil {
			log.Fatalf("测试通知失败: %v", err)
		}
		writeJSON(map[string]string{
			"status":  "ok",
			"channel": channel,
			"message": "测试通知已发送",
		})
	default:
		log.Fatalf("未知命令: %s", command)
	}
}

func parseNotifyChannel(args []string) (string, error) {
	fs := flag.NewFlagSet("test-notify", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	channel := fs.String("channel", "", "通知渠道: email|wxpusher")
	if err := fs.Parse(args); err != nil {
		return "", err
	}
	if *channel == "" {
		return "", fmt.Errorf("缺少 --channel 参数")
	}
	return *channel, nil
}

func buildNotifiers(cfg *config.Config) []notifier.Notifier {
	var notifiers []notifier.Notifier
	if cfg.Email.Enabled {
		notifiers = append(notifiers, notifier.NewEmailNotifier(cfg.Email))
	}
	if cfg.WxPusher.Enabled {
		notifiers = append(notifiers, notifier.NewWxPusherNotifier(cfg.WxPusher))
	}
	return notifiers
}

func runDaemon(c *checker.Checker) {
	stop := make(chan struct{})
	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
		<-sig
		log.Println("收到退出信号，正在关闭...")
		close(stop)
	}()

	log.Println("电量监控已启动")
	c.Run(stop)
	log.Println("已退出")
}

func writeJSON(v any) {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(v); err != nil {
		log.Fatal(fmt.Errorf("写出 JSON 失败: %w", err))
	}
}
