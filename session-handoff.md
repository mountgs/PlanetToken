# Session Handoff

## 2026-09-18 Hide model mapping from ordinary users

- 代码已落地：用户日志/任务接口剥映射字段；OpenAI 兼容出站 JSON 的 `model` 回写请求模型；前台管理员全局视图才展示映射。
- 自动化证据：`go test ./common ./relay/helper ./model`；controller `TestTaskLogDTOHidesMappedModelFromUsers`；前端 `web/src/features/usage-logs/lib/__tests__/model-mapping.test.ts` 3 pass。
- 下次继续：等 WSL Docker 稳定后，用普通用户打开 `http://localhost:5173` 走用量/任务页；若库中无映射日志，先配一条模型映射再打一条请求。现场 `/v1/chat/completions` 与 `/v1/responses` 能通则核对回包 `model` 等于请求名。

## 2026-09-08 Responses Retry Handoff

- `feat-codex-008` 已完成。实现集中在 `service/responses_retry.go`、`controller/relay.go` 和 `relay/channel/codex/responses_retry.go`，未复制 sub2api 的业务层结构，降低后续合并 new-api 上游更新的冲突面。
- 默认行为：每渠道 1+5 次，指数退避 `0.5/1/2/4/8s`，合法 `Retry-After` 优先，总预算 60 秒；同渠道耗尽后仅在 `RetryTimes > 0` 时切换渠道。
- HTTP 200 SSE capacity 在语义输出前可重放；文本、工具、图片等真实输出后不重放，只发送脱敏、规范化的单个 `server_error` 终止事件。
- 失败渠道会在当前请求排除，并按渠道+模型冷却 30 秒；Redis 可用时跨实例共享，Redis 异常时本地缓存继续生效。
- 同渠道循环不会重新选多 Key；重试失败清理 affinity，最终成功才记录当前渠道 affinity。计费仍只由最终成功的原有 Responses 路径执行一次。
- 管理后台的 Request retry 区域已增加三个 Responses 参数，API 侧也执行相同范围校验。
- 已通过：Codex/service/controller 定向测试、`relaykit` 全量测试、前端生产构建、WSL 挂载路径定向测试。
- 基线限制：`service` 全包有共享缓存污染；`controller` 全包在 Windows 上受 SQLite 文件句柄清理问题影响；原生 Linux Go 测试因 WSL Docker 需要 sudo 密码未执行。

### 部署后验收

1. 将 `RetryTimes` 设为至少 `1`，保持 `SameChannelRetryTimes=5`、预算 `60`、冷却 `30`。
2. 用可稳定模拟 capacity 的测试渠道发起流式和非流式 `/v1/responses` 请求，确认首渠道产生 6 个 attempt 后切换且只产生一条最终消费日志。
3. 检查日志 `other.admin_info.responses_retry.attempts`、最终渠道 ID、多 Key index 和错误码。
4. 多实例环境检查 Redis 中 `new-api:responses_channel_cooldown:v1:*` 的 TTL，并确认另一实例避开冷却渠道。

## 2026-08-13 Implementation Handoff

- Current active feature is `feat-codex-image-stream-007`; `001..005` are done and `006` is blocked on production proxy-chain validation.
- Code now includes the complete phase-1 Codex Images streaming path in `relay/channel/codex/image_sse.go` plus focused tests.
- Regression tests in `relay/channel/codex/image_sse_test.go` pass for structured upstream field passthrough, sanitization, `response.incomplete(content_filter)` classification, and completed-without-image refusal text.
- Scope: preserve only sanitized `message/type/code/param`; do not change billing or add same-channel retries.

## Current Objective

- **Goal:** Codex Plus `gpt-image-2` 上游错误可解释透传
- **Status:** **feat-codex-image-stream-007 实施中**；回归测试已建立
- **Epic:** feat-codex-image-stream-epic

## 当前实现入口

- **Epic:** `feat-codex-image-stream-epic`
- **契约:** `docs/requirements/codex-image-streaming.md`
- **状态:** `007` 进行中；`006` 等待生产验收。
- **入口:** 在 `relay/channel/codex/image_sse.go` 增加结构化错误模型、白名单解析、脱敏和状态分类，使现有红灯测试通过。

## 下次会话：从这里开始

1. 实现 Image 2 结构化错误解析和脱敏 SSE 输出
2. 识别 `response.incomplete` 与无图拒绝文本
3. 运行 Codex/relay/controller 定向测试和 `init.sh`
4. 保持 `007` 未完成，直到全部自动化验收通过

## 文档地图

| 读者 | 文档 |
|------|------|
| **管理员/运维** | [codex-relay-production.md](./docs/installation/codex-relay-production.md) |
| **终端用户** | [relay-client-integration.md](./docs/installation/relay-client-integration.md) |
| **Harness 状态** | [feature_list.json](./feature_list.json) |
| **执行进度** | [progress.md](./progress.md) |

## 校验脚本（下次带 token 跑）

```bash
chmod +x scripts/verify-codex-relay.sh
./scripts/verify-codex-relay.sh --url {BASE_URL} --phase infra
./scripts/verify-codex-relay.sh --url {BASE_URL} --token sk-xxx --model gpt-5.3-codex --phase all
```

## Blockers（下次解除）

- [ ] Plus OAuth JSON（含 `refresh_token`）
- [ ] 测试用 `sk-` 令牌

## 本会话已完成

- [x] 生产规范文档 + 执行清单
- [x] verify-codex-relay.sh
- [x] feature_list feat-codex-001…006
- [x] 文档交叉链接
