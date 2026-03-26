![GitHub Actions Workflow Status](https://img.shields.io/github/actions/workflow/status/yockii/wangshu/build.yml) ![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/yockii/wangshu) ![GitHub last commit](https://img.shields.io/github/last-commit/yockii/wangshu) [![Build Release](https://github.com/yockii/wangshu/actions/workflows/build.yml/badge.svg)](https://github.com/yockii/wangshu/actions/workflows/build.yml) ![GitHub Release](https://img.shields.io/github/v/release/yockii/wangshu) ![GitHub Release Date](https://img.shields.io/github/release-date/yockii/wangshu) ![GitHub Downloads (all assets, all releases)](https://img.shields.io/github/downloads/yockii/wangshu/total)


# 望舒 (Wangshu)

> *"前望舒使先驱兮，后飞廉使奔属。"* —— 屈原《离骚》

**望舒引路，智慧同行。**

望舒，中国神话中为月亮驾车的神，寓意照亮前路、引导方向。本项目以此为名，致力于成为个人/企业的 AI 助手，驾驭电脑和系统，陪伴工作之旅，照亮工作之路，引导团队高效协作，做稳定可靠的数字员工。

> 望舒率先采用浏览器自动化能力，实现了飞书渠道的全自动配置，无需参考教程来一步步配置飞书的各种权限、回调、发布等，完全由望舒自动完成（当然，你需要扫码登录一下）

> 支持x86(amd64)、arm64、Loong64架构，windows、macos、linux系统、信创环境

<details>
<summary>你可以查看视频：</summary>
Youtube视频

[![望舒-自动创建飞书应用](https://img.youtube.com/vi/IL5lKSv4Jl4/0.jpg)](https://www.youtube.com/watch?v=IL5lKSv4Jl4)

请点击观看视频（bilibili）

[望舒-自动化创建飞书应用](https://www.bilibili.com/video/BV11DwrzjEVQ)
</details>

## 愿景

望舒的目标是成为你的**个人 AI 助理**和**企业数字员工**：

- 🧑‍💼 **个人 AI 助理** - 理解你的工作习惯，协助处理日常任务，成为你的智能工作伙伴
- 🏢 **企业 AI 员工** - 稳定可靠地执行业务流程，降低重复劳动成本，提升团队协作效率
- 🚀 **未来路线** - 持续演进为更智能的自主代理，具备更强的任务规划与执行能力

## 特性

- 🌙 **照亮前路** - 如望舒为月驾车，为你的工作之路提供智能指引
- 🛡️ **稳定可靠** - 零依赖部署，单一二进制文件，低资源占用
- 🤝 **智慧同行** - 陪伴工作全程，理解上下文，高效协作
- 🔧 **灵活扩展** - 支持多渠道接入，丰富的工具和技能系统

## 核心优势

### 零依赖部署

Go 语言编译为单一二进制文件，无需安装任何运行时环境。下载、运行、开始使用——就这么简单。

### 约定大于配置

我们相信好的默认值比灵活的配置更重要。望舒大部分行为都有合理的默认值，让你开箱即用，而不是在配置文件中迷失。

### 专注用户体验

特别是在群聊场景下，望舒会自动：
- 识别发送者身份并显示姓名
- 保留最近的消息上下文
- 只在被 @ 时响应，不打扰正常群聊

### 渐进式复杂度

从简单的单文件配置开始，需要时再扩展。不需要一开始就面对数百个配置项。

### 跨平台支持

原生支持 Windows、macOS、Linux，无需 WSL 或其他兼容层。

## 适用场景

- 🚀 想快速部署一个 AI 助手，不想折腾环境配置
- 🏢 企业内部使用，需要稳定、低资源占用
- 👥 群聊场景为主，需要良好的用户识别和上下文
- 🖥️ Windows 用户，不想安装 WSL2

---

## 快速开始

### 安装

#### 方式一：下载可执行文件

从 [Releases](https://github.com/yockii/wangshu/releases) 页面下载对应平台的可执行文件。

#### 方式二：使用 Docker

使用 Docker 可以快速部署 望舒，所有数据都存储在 `~/.wangshu` 目录中。

```bash
# 拉取最新镜像
docker pull ghcr.io/yockii/wangshu:latest

# 创建配置目录
mkdir -p ~/.wangshu

# 创建配置文件（首次运行也可以自动创建，但需要修改后重启）
cat > ~/.wangshu/config.json << EOF
{
  "agents": {
    "default": {
      "workspace": "/root/.wangshu/workspace",
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
    "feishuProd": {
      "type": "feishu",
      "enabled": true,
      "agent": "default",
      "app_id": "your-app-id",
      "app_secret": "your-app-secret"
    }
  },
  "skill": {
    "global_path": "/root/.wangshu/skills"
  }
}
EOF

# 运行容器
docker run -d \
  --name wangshu \
  -v ~/.wangshu:/root/.wangshu \
  ghcr.io/yockii/wangshu:latest
```

**说明：**
- `-v ~/.wangshu:/root/.wangshu` - 将本地目录挂载到容器，所有数据（配置、工作空间、技能、会话等）都会持久化
- 容器内的配置文件路径为 `/root/.wangshu/config.json`
- 查看日志：`docker logs -f wangshu`
- 停止容器：`docker stop wangshu`
- 重启容器：`docker restart wangshu`

**使用特定版本：**

```bash
docker pull ghcr.io/yockii/wangshu:v0.1.0
docker run -d --name wangshu -v ~/.wangshu:/root/.wangshu ghcr.io/yockii/wangshu:v0.1.0
```

### 配置指南

> **⚠️ 重要提示：路径配置建议**
>
> 配置文件中的所有目录路径（如 `workspace`、`global_path` 等）**强烈建议使用绝对路径**。
>
> **为什么？**
> - 相对路径是相对于**程序运行时的工作目录**，而不是程序文件所在的目录
> - 例如：在 `C:\` 目录下运行 `D:\aa\bbb\wangshu.exe`，相对路径 `./skills` 实际指向 `C:\skills`，而不是 `D:\aa\bbb\skills`
> - 使用绝对路径可以避免因运行目录不同导致的路径错误
>
> **推荐做法：**
> ```json
> {
>   "agents": {
>     "default": {
>       "workspace": "C:/Users/YourName/.wangshu/workspace"
>     }
>   },
>   "skill": {
>     "global_path": "C:/Users/YourName/.wangshu/skills",
>     "builtin_path": "D:/path/to/wangshu/skills"
>   }
> }
> ```
>
> **支持波浪号扩展：**
> 配置文件支持 `~` 符号，会自动扩展为用户主目录：
> ```json
> {
>   "agents": {
>     "default": {
>       "workspace": "~/.wangshu/workspace"
>     }
>   }
> }
> ```
> 上述配置在 Windows 上会自动展开为 `C:\Users\YourName\.wangshu\workspace`

#### 1. Web界面配置（推荐）

望舒 提供了独立的 Web 管理程序 [Wangshu-Manager](https://github.com/yockii/wangshu-manager)，支持通过浏览器进行聊天和管理。

**安装和启动 Web 管理程序：**

**方式一：下载二进制包（推荐）**

```bash
# 1. 从 Wangshu-Manager Releases 页面下载对应平台的二进制包
# 访问：https://github.com/yockii/wangshu-manager/releases
# 下载的 zip 包包含：
#   - wangshu-manager（可执行文件）
#   - static/（Web 界面静态文件目录）

# 2. 解压 zip 包
# Windows
# 解压后得到：wangshu-manager.exe 和 static/ 目录

# Linux/macOS
unzip wangshu-manager-linux-amd64.zip 
# 解压后得到：wangshu-manager 和 static/ 目录

# 3. 运行
# Windows
wangshu-manager.exe

# Linux/macOS
chmod +x wangshu-manager
./wangshu-manager
```

**方式二：从源码编译**

```bash
# 克隆仓库
git clone https://github.com/yockii/wangshu-manager.git
cd wangshu-manager

# 编译
go build -o wangshu-manager -ldflags "-w -s" ./cmd/

# 运行（默认监听8080端口）
./wangshu-manager
```

**运行配置：**

```bash
# 默认配置（监听8080端口，使用默认token）
./wangshu-manager

# 自定义配置
./wangshu-manager -addr :9000 -token my-secret-token -wangshu-path ~/.wangshu
```

**重要提示：**

1. **目录结构**：确保 `wangshu-manager` 可执行文件和 `static/` 目录在同一级目录下，否则 Web 界面无法正常加载。
2. **实例管理**：为了让 Web 管理程序能够识别和管理 Wangshu 实例，建议将 `wangshu` 和 `wangshu-manager` 放置在**同一目录**下。这样管理器可以自动发现 Wangshu 可执行文件，并提供启动/停止/重启等管理功能。

**配置 Wangshu 使用 Web Channel：**

在 `~/.wangshu/config.json` 中添加 Web Channel 配置：

```json
{
  "channels": {
    "webTest": {
      "type": "web",
      "enabled": true,
      "agent": "default",
      "host_address": "localhost:8080",
      "token": "your-secret-token"
    }
  }
}
```

**访问 Web 界面：**

打开浏览器访问 `http://localhost:8080?token=your-secret-token`

**Web 管理程序功能：**
- 💬 实时聊天界面
- 🖥️ 望舒 实例管理（启动/停止/重启）
- 📋 会话管理
- 📝 任务管理
- ⏰ 定时任务管理
- ⚙️ 配置管理
- 🔌 完整的 REST API

**API 开发：**

Web 管理程序提供完整的 REST API 和 WebSocket 接口，支持第三方开发自己的界面。详细的 API 文档请参考 [wangshu-manager 仓库](https://github.com/yockii/wangshu-manager)。

#### 2. 飞书渠道配置

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
     - `contact:contact.base:readonly`（获取通讯录基本信息）
     - `contact:user.base:readonly`（获取用户基本信息）
     - `im:chat:read`（查看群消息）
     - `im:message`（发送消息）
     - `im:message.group_msg`（获取群聊中所有信息，方便识别上下文）
     - `im:message.p2p_msg:readonly`（单聊消息读取）
     - `im:message:readonly`（获取单聊、群组消息）
     - `im:message:send_as_bot`（以机器人身份发送消息）
     - `im:resource`（获取与上传图片或文件资源）
   - 可选（根据需要开启）：
     - `im:chat.members:read`（读取群组成员，群聊时识别用户身份）
     - `im:message:group_at_msg`（获取群组@消息）不再需要，否则会重复接收
5. 发布应用
6. 配置事件订阅（需要将前序配置好并启动程序）：
   - 进入「事件订阅」页面
   - 采用长连接方式（需要启动程序连接上后才能配置）
   - 添加事件：`im.message.receive_v1`（接收消息）

#### 2. LLM Provider 配置

望舒 支持多种 LLM Provider：

**OpenAI 兼容接口**

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
      "model": "gpt-4",
      "temperature": 0.7
    }
  }
}
```

**Anthropic Claude（原生支持）**

```json
{
  "providers": {
    "claude-main": {
      "type": "anthropic",
      "api_key": "sk-ant-api03-xxx",
      "base_url": ""
    }
  },
  "agents": {
    "default": {
      "provider": "claude-main",
      "model": "claude-3-5-sonnet-20241022",
      "temperature": 0.7,
      "max_tokens": 16384
    }
  }
}
```

**说明：**
- OpenAI 类型：支持所有兼容 OpenAI API 的服务（通义千问、文心一言、智谱等），只需修改 `base_url`
- Anthropic 类型：使用官方 Anthropic SDK，支持 Claude 所有模型和功能以及各种coding plan(大部分以Claude Code为标准)
- `max_tokens` 配置可选，不传则使用默认值（Anthropic为4096）

### 自定义技能

望舒 支持通过手动添加技能来扩展功能。技能定义文件为 `SKILL.md`，包含以下内容：

```markdown
# 技能名称

## 描述
技能的详细描述

## 工具
技能提供的工具列表

## 使用方法
技能的使用说明
```

将技能文件放置在以下目录：
- 全局技能目录（默认）：`~/.wangshu/skills`

系统会自动发现并加载这些技能。

## 功能特性

### 多平台接入
- 当前支持飞书渠道和 Web 渠道
- 统一的消息总线架构
- 灵活的渠道管理机制
- 易于扩展其他通信渠道

### 飞书渠道增强
- **群成员持久化缓存**：启动时自动加载，解决冷启动 @用户问题
- **按渠道隔离存储**：避免不同渠道间 chat_id 冲突
- **自动维护**：获取成员后自动保存到文件
- **智能识别**：自动识别发送者身份并显示姓名

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

### 创新的 Action 机制 🆕
- **确定性执行**：相同输入 → 相同输出，消除 LLM 不确定性
- **可审计可调试**：完整的执行日志，每一步都可追溯
- **DSL 驱动**：使用 YAML 定义执行流程，支持条件判断、循环、错误处理
- **Capability 抽象**：统一的系统能力描述，确保 Action 可移植性
- 详细文档：[ACTION.md](./ACTION.md)

### 会话管理
- 会话持久化存储
- 会话过期清理
- 完整的消息历史记录

## 架构设计

望舒 采用模块化设计，核心组件包括：

- **Agent**: 智能助手核心，负责对话处理和任务执行
- **Channel**: 通信渠道适配器，处理不同平台的接入
- **LLM Provider**: 大语言模型提供者接口（OpenAI、Anthropic Claude）
- **Tools**: 工具系统，提供丰富的功能扩展
- **Skills**: 技能系统，支持可插拔的能力扩展
- **Cron Manager**: 定时任务管理器
- **Task Executor**: 异步任务执行器
- **Session Manager**: 会话管理器
- **Bus**: 消息总线，处理消息分发

## 测试覆盖

望舒 拥有完善的测试体系，确保代码质量和功能稳定性：

- **LLM Provider**: 15 个测试（Claude Provider）
- **Channel**: 135 个测试（Base、Feishu、Web、消息类型）
- **Tools**: 163 个测试（文件系统、Shell、网络、注册表）
- **核心模块**: 67 个测试（Config、Session、Task、Cron）
- **总计**: 380+ 个测试全部通过

## 文档

- [ACTION.md](./docs/ACTION.md) - **Action 机制**（创新特性：确定性执行）
- [ARCHITECTURE.md](./docs/ARCHITECTURE.md) - 架构设计文档
- [ROADMAP.md](./docs/ROADMAP.md) - 发展路线图
- [docs/](./docs/) - 更多技术文档和实现总结

## 后续计划

### 渠道扩展
<!-- - [ ] 微信渠道 -->
- [ ] 钉钉渠道
- [ ] Telegram 渠道
- [ ] Slack 渠道
<!-- - [ ] 企业微信渠道 -->

### LLM Provider 扩展
- [x] Anthropic Claude 原生支持（已完成 ✅）
- [ ] Google Gemini 支持
- [ ] Ollama 本地模型支持
- [ ] 更多国产大模型适配

### 飞书渠道增强
- [x] 群成员持久化缓存（已完成 ✅）
- [ ] 更多高级功能开发

### 功能增强
<!-- - [ ] 图像处理工具 -->
<!-- - [ ] 代码分析工具 -->
<!-- - [ ] 数据库操作工具 -->
- [ ] 更多内置技能
- [ ] 技能市场（在线安装）
- [x] Web 管理界面（已迁移至独立仓库 [wangshu-manager](https://github.com/yockii/wangshu-manager)）

## 许可证

MIT License
