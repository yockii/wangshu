# YoClaw

YoClaw 是一个功能强大的智能对话机器人框架，支持多种通信渠道接入，提供丰富的工具和灵活的任务执行能力。

## 快速开始

### 安装

从 [Releases](https://github.com/yockii/YoClaw/releases) 页面下载对应平台的可执行文件。

### 使用方法

详细的使用指南请参考 [Wiki - 使用指南](https://github.com/yockii/YoClaw/wiki/%E4%BD%BF%E7%94%A8%E6%8C%87%E5%8D%97)。

## 功能特性

### 多平台接入
- 支持飞书等多种通信渠道
- 统一的消息总线架构
- 灵活的渠道管理机制

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

### 丰富的工具生态

#### 文件系统工具
- `read_file` - 读取文件内容
- `write_file` - 写入文件内容
- `edit_file` - 编辑文件
- `list_dir` - 列出目录内容
- `find_file` - 查找文件
- `grep` - 搜索文件内容
- `rename_file` - 重命名文件

#### Shell 工具
- `exec` - 执行 Shell 命令
- `process` - 进程管理
- `auto_interactive` - 自动交互式命令执行

#### 网络工具
- `web_search` - 网络搜索
- `web_fetch` - 获取网页内容
- `browser` - 浏览器操作

#### 系统工具
- `cron` - 定时任务管理
- `message` - 发送消息
- `task` - 异步任务管理

#### 内存工具
- `memory` - 记忆管理

#### 内置工具
- `sleep` - 延迟执行
- `get_time` - 获取当前时间

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

## 许可证

MIT License
