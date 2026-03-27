# Action 机制

> **创新点**：Action 是望舒提出的一种确定性执行机制，解决了 LLM 调用工具时结果不确定的核心问题。

## 背景

现有的 Skill 是一套开放的标准，它包含了指导大模型执行的一系列要求，甚至还自带了各种 scripts 等内容，用于执行一些特定的任务。

但是，Skill 在大模型选择执行时，会因为大模型的不确定性，导致执行结果的不一致。同样的输入，不同的执行可能产生不同的结果，这对于需要稳定、可预测输出的场景是一个严重的问题。

## 什么是 Action

**Action 是一种类似 Skill 的机制，但由系统 Action 执行引擎执行的，可审计、可追溯、可调试的单一执行单元。**

Action 主要通过调用 Tool 来完成一系列动作任务，其自带判断、流程控制等能力，并确保了**相同的输入下，Action 的执行结果是确定的**。

### 核心特性

| 特性 | 说明 |
|------|------|
| **确定性执行** | 相同输入 → 相同输出，消除 LLM 不确定性 |
| **可审计** | 完整的执行日志，每一步都可追溯 |
| **可调试** | 支持单步执行、断点、变量查看 |
| **流程控制** | 内置条件判断、循环、错误处理 |
| **结构化输出** | Tool 输出结构化数据，便于 DSL 引用 |

### Action vs Skill 对比

| 维度 | Skill | Action |
|------|-------|--------|
| 执行方式 | LLM 驱动 | 引擎驱动 |
| 结果确定性 | 不确定 | **确定** |
| 可审计性 | 弱 | **强** |
| 适用场景 | 开放式任务 | 固定流程任务 |
| 编写门槛 | 低 | 中等（DSL） |

---

## Capability

由于不同的系统可能会提供不同的 Tool，甚至相同的 Tool 都会有不同的名称，为了确保 Action 的可移植性和可执行性，我们引入了 **Capability** 的概念。

**Capability 是一种描述系统能力的抽象**，它屏蔽了底层工具的具体实现细节。

### 命名规范

Capability 采用 `<domain>.<action>` 的格式：

- 全部小写
- 不允许驼峰、下划线
- 不允许多级嵌套

**示例**：`time.now`、`fs.read`、`web.search`

### 标准 Capability 清单

#### time
```yaml
time.now      # 获取当前时间
time.sleep    # 延时执行
```

#### fs（文件系统）
```yaml
fs.read       # 读取文件
fs.write      # 写入文件
fs.list       # 列出目录
fs.move       # 移动文件
fs.copy       # 复制文件
fs.delete     # 删除文件
fs.search     # 搜索文件
fs.edit       # 编辑文件
```

#### web（网络请求）
```yaml
web.search    # 网络搜索
web.fetch     # 获取网页内容
```

#### browser（浏览器自动化）
```yaml
browser.open       # 打开浏览器页面
browser.click      # 点击元素
browser.fill       # 填充表单
browser.html       # 获取页面HTML
browser.screenshot # 截图
browser.wait       # 等待元素
browser.close      # 关闭浏览器
browser.run_script # 执行浏览器脚本
```

#### text（文本处理）
```yaml
text.search   # 文本搜索
```

#### llm（大语言模型）
```yaml
llm.generate  # 生成文本
llm.embed     # 文本向量化
llm.rerank    # 重排序
```

#### knowledge（知识库）
```yaml
knowledge.search   # 知识检索
knowledge.store    # 知识存储
```

#### message（消息通知）
```yaml
message.send       # 发送消息
```

---

## Capability 响应数据规范

每个 Capability 执行后返回统一的 `ActionOutput` 结构：

```json
{
  "status": "success | failed",
  "message": "执行结果描述",
  "data": { ... },
  "trace": { ... }
}
```

其中 `data` 字段根据不同 Capability 有不同的结构，详见下文。

---

### time.now

获取当前时间。

**输入参数**：无

**输出数据**：

