# Action
由于现有的Skill是一套开放的标准，它包含了指导大模型执行的一系列要求，甚至还自带了各种scripts等内容，用于执行一些特定的任务。
但是，skill在大模型选择执行时，还是会因为大模型的不确定性，导致执行结果的不一致。
为了解决这个问题，我们引入了Action的概念。
> Action是一种类似Skill机制，但由系统Action执行引擎执行的，可审计、可追溯、可调试的单一执行单元。Action主要通过调用Tool来完成一系列动作任务，其自带判断、流程控制等能力，并确保了相同的输入下，Action的执行结果是确定的。

## Capability
由于不同的系统可能会提供不同的tool，甚至相同的tool都会有不同的名称，为了确保Action的可执行性，我们引入了Capability的概念。
Capability是一种描述系统能力的抽象，它包含了系统提供的所有工具。

capability采用<domain>.<action>的格式，例如：time.now、fs.read等。全部小写，不允许驼峰、下划线、多级嵌套

### time
```yaml
time.now
time.sleep
```

### fs
```yaml
fs.read
fs.write
fs.list
fs.move
fs.copy
fs.delete
fs.search
fs.edit
```

### web（非浏览器）
```yaml
web.search
web.fetch
```

### browser
```yaml
browser.open
browser.run_script # 用于浏览器自动化脚本操作
```

### text
```yaml
text.search
```

### llm
```yaml
llm.generate
llm.embed
llm.rerank
```

### knowledge
```yaml
knowledge.search
knowledge.store
```

## Action DSL V1 规范
```markdown
---
# YAML 元信息区（AI读取）
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

### Step结构
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
统一使用模板语法`{{...}}`来引用context中的数据。

可引用数据源
```
{{input.xxx}}                        # 输入参数

{{steps.step_id.structured.xxx}}     # 结构化输出（推荐）

{{steps.step_id.raw}}                # 原始输出（文本）

{{item.xxx}}                         # 循环变量
```
注意，这里需要Tool的输出结果支持结构化数据，tool可以输出 structred及raw数据，可以由适配层（mapping）来处理（如调用大模型再次结构化），也可以直接由tool支持结构化输出。

### Tool输出约定
```yaml
structured: any     # 给 DSL 使用（可为空）
raw: string         # 给 LLM(当然，llm也可以直接使用结构化的输出) / 展示（文本）
```

### 执行上下文
```
context = {
  input,
  steps: {
    step_id: {
      structured,
      raw
    }
  }
}
```

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

