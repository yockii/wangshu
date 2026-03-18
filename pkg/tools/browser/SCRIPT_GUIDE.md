# 浏览器自动化任务脚本说明书

本文档描述了 `run_task` action 支持的任务脚本格式和所有可用功能。

## 目录

- [快速开始](#快速开始)
- [脚本结构](#脚本结构)
- [变量支持](#变量支持)
- [选择器策略](#选择器策略)
- [Step 步骤](#step-步骤)
- [Action 类型详解](#action-类型详解)
- [检测条件详解](#检测条件详解)
- [数据提取详解](#数据提取详解)
- [条件判断详解](#条件判断详解)
- [错误处理](#错误处理)
- [完整示例](#完整示例)

***

## 快速开始

```json
{
  "action": "run_task",
  "script": {
    "name": "示例任务",
    "steps": [
      {"id": "s1", "action": "open", "params": {"url": "https://example.com"}},
      {"id": "s2", "action": "extract", "fields": {"title": {"selector": "h1", "attr": "text"}}}
    ]
  }
}
```

***

## 脚本结构

```typescript
interface TaskScript {
  name: string;              // 任务名称
  description?: string;      // 任务描述（可选）
  steps: Step[];             // 步骤列表
}
```

### 示例

```json
{
  "name": "飞书应用配置",
  "description": "自动创建飞书应用并获取凭证",
  "steps": [...]
}
```

***

## 变量支持

脚本支持变量替换，格式为 `${var_name}`。变量可以在调用时传入，实现脚本复用。

### 使用示例

```json
{
  "name": "打开应用配置页",
  "steps": [
    {
      "id": "s1_open",
      "action": "open",
      "params": {
        "url": "https://open.feishu.cn/app/${app_id}/config"
      }
    },
    {
      "id": "s2_fill",
      "action": "fill",
      "params": {
        "label": "应用名称",
        "value": "${app_name}"
      }
    }
  ]
}
```

### 调用时传入变量

```go
params := map[string]string{
    "script": scriptJSON,
    "variables": `{
        "app_id": "cli_a9143b997b38dcd6",
        "app_name": "望舒"
    }`,
}
```

### 变量支持范围

变量可以在以下位置使用：

- URL 地址
- 选择器（selector、within 等）
- 填充值（value）
- 文本内容（text、label 等）
- 所有字符串参数

***

## 选择器策略

现代前端框架（React、Vue 等）生成的 DOM 往往缺乏稳定的 ID 或 class，本工具提供多种选择器策略来应对这种情况。

### 支持的定位参数

| 参数          | 类型     | 说明                  | 示例             |
| ----------- | ------ | ------------------- | -------------- |
| selector    | string | 直接指定 CSS 选择器        | `#submit-btn`  |
| label       | string | 通过 label 文本定位表单元素   | `"用户名"`        |
| text        | string | 通过元素文本内容定位          | `"登录"`         |
| role        | string | 通过 ARIA 角色定位        | `button`       |
| role\_name  | string | 配合 role 使用，指定角色名称   | `"提交"`         |
| testid      | string | 通过 data-testid 属性定位 | `"submit-btn"` |
| placeholder | string | 通过 placeholder 属性定位 | `"请输入用户名"`     |
| title       | string | 通过 title 属性定位       | `"关闭"`         |
| alt         | string | 通过 alt 属性定位图片       | `"商品图片"`       |
| near        | string | 定位某文本附近的元素          | `"确定"`         |
| tag         | string | 配合 near 使用，指定标签名    | `button`       |

### 范围限定参数

| 参数            | 类型            | 说明                                   |
| ------------- | ------------- | ------------------------------------ |
| within        | string        | 限定搜索范围的容器选择器                         |
| within\_index | string/number | 选择第几个容器：`"first"`、`"last"` 或数字（从0开始） |

### 元素选择参数

| 参数    | 类型            | 说明                                       |
| ----- | ------------- | ---------------------------------------- |
| index | string/number | 选择第几个元素：`"first"`（默认）、`"last"` 或数字（从0开始） |

### 优先级

当同时提供多个参数时，按以下优先级使用：

`selector` > `label` > `text` > `role` > `testid` > `placeholder` > `title` > `alt` > `near`

### 使用示例

#### 1. 通过 label 定位表单（推荐）

适用于表单输入框，即使 input 没有 ID，只要有关联的 label 即可：

```json
{
  "action": "fill",
  "params": {
    "label": "应用名称",
    "value": "my-app"
  }
}
```

#### 2. 通过文本定位按钮

```json
{
  "action": "click",
  "params": {
    "text": "创建应用"
  }
}
```

#### 3. 通过 placeholder 定位输入框

```json
{
  "action": "fill",
  "params": {
    "placeholder": "请输入应用名称",
    "value": "my-app"
  }
}
```

#### 4. 通过 ARIA 角色定位

```json
{
  "action": "click",
  "params": {
    "role": "button",
    "role_name": "提交"
  }
}
```

#### 5. 通过 data-testid 定位（最稳定）

如果页面有 data-testid 属性，这是最稳定的方式：

```json
{
  "action": "click",
  "params": {
    "testid": "create-app-btn"
  }
}
```

#### 6. 组合定位（near）

定位某文本附近的元素：

```json
{
  "action": "click",
  "params": {
    "tag": "button",
    "near": "应用名称"
  }
}
```

#### 7. 限定搜索范围（within）

在有多个相同元素时，限定搜索范围：

```json
{
  "action": "click",
  "params": {
    "text": "添加",
    "within": ".ability-card:has-text('机器人')"
  }
}
```

#### 8. 选择特定元素（index）

选择多个匹配元素中的特定一个：

```json
{
  "action": "click",
  "params": {
    "selector": ".list-item",
    "index": "last"
  }
}
```

```json
{
  "action": "click",
  "params": {
    "selector": ".list-item",
    "index": 2
  }
}
```

#### 9. 组合使用 within 和 index

```json
{
  "action": "click",
  "params": {
    "selector": "button",
    "within": ".card",
    "within_index": 1,
    "index": "last"
  }
}
```

含义：在第2个 `.card` 容器中，点击最后一个 `button`。

### Playwright 高级选择器

除了上述参数，`selector` 参数还支持 Playwright 的高级选择器语法：

#### 文本选择器

```json
{"selector": "text=登录"}
{"selector": "button:has-text('提交')"}
{"selector": ":text('确定')"}
```

#### 组合选择器

```json
{"selector": "label:has-text('用户名') >> input"}
{"selector": ".card >> text=详情"}
```

#### 位置选择器

```json
{"selector": "button >> nth=0"}
{"selector": "button >> first"}
{"selector": "button >> last"}
```

#### 可见性过滤

```json
{"selector": "button:visible"}
{"selector": "button:hidden"}
```

#### has 选择器

```json
{"selector": ".card:has-text('机器人')"}
{"selector": ".card:has(button)"}
```

### 最佳实践

1. **优先使用语义化定位**：`label`、`text`、`role` 比位置选择器更稳定
2. **data-testid 是首选**：如果可以控制页面代码，添加 data-testid 属性
3. **避免使用 nth-child**：`div:nth-child(3)` 这类选择器极易失效
4. **组合使用提高精度**：`within` + `index` 可以精确定位复杂结构
5. **使用变量复用脚本**：通过 `${var_name}` 实现脚本模板化

***

## Step 步骤

每个步骤的基本结构：

```typescript
interface Step {
  id: string;                    // 步骤唯一标识
  action: string;                // 动作类型
  description?: string;          // 步骤描述（可选）
  params?: object;               // 动作参数
  timeout?: number;              // 超时时间（毫秒）
  on_error?: string;             // 错误处理策略: "fail"(默认) | "continue"
  
  // wait_for_user 专用
  detect?: DetectCondition;      // 检测条件
  
  // extract 专用
  fields?: object;               // 字段配置
  
  // condition 专用
  check?: CheckCondition;        // 检查条件
  then?: Step[];                 // 条件满足时执行
  else?: Step[];                 // 条件不满足时执行
}
```

***

## Action 类型详解

### 1. open - 打开页面

打开指定的 URL。

```json
{
  "id": "step1",
  "action": "open",
  "params": {
    "url": "https://example.com"
  }
}
```

| 参数  | 类型     | 必填 | 说明       |
| --- | ------ | -- | -------- |
| url | string | 是  | 要打开的 URL |

***

### 2. click - 点击元素

点击页面上的元素。

```json
{
  "id": "step2",
  "action": "click",
  "params": {
    "text": "创建应用",
    "within": ".modal"
  }
}
```

| 参数            | 类型            | 必填 | 说明                |
| ------------- | ------------- | -- | ----------------- |
| selector      | string        | 否  | CSS 选择器           |
| label         | string        | 否  | 通过 label 文本定位     |
| text          | string        | 否  | 通过元素文本定位          |
| role          | string        | 否  | 通过 ARIA 角色定位      |
| testid        | string        | 否  | 通过 data-testid 定位 |
| placeholder   | string        | 否  | 通过 placeholder 定位 |
| within        | string        | 否  | 限定搜索范围            |
| within\_index | string/number | 否  | 选择第几个容器           |
| index         | string/number | 否  | 选择第几个元素           |

> **提示**：选择器参数至少提供一个，优先级参见[选择器策略](#选择器策略)。

***

### 3. fill - 填充表单

向输入框填充文本。

```json
{
  "id": "step3",
  "action": "fill",
  "params": {
    "label": "应用名称",
    "value": "my-app"
  }
}
```

| 参数            | 类型            | 必填 | 说明                    |
| ------------- | ------------- | -- | --------------------- |
| value         | string        | 是  | 要填充的值                 |
| selector      | string        | 否  | CSS 选择器               |
| label         | string        | 否  | 通过 label 文本定位（推荐用于表单） |
| text          | string        | 否  | 通过元素文本定位              |
| role          | string        | 否  | 通过 ARIA 角色定位          |
| testid        | string        | 否  | 通过 data-testid 定位     |
| placeholder   | string        | 否  | 通过 placeholder 定位     |
| within        | string        | 否  | 限定搜索范围                |
| within\_index | string/number | 否  | 选择第几个容器               |
| index         | string/number | 否  | 选择第几个元素               |

> **提示**：选择器参数至少提供一个，优先级参见[选择器策略](#选择器策略)。

***

### 4. wait - 等待

支持三种等待模式：等待元素状态、等待指定时间。

#### 等待元素状态

```json
{
  "id": "step4",
  "action": "wait",
  "params": {
    "selector": ".loading",
    "state": "hidden",
    "timeout": 10000
  }
}
```

#### 等待指定时间

```json
{
  "id": "step4",
  "action": "wait",
  "params": {
    "duration": 1000
  }
}
```

| 参数       | 类型     | 必填     | 说明                                                |
| -------- | ------ | ------ | ------------------------------------------------- |
| duration | number | 否      | 直接等待指定时间（毫秒）                                      |
| state    | string | 否      | 元素状态：`visible`（默认）、`hidden`、`attached`、`detached` |
| timeout  | number | 否      | 超时时间（毫秒），默认 30000                                 |
| selector | string | 否      | CSS 选择器                                           |
| text     | string | 否      | 通过元素文本定位                                          |
| ...      | <br /> | <br /> | 其他选择器参数                                           |

**state 参数说明**：

| state    | 说明           |
| -------- | ------------ |
| visible  | 等待元素可见（默认）   |
| hidden   | 等待元素隐藏/消失    |
| attached | 等待元素附加到 DOM  |
| detached | 等待元素从 DOM 移除 |

**使用场景**：

```json
// 等待加载动画消失
{
  "action": "wait",
  "params": {
    "selector": ".spinner",
    "state": "hidden"
  }
}

// 等待对话框关闭
{
  "action": "wait",
  "params": {
    "selector": ".modal",
    "state": "hidden"
  }
}

// 操作后等待页面稳定
{
  "action": "wait",
  "params": {
    "duration": 1000
  }
}
```

***

### 5. wait\_for\_user - 等待用户操作

等待用户完成某些操作（如登录），通过检测条件判断是否完成。

```json
{
  "id": "step5",
  "action": "wait_for_user",
  "description": "请在浏览器中登录飞书开放平台",
  "timeout": 300000,
  "detect": {
    "condition": "url_contains",
    "value": "/app"
  }
}
```

| 参数          | 类型     | 必填 | 说明                 |
| ----------- | ------ | -- | ------------------ |
| description | string | 否  | 提示信息，会显示给用户        |
| timeout     | number | 否  | 超时时间（毫秒），默认 300000 |
| detect      | object | 是  | 检测条件配置             |

***

### 6. extract - 提取数据

从页面提取数据，支持单值和列表。

**提取单个字段**：

```json
{
  "id": "step6",
  "action": "extract",
  "fields": {
    "title": {"selector": "h1", "attr": "text"},
    "price": {"selector": ".price", "attr": "text"},
    "image": {"selector": "img.main", "attr": "src"}
  }
}
```

**提取列表数据**：

```json
{
  "id": "step7",
  "action": "extract",
  "fields": {
    "products": {
      "type": "list",
      "container": ".product-list > .item",
      "fields": {
        "name": {"selector": "h3", "attr": "text"},
        "price": {"selector": ".price", "attr": "text"},
        "link": {"selector": "a", "attr": "href"}
      }
    }
  }
}
```

| 字段配置      | 类型     | 必填 | 说明                     |
| --------- | ------ | -- | ---------------------- |
| selector  | string | 否  | CSS 选择器                |
| label     | string | 否  | 通过 label 文本定位          |
| text      | string | 否  | 通过元素文本定位               |
| attr      | string | 是  | 要提取的属性                 |
| type      | string | 否  | 类型：留空=单值，"list"=列表     |
| container | string | 条件 | 列表容器选择器（type=list 时必填） |
| fields    | object | 条件 | 列表项字段配置（type=list 时必填） |

**支持的 attr 值**：

| attr             | 说明          |
| ---------------- | ----------- |
| text             | 元素文本内容（默认）  |
| value            | 输入框的值       |
| html / innerHTML | 元素内部 HTML   |
| src              | 图片/资源地址     |
| href             | 链接地址        |
| alt              | 替代文本        |
| title            | 标题属性        |
| placeholder      | 占位符         |
| 其他属性             | 任意 HTML 属性名 |

***

### 7. screenshot - 截图

保存当前页面的截图。

```json
{
  "id": "step8",
  "action": "screenshot",
  "params": {
    "path": "screenshot.png"
  }
}
```

| 参数   | 类型     | 必填 | 说明            |
| ---- | ------ | -- | ------------- |
| path | string | 否  | 截图保存路径，默认自动生成 |

***

### 8. scroll - 滚动页面

滚动页面到指定方向。

```json
{
  "id": "step9",
  "action": "scroll",
  "params": {
    "direction": "down",
    "amount": 500
  }
}
```

| 参数        | 类型     | 必填 | 说明                   |
| --------- | ------ | -- | -------------------- |
| direction | string | 否  | 方向："down"(默认) 或 "up" |
| amount    | number | 否  | 滚动像素，默认 500          |

***

### 9. hover - 悬停元素

将鼠标悬停在指定元素上。

```json
{
  "id": "step10",
  "action": "hover",
  "params": {
    "text": "菜单项"
  }
}
```

| 参数       | 类型     | 必填 | 说明                |
| -------- | ------ | -- | ----------------- |
| selector | string | 否  | CSS 选择器           |
| label    | string | 否  | 通过 label 文本定位     |
| text     | string | 否  | 通过元素文本定位          |
| role     | string | 否  | 通过 ARIA 角色定位      |
| testid   | string | 否  | 通过 data-testid 定位 |

> **提示**：选择器参数至少提供一个。

***

### 10. select - 下拉选择

选择下拉框中的选项。

```json
{
  "id": "step11",
  "action": "select",
  "params": {
    "label": "国家",
    "value": "china"
  }
}
```

| 参数       | 类型     | 必填 | 说明          |
| -------- | ------ | -- | ----------- |
| selector | string | 是  | CSS 选择器     |
| value    | string | 是  | 选项的 value 值 |

***

### 11. condition - 条件判断

根据条件执行不同的步骤。

```json
{
  "id": "step12",
  "action": "condition",
  "check": {
    "selector": ".ability-card:has-text('机器人') button:has-text('添加')",
    "exists": true
  },
  "then": [
    {
      "id": "t1",
      "action": "click",
      "params": {
        "text": "添加",
        "within": ".ability-card:has-text('机器人')"
      }
    }
  ],
  "else": []
}
```

***

### 12. goto - 跳转页面

在当前页面跳转到新 URL。

```json
{
  "id": "step13",
  "action": "goto",
  "params": {
    "url": "https://example.com/page2"
  }
}
```

***

### 13. back - 后退

浏览器后退。

```json
{
  "id": "step14",
  "action": "back"
}
```

***

### 14. refresh - 刷新

刷新当前页面。

```json
{
  "id": "step15",
  "action": "refresh"
}
```

***

### 15. clipboard - 剪贴板操作

点击复制按钮并读取剪贴板内容，用于获取隐藏的密钥等信息。

**基本用法**：

```json
{
  "id": "step16",
  "action": "clipboard",
  "params": {
    "selector": ".copy-btn",
    "field": "app_secret"
  }
}
```

**使用 within 限定范围**：

```json
{
  "id": "step16",
  "action": "clipboard",
  "params": {
    "selector": "svg[data-icon='CopyOutlined']",
    "within": ".auth-info__secret",
    "field": "app_secret"
  }
}
```

**不点击直接读取剪贴板**：

```json
{
  "id": "step17",
  "action": "clipboard",
  "params": {
    "field": "clipboard_content"
  }
}
```

| 参数       | 类型     | 必填     | 说明                   |
| -------- | ------ | ------ | -------------------- |
| field    | string | 否      | 存储字段名，默认 "clipboard" |
| selector | string | 否      | 复制按钮选择器，不提供则直接读取剪贴板  |
| text     | string | 否      | 通过文本定位复制按钮           |
| within   | string | 否      | 限定搜索范围               |
| ...      | <br /> | <br /> | 其他选择器参数              |

> **提示**：如果提供选择器参数，会先点击复制按钮，再读取剪贴板。

***

## 检测条件详解

用于 `wait_for_user` action 的 `detect` 字段。

### url\_changed - URL 变化

检测 URL 是否发生变化。

```json
{
  "detect": {
    "condition": "url_changed",
    "from": "/login",
    "to": "/dashboard"
  }
}
```

| 参数   | 类型     | 必填 | 说明                    |
| ---- | ------ | -- | --------------------- |
| from | string | 否  | 原始 URL 片段，离开此 URL 即触发 |
| to   | string | 否  | 目标 URL 片段，到达此 URL 即触发 |

`from` 和 `to` 至少填一个。

***

### url\_contains - URL 包含

检测 URL 是否包含指定字符串。

```json
{
  "detect": {
    "condition": "url_contains",
    "value": "/app"
  }
}
```

| 参数    | 类型     | 必填 | 说明          |
| ----- | ------ | -- | ----------- |
| value | string | 是  | URL 应包含的字符串 |

***

### element\_appear - 元素出现

检测指定元素是否出现在页面上。

```json
{
  "detect": {
    "condition": "element_appear",
    "selector": ".user-avatar"
  }
}
```

| 参数       | 类型     | 必填 | 说明      |
| -------- | ------ | -- | ------- |
| selector | string | 是  | CSS 选择器 |

***

### element\_disappear - 元素消失

检测指定元素是否从页面上消失。

```json
{
  "detect": {
    "condition": "element_disappear",
    "selector": ".loading-spinner"
  }
}
```

| 参数       | 类型     | 必填 | 说明      |
| -------- | ------ | -- | ------- |
| selector | string | 是  | CSS 选择器 |

***

### manual\_confirm - 手动确认

等待用户在控制台按 Enter 键确认。

```json
{
  "detect": {
    "condition": "manual_confirm"
  }
}
```

***

## 数据提取详解

### 单值提取

**使用 selector**：

```json
{
  "fields": {
    "field_name": {
      "selector": "css-selector",
      "attr": "text"
    }
  }
}
```

**使用 label 定位（推荐用于表单）**：

```json
{
  "fields": {
    "app_id": {
      "label": "App ID",
      "attr": "value"
    }
  }
}
```

**使用 text 定位**：

```json
{
  "fields": {
    "title": {
      "text": "商品详情",
      "attr": "text"
    }
  }
}
```

### 字段配置参数

| 参数          | 类型     | 必填 | 说明                                |
| ----------- | ------ | -- | --------------------------------- |
| attr        | string | 是  | 要提取的属性：text/value/html/src/href 等 |
| selector    | string | 否  | CSS 选择器                           |
| label       | string | 否  | 通过 label 文本定位                     |
| text        | string | 否  | 通过元素文本定位                          |
| role        | string | 否  | 通过 ARIA 角色定位                      |
| testid      | string | 否  | 通过 data-testid 定位                 |
| placeholder | string | 否  | 通过 placeholder 定位                 |

### 列表提取

```json
{
  "fields": {
    "list_name": {
      "type": "list",
      "container": ".item-container",
      "fields": {
        "sub_field": {"selector": ".sub-selector", "attr": "text"}
      }
    }
  }
}
```

### 嵌套提取示例

提取商品列表，每个商品包含名称、价格：

```json
{
  "fields": {
    "products": {
      "type": "list",
      "container": ".product-item",
      "fields": {
        "name": {"selector": "h3", "attr": "text"},
        "price": {"selector": ".price", "attr": "text"}
      }
    }
  }
}
```

***

## 条件判断详解

用于 `condition` action 的 `check` 字段。

```typescript
interface CheckCondition {
  selector?: string;        // CSS 选择器
  label?: string;           // 通过 label 文本定位
  text?: string;            // 通过元素文本定位
  role?: string;            // 通过 ARIA 角色定位
  testid?: string;          // 通过 data-testid 定位
  placeholder?: string;     // 通过 placeholder 定位
  exists?: boolean;         // 元素是否存在
  visible?: boolean;        // 元素是否可见
  text_equals?: string;     // 元素文本是否等于
  text_contains?: string;   // 元素文本是否包含
}
```

### 示例

**检查元素是否存在**：

```json
{"selector": ".login-btn", "exists": true}
```

**通过 label 检查输入框是否存在**：

```json
{"label": "用户名", "exists": true}
```

**通过文本检查元素是否存在**：

```json
{"text": "登录", "exists": true}
```

**检查元素是否可见**：

```json
{"selector": ".modal", "visible": true}
```

**检查文本内容**：

```json
{"selector": ".status", "text_equals": "已完成"}
```

```json
{"selector": ".message", "text_contains": "成功"}
```

***

## 错误处理

### 默认行为

当步骤执行失败时，任务会立即终止，返回错误信息和当前步骤截图。

### on\_error 配置

可以设置 `on_error` 为 `"continue"` 来忽略错误继续执行：

```json
{
  "id": "step_optional",
  "action": "click",
  "params": {"selector": ".optional-btn"},
  "on_error": "continue"
}
```

***

## 完整示例

### 示例1：飞书应用配置（带变量）

```json
{
  "name": "飞书应用配置",
  "description": "配置飞书应用并获取凭证",
  "steps": [
    {
      "id": "open_app_page",
      "action": "open",
      "params": {"url": "https://open.feishu.cn/app/${app_id}/config"}
    },
    {
      "id": "wait_page_load",
      "action": "wait",
      "params": {"duration": 1000}
    },
    {
      "id": "check_robot_add",
      "action": "condition",
      "check": {
        "selector": ".ability-card:has-text('机器人') button:has-text('添加')",
        "exists": true
      },
      "then": [
        {
          "id": "click_robot_add",
          "action": "click",
          "params": {
            "text": "添加",
            "within": ".ability-card:has-text('机器人')"
          }
        }
      ],
      "else": []
    },
    {
      "id": "get_app_id",
      "action": "extract",
      "fields": {
        "app_id": {
          "selector": ".auth-info__appid .secret-code__code",
          "attr": "text"
        }
      }
    },
    {
      "id": "get_app_secret",
      "action": "clipboard",
      "params": {
        "selector": "svg[data-icon='CopyOutlined']",
        "within": ".auth-info__secret",
        "field": "app_secret"
      }
    }
  ]
}
```

### 示例2：电商商品采集

```json
{
  "name": "商品信息采集",
  "steps": [
    {
      "id": "open_page",
      "action": "open",
      "params": {"url": "https://shop.example.com/products"}
    },
    {
      "id": "wait_load",
      "action": "wait",
      "params": {
        "selector": ".loading",
        "state": "hidden"
      }
    },
    {
      "id": "scroll_more",
      "action": "scroll",
      "params": {"amount": 1000}
    },
    {
      "id": "wait_content",
      "action": "wait",
      "params": {"duration": 500}
    },
    {
      "id": "extract_products",
      "action": "extract",
      "fields": {
        "products": {
          "type": "list",
          "container": ".product-item",
          "fields": {
            "name": {"selector": "h3", "attr": "text"},
            "price": {"selector": ".price", "attr": "text"},
            "link": {"selector": "a", "attr": "href"}
          }
        }
      }
    }
  ]
}
```

### 示例3：多步骤表单填写

```json
{
  "name": "多步骤表单",
  "steps": [
    {
      "id": "open_form",
      "action": "open",
      "params": {"url": "https://example.com/form"}
    },
    {
      "id": "fill_name",
      "action": "fill",
      "params": {"label": "姓名", "value": "${user_name}"}
    },
    {
      "id": "fill_email",
      "action": "fill",
      "params": {"label": "邮箱", "value": "${user_email}"}
    },
    {
      "id": "select_country",
      "action": "select",
      "params": {"label": "国家", "value": "china"}
    },
    {
      "id": "submit",
      "action": "click",
      "params": {"text": "提交"}
    },
    {
      "id": "wait_success",
      "action": "wait",
      "params": {
        "text": "提交成功",
        "timeout": 10000
      }
    }
  ]
}
```

***

## 最佳实践

1. **选择器优先级**：优先使用 `data-testid`、`label`、`text` 等稳定的选择器，避免使用易变的 class。
2. **合理设置超时**：`wait_for_user` 的超时应足够用户完成操作，建议 5 分钟以上。
3. **错误处理**：对于可选步骤，使用 `on_error: "continue"` 避免任务中断。
4. **分步调试**：复杂任务建议分多个脚本执行，便于定位问题。
5. **截图留证**：关键步骤后添加截图，便于问题排查。
6. **使用变量**：通过 `${var_name}` 实现脚本复用，避免硬编码。
7. **等待策略**：操作后先 `wait duration` 等待页面稳定，再 `wait element` 等待元素出现。
8. **范围限定**：多个相同元素时，使用 `within` 限定范围，避免误操作。
9. **条件判断**：使用 `condition` 处理可选操作，提高脚本健壮性。
10. **剪贴板获取**：隐藏的密钥等无法直接读取的值，使用 `clipboard` action 获取。

