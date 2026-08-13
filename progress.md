# Session Progress Log

## 2026-08-12 Codex Image Streaming Implementation

- Implemented explicit `stream:true` routing while preserving Codex upstream SSE aggregation for missing/false `stream`.
- Added `partial_images` validation for JSON and multipart requests (`0..3`).
- Added a Codex-specific complete SSE frame decoder with `event`, multiline `data`, comments, CRLF, `[DONE]`, EOF-before-close delivery, and a 128 MiB limit.
- Added Images partial/completed/error mapping, immediate flush, URL data URLs, independent heartbeat configuration, disconnect cancellation, retry output markers, and completed-count billing settlement.
- Added focused tests for request bounds, frame parsing, event mapping, sanitization, disconnect closure, heartbeat classification, pricing count, and env validation.
- Focused backend tests and formatting checks pass. Production proxy-chain validation remains pending.

## Current State

**Last Updated:** 2026-08-13
**Active Epic:** feat-codex-epic — Codex CLI 生产商用接入  
**Phase:** **feat-codex-image-stream-001..005 本地验证完成；006 等待生产计费与复杂生图 E2E**
**Commercial models:** `gpt-5.4`、`gpt-5.5`（仅对外售卖此二档）  
**Server:** `{SERVER_HOST}`（具体主机勿再写入公开仓库；本地私密记录维护）

## Queued Work（未激活）

- `feat-codex-image-stream-epic`：Codex Image 2 显式 `stream:true`、Images SSE 事件转换、10 秒独立心跳、断连取消、重试边界与按次计费保护。
- 需求契约：`docs/requirements/codex-image-streaming.md`。
- 2026-08-13 已完成 `001..005` 实现和本地 Go 验证；当前唯一进行中功能为 `006`，等待生产凭据与 Nginx 日志。
- 实施顺序：`001 请求契约` → `002 frame 处理器` → `003 事件输出` + `004 心跳配置` → `005 断连/重试` → `006 计费/E2E`。

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

- 当前 Windows 环境无法启动 WSL，`./init.sh` 报 `CreateInstance/E_ACCESSDENIED`
- `go test ./...` 的根包缺少 `web/classic/dist` 构建产物；`service` 渠道亲和缓存用例仅在全量运行时受共享状态影响，单独复跑 PASS
- `/openai/*` 路径线上可能未生效（返回 SPA HTML）；Codex CLI 暂用 `{BASE}/v1`
- 发版含 `router/relay-router.go` OpenAI 别名后，可改 `base_url = .../openai`

## 安全提醒

- OAuth JSON 与 sk- 勿提交 Git

## Migration Note

**2026-07-15:** 生产配置/文档/脚本/harness 资产已从 `/home/mountgs/new-api` 迁移到 `/home/mountgs/PlanetToken`（fork `mountgs/PlanetToken`）。应用层 Codex 代码补丁未自动迁移（需对照最新 upstream 再合）。
