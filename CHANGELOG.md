## [0.4.3] - 2026-03-18

### 🚀 Features

- *(browser)* 添加浏览器自动化任务引擎及相关文档
- 添加飞书渠道自动化创建技能
- *(浏览器任务)* 支持变量默认值语法并更新文档
- *(browser)* 添加拦截google-analytics.com请求的功能

### 🐛 Bug Fixes

- *(config)* 修正memory_organize_time的json标签拼写错误

### 💼 Other

- 添加构建时版本号参数支持

### 🚜 Refactor

- *(飞书渠道)* 自动创建飞书渠道技能整合到内嵌技能中

### 📚 Documentation

- Update CHANGELOG for v0.4.2
- *(browser)* 修正文档中的格式和拼写错误
- *(SCRIPT_GUIDE)* 补充提取和剪贴板操作的变量注册说明
- Update CHANGELOG for v0.4.3

### ⚙️ Miscellaneous Tasks

- *(workflow)* 添加同步发布到Gitee的工作流步骤
- *(workflow)* 更新Gitee发布action的分支引用
- *(workflow)* 修复构建产物文件列表输出格式
## [0.4.2] - 2026-03-13

### 🚀 Features

- *(历史消息)* 添加历史消息压缩的防抖机制和字符数阈值
- *(tui)* 实现终端用户界面及日志系统
- *(tui)* 重构运行时界面并添加监控面板
- *(tui)* 添加配置向导和保存确认功能

### 🚜 Refactor

- *(tui)* 重构配置面板为独立组件并优化交互逻辑

### 📚 Documentation

- Update CHANGELOG for v0.4.1
## [0.4.1] - 2026-03-13

### 🚀 Features

- *(browser)* 添加浏览器配置和持久化支持

### 📚 Documentation

- Update CHANGELOG for v0.4.0
## [0.4.0] - 2026-03-12

### 🚀 Features

- 添加群聊类型支持并优化定时任务处理
- *(tui)* 添加终端用户界面配置向导

### 🐛 Bug Fixes

- *(prompts)* 针对定时任务优化系统提示词

### 📚 Documentation

- Update CHANGELOG for v0.3.1
## [0.3.1] - 2026-03-12

### 🚀 Features

- 实现定时记忆整理功能，支持配置整理时间
- *(记忆整理)* 新增记忆整理功能及相关提示词
- *(配置)* 添加引导用户填写配置的功能
- *(filesystem)* 添加文件复制工具支持
- *(任务系统)* 添加任务ID参数支持并优化子任务提示

### 🐛 Bug Fixes

- *(agent)* 当消息类型为文件时，添加消息记录以告知大模型文件信息
- *(task)* 修复任务状态更新逻辑并限制描述修改方式
- *(feishu)* 优化群聊消息中提及agent时的内容格式
- *(version_tool)* 修复版本比较逻辑错误

### 📚 Documentation

- Update CHANGELOG for v0.3.0
- *(filesystem)* 更新重命名文件工具的说明，增加移动功能和覆盖说明
- Update CHANGELOG for v0.3.1
## [0.3.0] - 2026-03-11

### 🚀 Features

- 实现定时记忆整理功能，支持配置整理时间
- 实现定时记忆整理功能，支持配置整理时间

### 📚 Documentation

- Update CHANGELOG for v0.2.16
- Update CHANGELOG for v0.3.0
## [0.2.16] - 2026-03-10

### 🚀 Features

- 添加开发环境日志级别设置和优化agent初始化逻辑

### 🐛 Bug Fixes

- *(cron)* 修复jsonschema发送给大模型时的问题

### 🚜 Refactor

- 将 Version 从常量改为变量
- *(cron)* 统一使用taskType字段替代type字段
## [0.2.14] - 2026-03-10

### 🚀 Features

- *(多模态)* 支持图片识别与处理功能
- 新增运行时工具支持并优化现有功能

### 💼 Other

- 合并远程main分支

### 📚 Documentation

- Update CHANGELOG for v0.2.13

### ⚙️ Miscellaneous Tasks

- *(workflow)* Release中使用当前变更，changelog才需要完整的变更记录
## [0.2.13] - 2026-03-09

### 🚀 Features

- *(runtime)* 添加Python、NPM和Git运行时工具

### 🐛 Bug Fixes

- *(feishu)* 飞书群聊消息导致程序退出的问题

### 📚 Documentation

- Update CHANGELOG for v0.2.12
## [0.2.12] - 2026-03-09

### 🚀 Features

- *(消息处理)* 添加对<think>和<thinking>标签内容的过滤
- *(版本管理)* 添加版本工具支持应用版本查询、更新和重启功能
- *(文件系统)* 扩展文件读取工具支持PDF、DOCX和XLSX格式

### 🐛 Bug Fixes

- *(正则表达式)* 修正think标签匹配模式以包含换行符

### 🚜 Refactor

- 将版本号常量移动到pkg/constant包中

### ⚙️ Miscellaneous Tasks

- *(release)* 添加自动生成CHANGELOG和发布说明功能
- *(workflow)* 在更新CHANGELOG前切换到main分支
## [0.2.11] - 2026-03-06

### 🚀 Features

- *(飞书渠道)* 实现文件下载功能并重构消息处理逻辑

### 🐛 Bug Fixes

- *(feishu)* 修复文件发送问题

### 📚 Documentation

- 更新README文档，添加Anthropic支持和飞书渠道增强
## [0.2.10] - 2026-03-06

