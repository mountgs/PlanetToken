# Session Progress Log

## 2026-09-18 Hide model mapping from ordinary users

- 普通用户只能看到请求模型；用户接口不返回 `is_model_mapped` / `upstream_model_name`；管理员全局日志保留。
- OpenAI 形态响应和错误 JSON 的结构化 `model` 回写为请求模型（含 chat/completions、Responses/Codex、playground、embedding/image/audio/realtime/task）。Claude/Gemini 原生回包不改；错误 `message` 不做字符串替换。
- 前台按 `isAdminView` 再挡一层：用量徽章、详情「模型映射」、任务「实际模型」仅管理员全局视图渲染。
- 验证：`go test ./common ./relay/helper ./model` PASS；`go test ./controller -run TestTaskLogDTOHidesMappedModelFromUsers` PASS；web 容器 `bun test` 映射单测 3 pass。
- WSL：已用现有 `planettoken-image-test` 数据卷重建 `new-api-dev:local`。浏览器验收未完成：Docker/WSL 容器反复秒级重启，库中无带映射消费日志，且只有 Root 账号 `localhost`、没有普通用户。

## 2026-09-08 Responses Retry Implementation

- 完成 `feat-codex-008`：Codex Responses 在语义输出前遇到 HTTP 200 SSE `response.failed` / `response.error` / `error` capacity、限流或暂时性失败时，执行 1 次初始请求 + 5 次同渠道重试。
- 同渠道退避为 `0.5s / 1s / 2s / 4s / 8s`，优先使用合法 `Retry-After`；超过 60 秒请求级预算时不再开始新尝试。
- 同渠道耗尽后按现有 `RetryTimes` 切换渠道，并在请求内排除失败渠道；渠道+模型冷却默认 30 秒，Redis 共享且本地缓存兜底。
- capacity/限流等标记为同渠道可重试的瞬时错误只进入临时冷却，不触发永久自动禁用。
- 同渠道重试保持首次选中的多 Key 凭据；失败时清理旧亲和，最终成功才绑定新渠道。
- 流式前导生命周期事件最多缓冲 1 MiB；首个文本、工具、图片等语义输出后禁止透明重放，瞬时失败规范化为单个 `server_error` 终止事件。
- 失败 attempt 不结算，最终成功只进入一次原有结算路径；失败链写入日志 `other.admin_info.responses_retry`。
- 新增后台参数：`SameChannelRetryTimes`、`ResponsesRetryMaxDurationSeconds`、`ResponsesChannelCooldownSeconds`，后端与前端均限制合法范围。
- 验证通过：Codex/service/controller 定向测试、`relaykit` 全量测试、WSL `/mnt/d` 路径调用 Go 的定向测试、前端格式检查和 `pnpm build`。
- 已知基线：`go test ./service` 的渠道亲和用量缓存测试存在共享状态污染，单独复跑通过；`go test ./controller` 在 Windows 上因既有 SQLite 临时文件句柄未释放而失败/超时；前端全量 typecheck 有两个无关测试文件的既有类型错误。
- WSL Ubuntu 可用，但没有 Linux Go；Docker 需要 sudo 密码，因此本轮未完成原生 Linux 二进制测试。

## 2026-08-12 Codex Image Streaming Implementation

- Implemented explicit `stream:true` routing while preserving Codex upstream SSE aggregation for missing/false `stream`.
- Added `partial_images` validation for JSON and multipart requests (`0..3`).
- Added a Codex-specific complete SSE frame decoder with `event`, multiline `data`, comments, CRLF, `[DONE]`, EOF-before-close delivery, and a 128 MiB limit.
- Added Images partial/completed/error mapping, immediate flush, URL data URLs, independent heartbeat configuration, disconnect cancellation, retry output markers, and completed-count billing settlement.
- Added focused tests for request bounds, frame parsing, event mapping, sanitization, disconnect closure, heartbeat classification, pricing count, and env validation.
- Focused backend tests and formatting checks pass. Production proxy-chain validation remains pending.

## Current State

**Last Updated:** 2026-09-18
**Active Epic:** feat-codex-008 — Responses 输出前瞬时失败重试与渠道故障转移
**Phase:** **feat-codex-008 已完成；Image 006 仍等待生产代理链路验收；普通用户隐藏模型映射代码已落地，WSL 浏览器验收未完成**
**Commercial models:** `gpt-5.4`、`gpt-5.5`（仅对外售卖此二档）  
**Server:** `{SERVER_HOST}`（具体主机勿再写入公开仓库；本地私密记录维护）

## Active Work

