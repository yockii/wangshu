package config

import (
	"fmt"
	"os"

	"github.com/yockii/wangshu/pkg/constant"
	"github.com/yockii/wangshu/pkg/utils"
)

func leadUserToFillConfig(cfg *Config) {
	dockerJudgeFile := "/.dockerenv"
	if _, err := os.Stat(dockerJudgeFile); err == nil {
		fmt.Println("检测到在Docker容器中运行，将跳过配置文件填写步骤")
		return
	}
	doLeadUserToFillConfig(cfg)
}

func doLeadUserToFillConfig(cfg *Config) {
	fmt.Print("\033[1;31;42m")
	fmt.Println("配置文件不存在，将引导您填写配置信息。")
	fmt.Println("请在控制台上输入配置信息，按 Enter 键确认。")
	fmt.Print("\033[0m")
	// provider
	fmt.Print("请输入要使用的大模型的api key（Open AI兼容）：")
	var apiKey string
	fmt.Scanln(&apiKey)
	cfg.Providers["myProvider"].APIKey = apiKey
	// base url
	fmt.Print("请输入要使用的大模型的base url（Open AI兼容， 默认：https://api.openai.com/v1）：")
	var baseURL string
	fmt.Scanln(&baseURL)
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	cfg.Providers["myProvider"].BaseURL = baseURL
	// model
	fmt.Print("请输入要使用的大模型的model（Open AI兼容， 默认：gpt-5）：")
	var model string
	fmt.Scanln(&model)
	if model == "" {
		model = "gpt-5"
	}
	cfg.Agents[constant.Default].Model = model

	// 路径处理
	p := utils.ExpandPath(cfg.Skill.GlobalPath)
	fmt.Printf("请输入要存放skills的路径（默认：%s）：", p)
	var skillPath string
	fmt.Scanln(&skillPath)
	if skillPath != "" {
		cfg.Skill.GlobalPath = skillPath
	}
	// agent路径
	agentWorkspace := utils.ExpandPath(cfg.Agents[constant.Default].Workspace)
	fmt.Printf("请输入你想让agent工作的目录（默认：%s）", agentWorkspace)
	var workspace string
	fmt.Scanln(&workspace)
	if workspace != "" {
		cfg.Agents[constant.Default].Workspace = workspace
	}

	// channel
	fmt.Println("请进入飞书工作台，新建或选择已有的企业自建应用(https://open.feishu.cn/app)，随后复制应用凭证")
	fmt.Print("请输入飞书的App ID：")
	var appID string
	fmt.Scanln(&appID)
	cfg.Channels["feishuTest"].AppID = appID
	fmt.Print("请输入飞书的App Secret：")
	var appSecret string
	fmt.Scanln(&appSecret)
	cfg.Channels["feishuTest"].AppSecret = appSecret
	cfg.Channels["feishuTest"].Enabled = true
	// 提示飞书配置
	fmt.Println("请进入飞书应用管理的添加应用能力->按能力添加：机器人，这一步将使得应用可以与你在飞书app中聊天")
	fmt.Print("完成后请按回车：")
	var enter string
	fmt.Scanln(&enter)
	// 继续配置
	fmt.Println("请进入飞书应用管理中的权限管理，开通以下权限")
	fmt.Println("contact:contact.base:readonly")
	fmt.Println("contact:user.base:readonly")
	fmt.Println("im:chat:read")
	fmt.Println("im:message")
	fmt.Println("im:message.group_msg")
	fmt.Println("im:message.p2p_msg:readonly")
	fmt.Println("im:message:readonly")
	fmt.Println("im:message:send_as_bot")
	fmt.Println("im:resource")
	fmt.Print("完成后请按回车：")
	fmt.Scanln(&enter)
	fmt.Println("现在你可以发布飞书应用了，发布完成后，再次回车，本向导将结束配置，但后续还需要做一件事情：")

	fmt.Print("\033[1;31;40m")

	fmt.Println("在启动完成后，你需要再次去飞书应用后台的事件与回调中，点击事件配置的小铅笔（订阅方式），使用长连接接受事件，并保存，这里必须在本应用连接上才能成功保存")
	fmt.Println("保存后，还需要在下方添加事件：im.message.receive_v1 【接收消息v2.0】")
	fmt.Println("如果需要，还可以在回调配置中使用长连接订阅方式接收回调")
	fmt.Print("请确保已发布应用并阅读知晓后续步骤后回车，本系统将结束向导并开始运行")

	fmt.Print("\033[0m")

	fmt.Scanln(&enter)
}
