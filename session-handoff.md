# Session Handoff

## 2026-08-13 Implementation Handoff

- Current active feature is `feat-codex-image-stream-006`; `001..005` have local Go test evidence and are done.
- Code now includes the complete phase-1 Codex Images streaming path in `relay/channel/codex/image_sse.go` plus focused tests.
- Focused backend tests pass. Keep `006` and the epic incomplete until production billing and proxy-chain validation are complete.

## Current Objective

- **Goal:** Codex Plus `gpt-image-2` 流式 Images API 与保活
- **Status:** **feat-codex-image-stream-006 实施中**；等待生产凭据
- **Epic:** feat-codex-image-stream-epic

## 已排队但未激活

- **Epic:** `feat-codex-image-stream-epic`
- **契约:** `docs/requirements/codex-image-streaming.md`
- **状态:** `001..005` 已完成本地验证；`006` 等待生产验收。
- **入口:** 配置固定 `ModelPrice`，执行 n=1/n>1 扣费验证和复杂生图 SSE/心跳 E2E，并核对 Nginx 日志。

## 下次会话：从这里开始

1. 部署包含 `feat-codex-007` 的构建
2. 在 Codex Plus 渠道启用 `gpt-image-2`，并配置模型倍率/分组
3. 用测试 `sk-` 验证 `/v1/images/generations` 与 `/v1/images/edits`
4. 将线上请求日志和结果补充到 `feature_list.json`

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