- `feat-codex-image-stream-007`：为 Image 2 增加白名单、脱敏的上游错误透传。
- 回归测试已转绿：结构化 `response.failed` 字段透传、敏感信息脱敏、`response.incomplete(content_filter)` 分类和 completed 无图拒绝文本均已覆盖。
- 范围边界：不修改计费，不新增同渠道重试，不返回完整上游 payload 或内部信息。
- `feat-codex-image-stream-006` 阻塞于生产 Nginx/CDN 验收，不与当前代码修改并行。
- `init.sh` 在当前 Windows Bash 环境因 PATH 未配置便携 Go 而阻断；PowerShell 定向 Go 测试已通过。

## Completed

- [x] 移植 `0bc23d4`：Codex Plus 渠道支持 `gpt-image-2` 生成/编辑，并暴露标准 Images API
- [x] `go test ./relay/channel/codex ./relay/constant` PASS
- [x] Codex 渠道创建，OAuth 可用
- [x] 渠道测试：`gpt-5.4`、`gpt-5.5` 请求通过
- [x] **模型定价**：`gpt-5.4`、`gpt-5.5` 已在「分组与模型定价」配置
- [x] `verify-codex-relay.sh --phase infra` 部分 PASS（`/api/status`、无 token `/v1/models` → 401）
- [x] 生产 harness 文档与脚本（`codex-relay-production.md`、`verify-codex-relay.sh`）

## In Progress（商用下一步）

### Step A — 补全定价与渠道模型（feat-codex-002 剩余）

- [ ] 确认已配置 `*-openai-compact`（或 `gpt-5.4-openai-compact` / `gpt-5.5-openai-compact`）
- [ ] 渠道模型仅勾选：`gpt-5.4`、`gpt-5.5`、上述 compact 变体（共 4 项）
- [ ] **渠道亲和** 开启，`codex cli trace` 规则存在
- [ ] **chat→responses** 保持关闭
- [ ] 令牌分组与渠道分组一致（如 `default`）
- [ ] **自用模式** 关闭（商用）

### Step B — 用户令牌（feat-codex-003）

- [ ] 创建商用测试用户 + `sk-` 令牌
- [ ] 额度 > 0；若启用模型限制，仅允许 5.4 / 5.5 / compact 四 ID

### Step C — 自动化 E2E（feat-codex-004）

```bash
export RELAY_URL={BASE_URL}
export RELAY_TOKEN=sk-你的令牌

./scripts/verify-codex-relay.sh --url "$RELAY_URL" --token "$RELAY_TOKEN" --model gpt-5.4 --phase all
./scripts/verify-codex-relay.sh --url "$RELAY_URL" --token "$RELAY_TOKEN" --model gpt-5.5 --phase all
```

### Step D — 客户端验收（feat-codex-005）

- [ ] Codex CLI：`base_url = {BASE_URL}/v1`（线上 `/openai` 未发版，暂用 `/v1`）
- [ ] `model = "gpt-5.5"` 或 `gpt-5.4`，`wire_api = "responses"`
- [ ] CL1–CL4：对话、流式、管理端日志

## Verification Evidence

| Check | Result |
|-------|--------|
| 渠道测试 gpt-5.4 / gpt-5.5 | PASS（用户确认） |
| 模型定价 gpt-5.4 / gpt-5.5 | PASS（用户确认） |
| `verify-codex-relay.sh --phase infra` | PARTIAL（2026-07-10；`/openai/models` 无 auth 返回 200，疑 nginx SPA，客户端用 `/v1`） |
| `verify-codex-relay.sh --phase all` | 待执行（需 sk-） |

## Known Issues

- WSL 可启动，但未安装 Linux Go；Docker 需要 sudo 密码。Windows Go 可从 WSL 挂载路径运行测试。
- `go test ./...` 的根包可能依赖前端构建产物；`service` 渠道亲和缓存用例仅在全量运行时受共享状态影响，单独复跑 PASS。
- `controller` 全包测试在 Windows 上存在既有 SQLite 临时文件句柄未释放；前端全量 typecheck 有两个无关测试文件的既有类型错误。
- `/openai/*` 路径线上可能未生效（返回 SPA HTML）；Codex CLI 暂用 `{BASE}/v1`
- 发版含 `router/relay-router.go` OpenAI 别名后，可改 `base_url = .../openai`

## 安全提醒

- OAuth JSON 与 sk- 勿提交 Git

## Migration Note

**2026-07-15:** 生产配置/文档/脚本/harness 资产已从 `/home/mountgs/new-api` 迁移到 `/home/mountgs/PlanetToken`（fork `mountgs/PlanetToken`）。应用层 Codex 代码补丁未自动迁移（需对照最新 upstream 再合）。
