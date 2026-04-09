## 功能
- 填写宿舍(栋+宿舍号)作为被查询对象(可以是多个,但一次只能查询一个,需要分别查询)
- 定时查询(自定义时间,每日几点)
- 剩余使用时长预测
- 自定义低阈值提醒(接入邮箱通知)
- 自定义低阈值提醒(接入Wxpusher通知)
- 为本项目制作OpenWrt LuCI控制台并合并打包为单个ipk

## 核心功能思路与工具包
### 发送HTTP请求(get_balance_from_api)
使用包:net/http用于发送请求;encoding/json用于解析返回的JSON数据;net/url处理POST表单数据  
关键函数:
- http.PostForm():发送POST请求
- json.Unmarshal():将JSON字符串转为Go结构体(Struct)
思路:定义一个struct来匹配API返回的JSON格式
### 定时任务(hourly_check_worker)
使用包:time  
关键函数:
- time.NewTicker():创建一个定时器,每隔一段时间向"通道(Channel)"发送信号
- go关键字:直接在函数前加go即可开启并发
思路:在main函数中使用go checkLoop()启动一个后台循环
### 电费预测
- 根据近x次(可自定义x)电费抄表时间和电费差计算大概的电量消耗速度,再根据最后一次电费抄表时间和剩余量计算当前剩余多少使用时间
- 自定义电量消耗速度(度/天)
思路:SQLite储存
### 通知功能(邮件/微信)
- 邮件:使用标准库net/smtp,或者更友好的第三方包github.com/jordan-wright/email
- 微信:接入WxPusher(具体实现见文档https://wxpusher.zjiecode.com/docs/#/)

## 当前项目结构：                                                             
工作区/                                                                   
├── main.go             # 程序入口,初始化并启动服务                       
├── api/                # 专门负责与学校电费接口通信                      
│   └── client.go                                                         
├── checker/            # 核心业务逻辑:定时任务、阈值判断                 
│   └── scheduler.go                                                      
├── notifier/           # 通知模块:邮件、微信等                           
│   └── email.go                                                          
├── config/             # 配置管理:读取宿舍列表、API地址、阈值            
│   └── config.go                                                         
├── models/             # 定义数据结构(如:电费响应的结构体)               
│   └── electricity.go                                                    
└── go.mod              # 依赖管理文件(go mod init luci-app-6echarging-go)
