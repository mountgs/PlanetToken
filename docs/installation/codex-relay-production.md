# Codex CLI 生产接入与校验规范

面向**管理员/运维**：在中转站（new-api）上配置 Plus OAuth Codex 渠道，并为终端用户提供 Codex CLI 接入。  
终端用户配置见 [中转站客户端接入指南](./relay-client-integration.md)。

> Harness 特性 ID：`feat-codex-001` … `feat-codex-006`（见仓库根目录 `feature_list.json`）

---

## 1. 架构与边界

| 层级 | 凭据 | 协议 |
|------|------|------|
| 终端用户 | `sk-` 令牌 | `POST /openai/responses` 或 `POST /v1/responses`（Codex CLI `wire_api = responses`） |
| 上游渠道 | Plus OAuth JSON | `https://chatgpt.com/backend-api/codex/responses` |

**不支持（本方案范围外）：**

- ChatGPT 桌面/Web 内置 Codex 应用
- 默认走 `/v1/chat/completions` 的 IDE（Cursor/Cline 等），除非单独开启 chat→responses 策略

---

## 2. 前置条件

| 项 | 要求 |
|----|------|
| 中转站 | `./scripts/verify-deployment.sh` 全部 PASS |
| 公网访问 | HTTP 80 或 HTTPS 443 可达；流式需 nginx `proxy_buffering off` |
| Plus 账号 | 有效 ChatGPT Plus 订阅，可导出 Codex OAuth JSON |
| OAuth JSON | 必含 `access_token`、`account_id`、`refresh_token` |

---

## 3. 管理员配置清单

### 3.1 创建 Codex 渠道（feat-codex-001）

1. **渠道** → **添加渠道**
2. 类型：**ChatGPT Subscription (Codex)**
3. Base URL：留空（默认 `https://chatgpt.com`）
4. 密钥：粘贴 OAuth JSON（单行或格式化均可）
5. 模型：勾选目标 Codex 模型（如 `gpt-5.3-codex`）
6. 分组：`default`（或专用分组如 `codex`）
7. 权重 / 优先级：按账号池规模设置（多账号时权重均分）
8. **禁止**使用多 Key 批量模式（Codex 渠道不支持）

**OAuth JSON 最低字段：**

```json
{
  "access_token": "...",
  "account_id": "...",
  "refresh_token": "...",
  "type": "codex"
}
```

### 3.2 渠道验收（feat-codex-001）

| # | 检查项 | 通过标准 |
|---|--------|----------|
| C1 | 渠道测试 | 管理后台「测试」成功（Codex 自动使用流式 `/v1/responses`） |
| C2 | 刷新凭证 | 「刷新凭证」成功，`expires_at` 更新 |
| C3 | Codex 用量 | 「Codex 账户和用量」返回 200，可见 5h/weekly 窗口 |
| C4 | refresh_token | JSON 含非空 `refresh_token`（否则自动刷新失败） |

### 3.3 系统策略（feat-codex-002）

| # | 设置项 | 要求 |
|---|--------|------|
| S1 | 渠道亲和 | **开启**；保留默认规则 `codex cli trace` |
| S2 | 模型倍率 | 为 `gpt-5*-codex*` 配置计费倍率 |
| S3 | 分组映射 | 用户/令牌分组与渠道分组一致 |
| S4 | chat→responses | **保持关闭**（Codex CLI 原生走 responses，无需转换） |

### 3.4 用户与令牌（feat-codex-003）

| # | 检查项 | 通过标准 |
|---|--------|----------|
| U1 | 测试用户 | 至少 1 个非管理员测试账号 |
| U2 | 令牌 | 创建 `sk-`，分组含 Codex 渠道所在组 |
| U3 | 额度 | 设置剩余额度 > 0 或无限额策略 |
| U4 | 模型列表 | 见自动化校验 §4.2 `relay` 阶段 |

---

## 4. 自动化校验

### 4.1 脚本

```bash
chmod +x scripts/verify-codex-relay.sh

# 仅基础设施（无需令牌）
./scripts/verify-codex-relay.sh --url http://127.0.0.1:3000 --phase infra

# 公网 + 完整 Codex E2E（需用户 sk- 与可用模型）
./scripts/verify-codex-relay.sh \
  --url {BASE_URL} \
  --token sk-xxxxxxxx \
  --model gpt-5.3-codex \
  --phase all
```

环境变量等价：`RELAY_URL`、`RELAY_TOKEN`、`RELAY_MODEL`。

### 4.2 阶段与通过标准

| 阶段 | 命令 | 通过标准 |
|------|------|----------|
| `infra` | `--phase infra` | `/api/status` → `success:true`；无 token 访问 `/v1/models` → 401 |
| `auth` | `--phase auth` | 假 token → 401；真 token → 200 |
| `relay` | `--phase relay` | `/v1/models` 含目标 model id |
| `codex` | `--phase codex` | `POST /v1/responses` 同步 200 且有 `id`；流式 200 且含 `data:` SSE 行 |
| `all` | 默认 | 以上全部 PASS |

### 4.3 代码级回归（开发/发版前）

```bash
./init.sh
```

必须：`go test ./...` 与 `web/default` typecheck 均无错误。

---

## 5. 客户端验收（feat-codex-005）

在**测试机**（Windows / macOS / Linux 各至少 1 台）执行：

### 5.1 Codex CLI 配置

`~/.codex/auth.json`：

```json
{ "OPENAI_API_KEY": "sk-用户令牌" }
```

`~/.codex/config.toml`：