### 🚀 Features

- 添加Claude AI提供者支持并优化任务管理
- *(飞书)* 添加群成员缓存持久化功能
## [0.2.9] - 2026-03-05

### 🐛 Bug Fixes

- *(shell)* 优化进程监控的锁机制，防止死锁

### 🚜 Refactor

- *(llm/openai)* 实现OpenAI提供者模块并重构浏览器工具
- *(cron)* 重构定时任务管理器并添加执行器功能

### 🧪 Testing

- *(network)* 添加浏览器工具测试页面及元素收集功能测试
## [0.2.8] - 2026-03-05

### 🚀 Features

- *(config)* 重构配置验证逻辑并添加测试用例

### 🐛 Bug Fixes

- *(网络工具)* 移除duckduckgo搜索引擎选项并设置baidu为默认

### 🚜 Refactor

- 将项目名称从YoClaw更改为望舒并更新相关文档
- *(agent)* 将工具调用和消息处理逻辑拆分到独立文件
- *(agent)* 将agent循环逻辑提取到单独文件
- *(bus/channel)* 重构消息总线和渠道接口，统一消息结构
- *(channel/feishu)* 飞书渠道重构并独立管理，便于维护
- *(web)* 重构WebSocket渠道功能并添加测试

### 📚 Documentation

- 更新README中的徽章排版和内容
- 更新README文档，完善项目介绍和愿景

### 🧪 Testing

- 增加文件系统工具测试覆盖率
- *(shell)* 添加exec和process工具的单元测试
- *(网络工具)* 添加web搜索、浏览器和网页抓取工具的测试用例
- 添加task、cron和session模块的单元测试
## [0.2.7] - 2026-03-03

### 🚀 Features

- *(消息)* 增加文件附件支持并重构消息发送接口

### ⚙️ Miscellaneous Tasks

- 添加.gitattributes文件以标记特定目录为vendored代码
## [0.2.6] - 2026-03-03

### 🚀 Features

- *(feishu)* 添加机器人openID获取和群聊消息处理功能
- *(飞书)* 增强群聊消息处理能力
- *(飞书)* 添加获取群组历史消息功能

### 🚜 Refactor

- *(constant)* 集中管理常量定义并更新相关引用
- *(config)* 提取路径处理逻辑到独立函数
- *(channel)* 重构飞书消息处理逻辑

### 📚 Documentation

- 更新README.md添加v0.2.6飞书群聊功能说明
## [0.2.5] - 2026-03-02

### 🚀 Features

- 重构任务管理模块并添加子任务支持
- 将默认配置文件路径改为用户主目录

### 🚜 Refactor

- 简化技能加载配置并调整默认路径
- *(config)* 修改配置结构为指针类型并添加空指针检查
## [0.2.4] - 2026-03-02

### 🚀 Features

- *(task/cron)* 添加任务和定时任务的更新功能
- *(cron)* 添加一次性任务支持

### 🐛 Bug Fixes

- 修正 frontmatter 正则表达式匹配问题

### 🚜 Refactor

- *(constant)* 将常量定义移动到pkg/constant并更新引用
## [0.2.3] - 2026-03-01

### 🐛 Bug Fixes

- *(cron)* 定时任务不起作用的bug
## [0.2.2] - 2026-03-01

### 🚀 Features

- *(文件系统工具)* 为写入文件工具添加追加模式支持
- *(agent)* 添加对话历史压缩功能以优化长对话处理
- *(agent)* 添加任务总结归档功能并优化任务处理逻辑

### 🚜 Refactor

- *(profile)* 移除文档元数据并更新代理提示

### 📚 Documentation

- *(agent)* 更新提示信息和消息工具描述
## [0.2.1] - 2026-02-28

### 🚀 Features

- *(workspace)* 重构工作区配置文件和目录结构
## [0.2.0] - 2026-02-28

### 🚀 Features

- *(channel)* 添加Web渠道支持并重构配置结构
## [0.1.0] - 2026-02-27

### 🚀 Features

- *(llm)* 添加支持JSON Schema的结构化输出功能
- *(agent)* 增强任务处理逻辑和提示信息
- *(docker)* 添加Docker支持及CI构建流程

### 🐛 Bug Fixes

- *(shell)* 改进Windows下PowerShell命令的执行处理

### 💼 Other

- *(Dockerfile)* 支持多架构构建

### 🚜 Refactor

- *(config)* 重构配置系统以支持多LLM提供商

### 📚 Documentation

- 更新 README 以更全面描述框架功能
- 更新引导文档和README内容

### ⚙️ Miscellaneous Tasks

- 更新构建工作流的镜像标签格式
## [0.0.3] - 2026-02-26

### 🚀 Features

- 添加任务管理、定时任务及工具注册功能
- *(shell)* 新增自动交互式工具和菜单分析功能

### 🚜 Refactor

- *(tools)* 重构工具调用以支持工作区参数传递
- *(agent)* 重构代理管理器和任务处理逻辑

### 📚 Documentation

- 添加项目README文件
## [0.0.2] - 2026-02-25

### 🐛 Bug Fixes

- 修复会话保存路径并添加工具调用支持
## [0.0.1] - 2026-02-25

### 🚀 Features

- *(workspace)* 添加工作区初始化模板文件

### 💼 Other

- 初版

### ⚙️ Miscellaneous Tasks

- 添加 GitHub Actions 构建发布工作流
