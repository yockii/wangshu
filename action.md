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
browser.open       # 打开浏览器
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