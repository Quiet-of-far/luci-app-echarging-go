package main

import (
	"fmt"
	"luci-app-echarging-go/api"
)

func main() {
	// 使用 api. 前缀来调用另一个包里的函数
	balance, err := api.GetBalance("44", "1207")
	if err != nil {
		fmt.Printf("警告：查询失败 -> %v\n", err)
		return
	}

	fmt.Printf("指挥中心收到：当前余额 %s 元\n", balance)
}