```toml
model_provider = "relay"
model = "gpt-5.3-codex"
model_reasoning_effort = "high"
disable_response_storage = true
preferred_auth_method = "apikey"

[model_providers.relay]
name = "relay"
base_url = "http://你的域名或IP/openai"
wire_api = "responses"
requires_openai_auth = true
```

> Codex 推荐 `base_url` 为 `{BASE}/openai`（与 CC Switch 一键导入一致）；`/v1` 路径同样可用。

### 5.2 客户端检查项

| # | 检查项 | 通过标准 |
|---|--------|----------|
| CL1 | 配置生效 | 修改后**重启终端** |
| CL2 | 简单对话 | `codex` 发起对话，有正常回复 |
| CL3 | 流式输出 | 终端可见增量输出，无中途断流 |
| CL4 | 管理端日志 | 日志页可见对应请求，渠道为 Codex，状态成功 |
| CL5 | CC Switch（可选） | 一键导入后 Codex CLI 同样可用 |

---

## 6. 运维监控（feat-codex-006）

| # | 项 | 频率 | 告警条件 |
|---|-----|------|----------|
| M1 | OAuth 自动刷新 | 每 10 分钟（内置任务） | 日志出现 `codex credential auto-refresh ... refresh failed` |
| M2 | Plus 用量 | 每日 | 5h 或 weekly 窗口 < 10% |
| M3 | 渠道错误率 | 每日 | 429/403 占比突增 → 降权或禁用渠道 |
| M4 | 用户额度 | 实时 | 余额不足拒绝服务（预期行为） |
| M5 | 备份 | 每日 | `./scripts/backup-db.sh` 成功 |

---

## 7. 故障排查

| 现象 | 可能原因 | 处理 |
|------|----------|------|
| `Invalid token` | sk- 错误或过期 | 重新创建令牌 |
| `/v1/responses` 或 `/openai/responses` 404 | Base URL 路径错误 | 客户端改为 `{BASE}/openai`（Codex）或 `{BASE}/v1` |
| 401 upstream | OAuth 过期且无 refresh_token | 重新导入 OAuth JSON 并刷新 |
| 模型不存在 | 渠道未勾选该模型或分组不匹配 | 检查渠道模型与令牌分组 |
| 流式中断 | nginx 缓冲未关 | `proxy_buffering off; proxy_read_timeout 600s` |
| 5h 限额耗尽 | 单 Plus 用户过多 | 增加 Plus 渠道或限制并发用户 |

---

## 8. 生产 Definition of Done

在 `feature_list.json` 中将 `feat-codex-004`、`feat-codex-005` 标为 `done` 前，须同时满足：

- [ ] `verify-deployment.sh` PASS（服务器本地或 `--url` 公网）
- [ ] `verify-codex-relay.sh --phase all` PASS（含真实 sk- 与模型）
- [ ] 管理后台渠道测试 + OAuth 刷新 + 用量查询 PASS
- [ ] 至少 1 台客户端 Codex CLI 人工验收（CL1–CL4）
- [ ] `progress.md` 与 `feature_list.json` evidence 字段已更新

---

## 9. 下次执行清单（按序）

下次会话从 **`feat-codex-001`** 开始，建议严格按下列顺序执行；每步完成后在 `feature_list.json` 更新 status / evidence。

### 准备材料

- [ ] Plus OAuth JSON（含 `access_token`、`account_id`、`refresh_token`）
- [ ] 中转站管理员账号
- [ ] 计划使用的 Codex 模型 ID（如 `gpt-5.3-codex`）

### Step 1 — feat-codex-001：渠道

1. 登录管理后台 → **渠道** → 添加 **ChatGPT Subscription (Codex)**
2. 粘贴 OAuth JSON，勾选模型，分组 `default`，保存并启用
3. 完成人工验收 **C1–C4**（§3.2 表格）

### Step 2 — feat-codex-002：策略

1. **设置** → 确认渠道亲和开启，`codex cli trace` 规则存在
2. 配置 Codex 模型倍率；确认 chat→responses **关闭**

### Step 3 — feat-codex-003：令牌

1. 创建测试用户 → **令牌** 页生成 `sk-`
2. 确认分组与额度

### Step 4 — feat-codex-004：自动化校验

在服务器或本机（能访问中转站）执行：

```bash
cd /www/wwwroot/new-api   # 或本仓库根目录

chmod +x scripts/verify-codex-relay.sh

# 基础设施
./scripts/verify-codex-relay.sh --url {BASE_URL} --phase infra

# 完整 E2E（替换 sk- 与 model）
export RELAY_URL={BASE_URL}
export RELAY_TOKEN=sk-你的测试令牌
export RELAY_MODEL=gpt-5.3-codex

./scripts/verify-codex-relay.sh --phase all
```

全部 `[PASS]` 后，将 `feat-codex-003`、`feat-codex-004` 标为 `done` 并写入 evidence。

### Step 5 — feat-codex-005：客户端

1. 测试机安装 Codex CLI
2. 按 [relay-client-integration.md](./relay-client-integration.md) §Codex 或 CC Switch 配置
3. 完成 **CL1–CL4**（§5.2）

### Step 6 — feat-codex-006：运维

1. 确认 OAuth 自动刷新日志正常
2. 查看 Codex 用量；执行一次 `backup-db.sh`
3. 更新 `progress.md`

### 发版前（可选）

```bash
./init.sh   # go test + frontend typecheck
```

---

## 相关文档

- [中转站客户端接入指南](./relay-client-integration.md)
- [生产环境部署指南](./production.md)
- [运维操作手册](./operations.md)
- [服务器上线步骤](./server-go-live.md)