```json
{
  "timestamp": "2024-01-15T10:30:00Z"
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| timestamp | string | ISO 8601 格式的时间戳 |

---

### time.sleep

延时执行。

**输入参数**：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| duration | string | 是 | 延时时长（如 `1s`, `500ms`, `1m`） |

**输出数据**：

```json
{
  "status": "completed",
  "duration": "1s"
}
```

---

### fs.read

读取文件内容。

**输入参数**：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| path | string | 是 | 文件路径（支持 `~` 扩展） |

**输出数据**：

```json
{
  "file": "/path/to/file.txt",
  "content": "文件内容...",
  "type": "text"
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| file | string | 文件路径 |
| content | string | 文件内容 |
| type | string | 文件类型（text, pdf, docx, xlsx 等） |

---

### fs.write

写入文件。

**输入参数**：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| path | string | 是 | 文件路径（支持 `~` 扩展） |
| content | string | 是 | 要写入的内容 |
| append | boolean | 否 | 是否追加模式，默认 false |

**输出数据**：

```json
{
  "file": "/path/to/file.txt",
  "content_written": "写入的内容...",
  "created": true
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| file | string | 文件路径 |
| content_written | string | 已写入的内容 |
| created | bool | 是否新建文件 |

---

### fs.list

列出目录内容。

**输入参数**：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| path | string | 是 | 目录路径（支持 `~` 扩展） |

**输出数据**：

```json
{
  "path": "/path/to/dir",
  "items": [
    { "name": "file1.txt", "is_dir": false, "size": 1024 },
    { "name": "subdir", "is_dir": true, "size": 0 }
  ]
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| path | string | 目录路径 |
| items | array | 目录项列表 |
| items[].name | string | 文件/目录名 |
| items[].is_dir | bool | 是否为目录 |
| items[].size | int | 文件大小（字节） |

---

### fs.copy

复制文件。

**输入参数**：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| src | string | 是 | 源文件路径 |
| dest | string | 是 | 目标文件路径 |
| overwrite | boolean | 否 | 是否覆盖已存在文件，默认 false |

**输出数据**：

```json
{
  "src": "/path/to/source.txt",
  "dest": "/path/to/dest.txt",
  "success": true
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| src | string | 源文件路径 |
| dest | string | 目标文件路径 |
| success | bool | 是否成功 |

---

### fs.move

移动/重命名文件。

**输入参数**：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| old_path | string | 是 | 原文件路径 |
| new_path | string | 是 | 新文件路径 |

**输出数据**：

```json
{
  "old_path": "/path/to/old.txt",
  "new_path": "/path/to/new.txt",
  "success": true
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| old_path | string | 原路径 |
| new_path | string | 新路径 |
| success | bool | 是否成功 |

---

### fs.delete

删除文件或目录。

**输入参数**：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| path | string | 是 | 要删除的文件或目录路径 |

**输出数据**：

```json
{
  "path": "/path/to/file.txt",
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| path | string | 删除的路径 |

---

### fs.edit

编辑文件内容（替换文本块）。

**输入参数**：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| file_path | string | 是 | 文件路径 |
| old_str | string | 是 | 要替换的文本（必须唯一匹配） |
| new_str | string | 是 | 替换后的文本 |

**输出数据**：

```json
{
  "file": "/path/to/file.txt",
  "replaced_text": "被替换的文本...",
  "success": true
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| file | string | 文件路径 |
| replaced_text | string | 被替换的文本 |
| success | bool | 是否成功 |

---

### fs.search

搜索文件。

**输入参数**：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| pattern | string | 是 | Glob 匹配模式（如 `*.go`, `**/main.go`） |

**输出数据**：

```json
{
  "pattern": "*.go",
  "matches": [
    "/path/to/main.go",
    "/path/to/utils.go"
  ]
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| pattern | string | 搜索模式 |
| matches | array | 匹配的文件路径列表 |

---

### text.search

文件内容搜索。

**输入参数**：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| pattern | string | 是 | 正则表达式模式 |
| path | string | 否 | 搜索目录，默认当前目录 |
| include | string | 否 | 文件过滤模式（如 `*.go`） |

**输出数据**：

```json
{
  "pattern": "func main",
  "matches": [
    { "path": "/path/to/main.go", "line": 10, "text": "func main() {" },
    { "path": "/path/to/other.go", "line": 5, "text": "func mainHelper() {" }
  ]
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| pattern | string | 搜索模式 |
| matches | array | 匹配结果列表 |
| matches[].path | string | 文件路径 |
| matches[].line | int | 行号 |
| matches[].text | string | 匹配的文本行 |

---

### web.search

网络搜索。

**输入参数**：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| query | string | 是 | 搜索关键词 |
| num_results | int | 否 | 返回结果数量，默认 10 |
| engine | string | 否 | 搜索引擎：`baidu`, `duckduckgo`, `auto` |

**输出数据**：

```json
{
  "query": "golang tutorial",
  "results": [
    {
      "title": "Go 语言教程",
      "url": "https://example.com/go-tutorial",
      "snippet": "Go 是一门开源编程语言..."
    }
  ]
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| query | string | 搜索关键词 |
| results | array | 搜索结果列表 |
| results[].title | string | 页面标题 |
| results[].url | string | 页面 URL |
| results[].snippet | string | 内容摘要 |

---

### web.fetch

获取网页内容。

**输入参数**：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| url | string | 是 | 目标 URL |
| timeout | int | 否 | 超时时间（秒），默认 10 |
| raw | boolean | 否 | 是否返回原始内容，默认 false |

**输出数据**：

```json
{
  "url": "https://example.com",
  "content": "<html>...",
  "status_code": 200,
  "headers": {
    "content-type": "text/html; charset=utf-8"
  }
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| url | string | 请求的 URL |
| content | string | 响应内容 |
| status_code | int | HTTP 状态码 |
| headers | object | 响应头 |

---

### browser.open

打开浏览器页面。

**输入参数**：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| url | string | 是 | 要打开的 URL |
| timeout | int | 否 | 超时时间（毫秒） |

**输出数据**：

```json
{
  "url": "https://example.com",
  "elements": [
    {
      "tag": "input",
      "visible": true,
      "enabled": true,
      "editable": true,
      "id_selector": "username",
      "type": "text",
      "placeholder": "请输入用户名"
    }
  ]
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| url | string | 页面 URL |
| elements | array | 页面元素列表（见 ElementInfo 结构） |

---

### browser.click

点击元素。

**输入参数**：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| selector | string | 是 | CSS 选择器 |
| timeout | int | 否 | 超时时间（毫秒） |

**输出数据**：

```json
{
  "elements": [
    {
      "tag": "button",
      "visible": true,
      "enabled": true,
      "text": "提交",
      "xpath_selector": "//button[@type='submit']"
    }
  ]
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| elements | array | 操作后的元素列表（见 ElementInfo 结构） |

---

### browser.fill

填充表单。

**输入参数**：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| selector | string | 是 | CSS 选择器 |
| text | string | 是 | 要填充的文本 |
| timeout | int | 否 | 超时时间（毫秒） |

**输出数据**：

```json
{
  "elements": [
    {
      "tag": "input",
      "visible": true,
      "enabled": true,
      "value": "填充的内容"
    }
  ]
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| elements | array | 操作后的元素列表（见 ElementInfo 结构） |

---

### browser.html

获取页面 HTML。

**输入参数**：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| format | string | 否 | 格式：`full`, `body`, `inner`, `text`，默认 `body` |
| start | int | 否 | 起始位置（字符偏移），用于分页，默认 0 |
| max_length | int | 否 | 最大获取长度，默认 50000 |

**输出数据**：

```json
{
  "format": "html",
  "start": 0,
  "max_length": 10000,
  "content": "<html><body>...",
  "next_start": 10000
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| format | string | 内容格式 |
| start | int | 起始位置 |
| max_length | int | 最大长度 |
| content | string | HTML 内容 |
| next_start | int | 下一页起始位置（用于分页） |

---

### browser.screenshot

截图。

**输入参数**：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| screenshot_path | string | 是 | 截图保存路径 |

**输出数据**：

```json
{
  "path": "/path/to/screenshot.png"
}
```

---

### browser.wait

等待元素。

**输入参数**：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| selector | string | 是 | CSS 选择器 |
| timeout | int | 否 | 超时时间（毫秒） |

**输出数据**：

```json
{
  "elements": [...]
}
```

---

### browser.close

关闭浏览器。

**输入参数**：

无

**输出数据**：

```json
{
  "status": "closed"
}
```

---

### message.send

发送消息。

**输入参数**：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| content | string | 是 | 消息内容 |
| fileType | string | 否 | 文件类型：`image`, `file` |
| filePath | string | 否 | 文件路径（当 fileType 有值时必填） |

**输出数据**：

```json
{
  "channel": "feishu",
  "message_id": "msg_123456",
  "timestamp": "2024-01-15T10:30:00Z"
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| channel | string | 渠道名称 |
| message_id | string | 消息 ID |
| timestamp | string | 发送时间戳 |

---

### ElementInfo 结构

浏览器相关 Capability 返回的元素信息结构：

```json
{
  "tag": "input",
  "visible": true,
  "enabled": true,
  "editable": true,
  "id_selector": "username",
  "name_selector": "user",
  "class_selector": "form-input",
  "xpath_selector": "//input[@id='username']",
  "data_selectors": {
    "testid": "login-username"
  },
  "type": "text",
  "name": "username",
  "placeholder": "请输入用户名",
  "value": "",
  "text": "",
  "href": "",
  "aria_label": "用户名输入框",
  "readonly": false,
  "required": true,
  "checked": false
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| tag | string | HTML 标签名 |
| visible | bool | 是否可见 |
| enabled | bool | 是否可用 |
| editable | bool | 是否可编辑 |
| id_selector | string | ID 选择器 |
| name_selector | string | name 选择器 |
| class_selector | string | class 选择器 |
| xpath_selector | string | XPath 选择器 |
| data_selectors | object | data-* 属性选择器 |
| type | string | input 类型 |
| name | string | 元素 name 属性 |
| placeholder | string | 占位符文本 |
| value | string | 当前值 |
| text | string | 元素文本内容 |
| href | string | 链接地址（a 标签） |
| aria_label | string | ARIA 标签 |
| readonly | bool | 是否只读 |
| required | bool | 是否必填 |
| checked | bool | 是否选中（checkbox/radio） |

---

## Action DSL V1 规范

Action 使用 Markdown 格式定义，包含两个区域：

```markdown
---
# YAML 元信息区（引擎解析）
……
---
# Markdown 说明区（人类阅读）
```

### 顶层结构

```yaml
id: string                # 唯一标识
name: string              # 名称
version: string           # 版本（建议 semver）

description: string       # 简短描述（给 LLM / UI）

capabilities:             # 依赖的 capability
  - string

inputs:                   # 输入定义
  <key>: <type>

outputs:                  # 输出定义（最终结果）
  <key>: <type>

config:                   # 执行控制
  max_steps: number       # 最大步骤数（默认 50）
  max_loop: number        # 默认循环上限（默认 10）
  timeout: string         # 超时（如 30s）

steps:                    # 执行流程（核心）
  - step
```

### Step 结构

```yaml
- id: string              # 唯一标识（必须）

  use: string             # 使用的 capability

  with:                   # 输入参数
    <key>: any

  assign_to: string       # （可选）将结果写入 context

  if: string              # （可选）条件表达式

  for_each: string        # （可选）循环（数组）
  item: string            # （可选）循环变量名（默认 item）
  max_loop: number        # （可选）覆盖全局循环限制

  on_error: string        # （可选）continue | break | retry
  retry: number           # （可选）重试次数
```

### 数据引用规则

统一使用模板语法 `{{...}}` 来引用 context 中的数据。

**可引用数据源：**

```
{{input.xxx}}                        # 输入参数

{{steps.step_id.structured.xxx}}     # 结构化输出（推荐）

{{steps.step_id.raw}}                # 原始输出（文本）

{{item.xxx}}                         # 循环变量
```

### Tool 输出约定

Tool 需要支持双输出模式：

```yaml
structured: any     # 结构化数据，给 DSL 引用
raw: string         # 原始文本，给 LLM/展示使用
```

### 执行上下文

```
context = {
  input,           # 输入参数
  steps: {         # 各步骤输出
    step_id: {
      structured,
      raw
    }
  }
}
```

---

## 完整示例

```markdown
---
id: web.search_and_summarize
name: 搜索并总结信息
version: 1.0.0

description: 搜索指定主题并总结主要内容

capabilities:
  - web.search
  - web.fetch
  - llm.generate

inputs:
  query: string

outputs:
  summary: string

config:
  max_steps: 20
  max_loop: 5
  timeout: 30s

steps:

  - id: search
    use: web.search
    with:
      query: "{{input.query}}"

  - id: fetch_pages
    for_each: "{{steps.search.structured.results}}"
    max_loop: 3
    use: web.fetch
    with:
      url: "{{item.url}}"

  - id: summarize
    use: llm.generate
    with:
      prompt: |
        请总结以下内容：

        {{steps.fetch_pages.raw}}
    assign_to: summary

---

# 功能说明

该 Action 用于搜索某个主题，并提取网页内容后生成总结。

# 输入参数

- query: 搜索关键词

# 输出结果

- summary: 总结内容

# 执行流程

1. 搜索相关网页
2. 获取网页内容（最多3个）
3. 汇总并生成总结

# 注意事项

- 依赖网络访问能力
- 若搜索结果为空，输出可能为空
```

---

## 实现原理

### 执行流程

```
┌─────────────┐
│ Action 定义  │
└──────┬──────┘
       │
       ▼
┌─────────────┐
│ 解析 YAML   │
└──────┬──────┘
       │
       ▼
┌─────────────┐
│ 验证 Capabilities │
└──────┬──────┘
       │
       ▼
┌─────────────┐
│ 创建执行上下文 │
└──────┬──────┘
       │
       ▼
┌─────────────┐
│ 依次执行 Steps │◄─────┐
└──────┬──────┘      │
       │             │
       ▼             │
┌─────────────┐      │
│ 模板替换参数 │      │
└──────┬──────┘      │
       │             │
       ▼             │
┌─────────────┐      │
│ 调用 Tool   │      │
└──────┬──────┘      │
       │             │
       ▼             │
┌─────────────┐      │
│ 存储输出结果 │──────┘
└──────┬──────┘
       │
       ▼
┌─────────────┐
│ 返回最终结果 │
└─────────────┘
```

### 核心组件

| 组件 | 职责 |
|------|------|
| Action Parser | 解析 Markdown 中的 YAML 元信息 |
| Capability Registry | 注册和管理系统能力 |
| Execution Context | 维护执行过程中的变量和状态 |
| Step Executor | 执行单个步骤，处理循环和条件 |
| Tool Adapter | 将 Tool 适配为 Capability |

---

## 最佳实践

### 1. 合理划分步骤

每个 Step 应该是一个原子操作，便于调试和复用。

### 2. 善用结构化输出

优先使用 `structured` 输出进行数据传递，避免文本解析。

### 3. 设置合理的限制

- `max_steps`: 防止无限执行
- `max_loop`: 控制循环次数
- `timeout`: 避免长时间阻塞

### 4. 完善文档说明

Markdown 说明区应该清晰描述：
- 功能说明
- 输入参数
- 输出结果
- 执行流程
- 注意事项

---

## 未来规划

- [ ] 可视化 Action 编辑器
- [ ] Action 市场（分享和复用）
- [ ] 更多内置 Capability
- [ ] 条件表达式增强
- [ ] 并行执行支持