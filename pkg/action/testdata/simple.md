---
id: web.simple_test
name: 简单搜索抓取测试
version: 1.0.0
description: 测试 web.search + web.fetch + for_each

capabilities:
  - web.search
  - web.fetch

inputs:
  query: string

outputs:
  pages: list

config:
  max_steps: 10
  max_loop: 3

steps:

  - id: search
    use: web.search
    with:
      query: "{{input.query}}"

  - id: fetch_pages
    use: web.fetch
    foreach: "steps.search.structured"
    max_loop: 3
    with:
      url: "{{item.url}}"
    assign_to: pages

---

# 说明

1. 先调用 web.search 获取结果列表
2. 遍历结果（最多3条）
3. 使用 web.fetch 获取网页内容
4. 输出保存到 pages