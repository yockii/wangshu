# YoClaw

YoClaw 是一个功能强大的智能对话机器人框架，支持多种通信渠道接入，提供丰富的工具和灵活的任务执行能力。

## 快速开始

### 安装

#### 方式一：下载可执行文件

从 [Releases](https://github.com/yockii/YoClaw/releases) 页面下载对应平台的可执行文件。

#### 方式二：使用 Docker

使用 Docker 可以快速部署 YoClaw，所有数据都存储在 `~/.yoClaw` 目录中。

```bash
# 拉取最新镜像
docker pull ghcr.io/yockii/yoclaw:latest

# 创建配置目录
mkdir -p ~/.yoClaw

# 创建配置文件（首次运行也可以自动创建，但需要修改后重启）
cat > ~/.yoClaw/config.json << EOF
{
  "agents": {
    "default": {
      "workspace": "/root/.yoClaw/workspace",
      "provider": "myProvider",
      "model": "gpt-4",
      "temperature": 0.7
    }
  },
  "providers": {
    "myProvider": {
      "type": "openai",
      "api_key": "your-api-key",
      "base_url": "https://api.openai.com/v1"
    }
  },
  "channels": {
    "feishu": {
      "enabled": true,
      "agent": "default",
      "app_id": "your-app-id",
      "app_secret": "your-app-secret"
    }
  },
  "skill": {
    "global_path": "/root/.yoClaw/skills",
    "builtin_path": "./skills"
  }
}
EOF

# 运行容器
docker run -d \
  --name yoclaw \
  -v ~/.yoClaw:/root/.yoClaw \
  ghcr.io/yockii/yoclaw:latest
```

**说明：**
- `-v ~/.yoClaw:/root/.yoClaw` - 将本地目录挂载到容器，所有数据（配置、工作空间、技能、会话等）都会持久化
- 容器内的配置文件路径为 `/root/.yoClaw/config.json`
- 查看日志：`docker logs -f yoclaw`
- 停止容器：`docker stop yoclaw`
- 重启容器：`docker restart yoclaw`

**使用特定版本：**

```bash
docker pull ghcr.io/yockii/yoclaw:v0.1.0
docker run -d --name yoclaw -v ~/.yoClaw:/root/.yoClaw ghcr.io/yockii/yoclaw:v0.1.0
```

### 配置指南

#### 1. 飞书渠道配置

1. 访问 [飞书开放平台](https://open.feishu.cn/) 并登录
2. 创建企业自建应用：
   - 进入「管理后台」→「应用管理」→「创建企业自建应用」
   - 填写应用名称和描述
3. 获取应用凭证：
   - 在应用详情页找到「凭证与基础信息」
   - 记录 `App ID` 和 `App Secret`
4. 配置机器人权限：
   - 进入「权限管理」
   - 开启以下权限：
     - `im:message`（发送消息）
     - `im:message:group_at_msg`（获取群组@消息）
     - `im:message:send_as_bot`（以机器人身份发送消息）
5. 发布应用
6. 配置事件订阅（需要将前序配置好并启动程序）：
   - 进入「事件订阅」页面
   - 采用长连接方式（需要启动程序连接上后才能配置）
   - 添加事件：`im.message.receive_v1`（接收消息）

#### 2. LLM Provider 配置

YoClaw 支持 OpenAI 兼容接口的 LLM Provider。在配置文件中配置：

```json
{
  "providers": {
    "myProvider": {
      "type": "openai",
      "api_key": "your-api-key",
      "base_url": "https://api.openai.com/v1"
    }
  },
  "agents": {
    "default": {
      "provider": "myProvider",
      "model": "gpt-4"
    }
  }
}
```

支持其他兼容 OpenAI 接口的提供商（如通义千问、文心一言等），只需修改 `base_url` 即可。

### 自定义技能

YoClaw 支持通过手动添加技能来扩展功能。技能定义文件为 `SKILL.md`，包含以下内容：

```markdown
# 技能名称

## 描述
技能的详细描述

## 工具
技能提供的工具列表

## 使用方法
技能的使用说明
```

将技能文件放置在以下目录之一：
- 全局技能目录：`~/.yoClaw/skills/`
- 内置技能目录：`./skills/`

系统会自动发现并加载这些技能。

## 功能特性

### 多平台接入
- 当前支持飞书渠道
- 统一的消息总线架构
- 灵活的渠道管理机制
- 易于扩展其他通信渠道

### 智能 Agent 系统
- 基于大语言模型的智能对话
- 支持多 Agent 并发运行
- 可配置的工作空间和会话管理
- 个性化配置支持（SOUL.md、IDENTITY.md 等）

### 定时任务系统
- 基于 Cron 表达式的定时任务调度
- 任务持久化存储，支持重启恢复
- 灵活的任务管理（创建、暂停、恢复、禁用、查询）
- 定时任务可转换为异步任务执行

### 异步任务系统
- 后台异步任务执行，不阻塞用户对话
- 任务优先级支持（urgent、high、normal、low）
- 分步执行机制，每次处理一步
- 任务历史记录，支持断点续传
- 任务完成自动通知

### 可扩展的 Skill 系统
- 自动发现和加载技能
- 基于文件的技能定义（SKILL.md）
- 灵活的技能安装机制

### 会话管理
- 会话持久化存储
- 会话过期清理
- 完整的消息历史记录

## 架构设计

YoClaw 采用模块化设计，核心组件包括：

- **Agent**: 智能助手核心，负责对话处理和任务执行
- **Channel**: 通信渠道适配器，处理不同平台的接入
- **LLM Provider**: 大语言模型提供者接口
- **Tools**: 工具系统，提供丰富的功能扩展
- **Skills**: 技能系统，支持可插拔的能力扩展
- **Cron Manager**: 定时任务管理器
- **Task Executor**: 异步任务执行器
- **Session Manager**: 会话管理器
- **Bus**: 消息总线，处理消息分发

## 后续计划

### 渠道扩展
<!-- - [ ] 微信渠道 -->
- [ ] 钉钉渠道
- [ ] Telegram 渠道
- [ ] Slack 渠道
<!-- - [ ] 企业微信渠道 -->

### LLM Provider 扩展
- [ ] Anthropic Claude 原生支持
- [ ] Google Gemini 支持
- [ ] Ollama 本地模型支持
- [ ] 更多国产大模型适配

### 功能增强
<!-- - [ ] 图像处理工具 -->
<!-- - [ ] 代码分析工具 -->
<!-- - [ ] 数据库操作工具 -->
- [ ] 更多内置技能
- [ ] 技能市场（在线安装）
- [ ] Web 管理界面

## 许可证

MIT License
