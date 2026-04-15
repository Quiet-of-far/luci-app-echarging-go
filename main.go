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

	c := checker.New(cfg, store)
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
		if len(args) > 0 {
			writeJSONError(fmt.Sprintf("不再支持通知渠道参数: %v", args))
			os.Exit(1)
		}
		if err := c.TestEmail(); err != nil {
			writeJSONError(err.Error())
			os.Exit(1)
		}
		writeJSON(map[string]string{
			"status":  "ok",
			"message": "测试邮件已发送",
		})
	default:
		writeJSONError(fmt.Sprintf("未知命令: %s", command))
		os.Exit(1)
	}
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

func writeJSONError(message string) {
	writeJSON(map[string]string{
		"status":  "error",
		"message": message,
	})
}
