# Session Handoff

## Current Objective

- **Goal:** Codex CLI 生产接入（文档 + harness）
- **Status:** **feat-codex-001 手动配置进行中**（管理后台）；C1–C4 验收待完成
- **Epic:** feat-codex-epic

## 下次会话：从这里开始

1. 打开 **`docs/installation/codex-relay-production.md` §9**
2. 准备 Plus OAuth JSON + 测试 `sk-`
3. 按 Step 1 → 6 执行，同步更新 `feature_list.json`

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
