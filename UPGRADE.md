# 升级指南

本文档提供了 YoClaw 各版本之间的升级指南，请根据你的当前版本查找对应的升级说明。

## v0.2.1 → v0.2.1+

### Workspace 目录结构调整

为了避免大模型写入的文件与配置文件冲突，workspace 目录结构已调整。

#### 主要变化

- 配置文件（AGENTS.md、BOOTSTRAP.md、IDENTITY.md、SOUL.md、TOOLS.md、USER.md 等）从 workspace 根目录移至 `workspace/profile/` 子目录
- 如果存在 `BOOTSTRAP.lock` 文件，也需要移动到 `workspace/profile/` 目录下

#### 迁移步骤

**Linux/macOS 用户：**

```bash
# 假设你的 workspace 目录位于 ~/.yoClaw/workspace
cd ~/.yoClaw/workspace

# 创建 profile 子目录
mkdir -p profile

# 移动配置文件到 profile 目录
mv AGENTS.md profile/
mv BOOTSTRAP.md profile/
mv HEARTBEAT.md profile/
mv IDENTITY.md profile/
mv SOUL.md profile/
mv TOOLS.md profile/
mv USER.md profile/

# 如果存在 BOOTSTRAP.lock，也移动到 profile 目录
if [ -f BOOTSTRAP.lock ]; then
    mv BOOTSTRAP.lock profile/
fi
```

**Windows 用户：**

```powershell
# 假设你的 workspace 目录位于 C:\Users\YourName\.yoClaw\workspace
cd C:\Users\YourName\.yoClaw\workspace

# 创建 profile 子目录
New-Item -ItemType Directory -Path profile -Force

# 移动配置文件到 profile 目录
Move-Item AGENTS.md profile\
Move-Item BOOTSTRAP.md profile\
Move-Item HEARTBEAT.md profile\
Move-Item IDENTITY.md profile\
Move-Item SOUL.md profile\
Move-Item TOOLS.md profile\
Move-Item USER.md profile\

# 如果存在 BOOTSTRAP.lock，也移动到 profile 目录
if (Test-Path BOOTSTRAP.lock) {
    Move-Item BOOTSTRAP.lock profile\
}
```

#### 新的目录结构

```
~/.yoClaw/workspace/
├── profile/              # agent 配置档案
│   ├── AGENTS.md
│   ├── BOOTSTRAP.md
│   ├── BOOTSTRAP.lock    # 如果存在
│   ├── HEARTBEAT.md
│   ├── IDENTITY.md
│   ├── SOUL.md
│   ├── TOOLS.md
│   └── USER.md
├── sessions/             # 会话数据
├── tasks/                # 任务数据
├── memory/               # 记忆数据
└── *.md                  # 大模型写入的文件（不会与配置冲突）
```

#### 为什么需要这个改动？

- **避免文件冲突**：大模型通过工具写入的 md 文件可能与配置文件重名
- **防止误读**：大模型可能会读取用户写入的 md 文件，导致行为异常
- **防止覆盖**：工具可能会覆盖这些重要的配置文件
- **目录清晰**：配置文件和运行时文件分离，易于管理

---

## v0.1.0 → v0.2.0+

### 配置文件格式变更

配置文件格式已发生重大变化，需要手动更新配置文件。

#### 主要变化

- `channels` 配置从固定字段改为 `map[string]ChannelConfig`，支持动态配置多个 channel
- 每个 channel 配置需要指定 `type` 字段（如 `"web"`、`"feishu"`）
- channel 名称可以自定义（如 `"webTest"`、`"feishuProd"`）

#### 迁移步骤

**v0.1.0 配置（旧格式）：**

```json
{
  "channels": {
    "web": {
      "enabled": true,
      "agent": "default",
      "host_address": "localhost:8080",
      "token": "your-token"
    }
  }
}
```

**v0.2.0+ 配置（新格式）：**

```json
{
  "channels": {
    "webTest": {
      "type": "web",
      "enabled": true,
      "agent": "default",
      "host_address": "localhost:8080",
      "token": "your-token"
    },
    "feishuProd": {
      "type": "feishu",
      "enabled": true,
      "agent": "default",
      "app_id": "your-app-id",
      "app_secret": "your-app-secret"
    }
  }
}
```

#### 主要区别

1. **添加 `type` 字段**：每个 channel 必须指定类型（`"web"`、`"feishu"` 等）
2. **自定义 channel 名称**：可以使用任意名称（如 `"webTest"`、`"feishuProd"`）
3. **支持多个同类型 channel**：可以配置多个 web 或 feishu channel

---

## 跨版本升级

如果你从 v0.1.0 直接升级到 v0.2.1+，需要同时完成上述两个升级步骤：

1. 先按照 **v0.1.0 → v0.2.0+** 的说明更新配置文件格式
2. 再按照 **v0.2.1 → v0.2.1+** 的说明调整 workspace 目录结构

---

## 常见问题

### Q: 升级后我的数据会丢失吗？

A: 不会。升级只会调整配置文件格式和目录结构，不会删除任何数据。会话、任务、记忆等数据都会保留。

### Q: 如果我跳过某个版本直接升级会怎样？

A: 请按照"跨版本升级"部分的说明，依次完成所有必要的升级步骤。

### Q: 升级失败怎么办？

A: 如果升级过程中遇到问题，可以：
1. 备份当前的配置和数据
2. 检查错误信息
3. 在 [GitHub Issues](https://github.com/yockii/YoClaw/issues) 提交问题
