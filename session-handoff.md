# Session Handoff

## Current Objective

- **Goal:** Codex Plus `gpt-image-2` 标准 Images API 接入
- **Status:** **feat-codex-007 代码完成**；等待部署环境做生成/编辑 E2E
- **Epic:** feat-codex-epic

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
