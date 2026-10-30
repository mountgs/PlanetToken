# Session Handoff

## Production release completed — 2026-10-30

- Customer UI is deployed: source `dd3ca371675a03cf4b862f2b034582a094ed7c58`, version `customer-ui-20261030-dd3ca3716`, digest `sha256:dfcfd30e281e81190c0abc9e2691d7dd2b092be27d39333deedc07857946634c`. Exact source is public, embedded in the customer sidebar link and checked out at the server source symlink. Actions run 41041503869 passed Linux frontend/affected backend checks and published linux/amd64. Local consolidated frontend run passed 118 tests/11 files.
- Backup and isolated restored-data two-startup rehearsal passed, including full user/token/channel row fingerprints. Only app container replaced; existing PostgreSQL/Redis/config retained. Final counts preserved: 157 users, 108 tokens, 6 channels, 19,807 logs. App healthy/no restarts after 60 seconds; existing API key, unauthenticated protection, Contact API, frontend assets and public reverse proxy checks passed.
- Protected server archive `/www/wwwroot/new-api/backups/release-20261030-dd3ca3716`; rollback via its `rollback.sh`, preserving live database. Ignored local deployment assets in `.local-tests/deploy-20261030/`; final off-server bundle transfer completed and 17 exported files checksum-verified in `final-backup/`. Independent workstation request also confirms public deployed version. Do not commit those files.
- Server currently exposes its existing HTTP IP site and has no domain/HTTPS configuration; do not claim TLS is deployed. No payment or paid-generation E2E performed. Full Windows Go suite remains non-green and new browser visual QA unavailable; feature stays in-progress under repository gates. Earlier release preparation/no-push notes below are historical.
- Contact configuration lives at root administrator `/system-settings/site/contact`, with hosted QR image URLs and Markdown preview. Real contact details remain unconfigured. Customer Online Chat uses the original `/playground`.

## Active release — 2026-10-30

- User authorized updating the existing remote server; this supersedes older no-deployment notes below. Recovered working SSH connection from prior deployment assets, no new credentials needed. Use PlanetToken-customer-ui and codex/customer-ui-rightapi only.
- Release workflow verifies Linux frontend and affected backend behavior, publishes linux/amd64 to GHCR and embeds exact source commit URL. Local typecheck (1,431 files), changed-file lint (54 TS files) and whitespace check pass. Full Windows Go suite remains non-green; consolidated frontend rerun has not completed.
- Before cutover: publish source/image, protect database/configuration/current image, restore production PostgreSQL snapshot in an isolated internal network, verify first and second startup, then pin digest and replace only app container. Keep existing env and data. Production update still pending at this commit.
- Deployment scripts, private connection data and evidence belong outside tracked source. Inspect disk capacity; preserve rollback and production data. Complete release evidence and source sync after health/authentication verification.

## 2026-10-30 Online Chat now reuses Playground

- User explicitly chose the original Playground for customer Online Chat. This supersedes the earlier decision to hide Playground and use external chat presets for that customer entry.
- Customer navigation now links Online Chat directly to `/playground`, independent of external preset configuration. Restored the original Playground route guard (administrator module toggle only; authenticated parent retained), removing the customer-role redirect introduced by the redesign. Existing `chat.playground` configuration controls entry visibility and route availability; administrator navigation and existing backend `/pg/chat/completions` behavior are unchanged. Reused the complete Playground without duplicating chat logic.
- Regression-first verification: before implementation 3 cases failed (destination, disabled-module entry and customer route access). After implementation, all 23 cases in customer-navigation/sidebar-config/Playground route-access pass. Typecheck passes in 1,431 files, changed-file lint 54 TS files, protected-header-safe formatting, `git diff --check` and production build pass. No backend changes this follow-up; previous full Go failures remain documented above/below.
- Restarted fixture preview with latest build at `http://127.0.0.1:4191/playground` and added synthetic model choices. HTTP verifies page and model fixtures; no new browser visual QA and no actual model generation: the fixture server does not connect upstream AI. Real deployment uses the existing session-authenticated Playground endpoint and billing. Evidence: `chat-before.log`, `chat-after.log`, `chat-typecheck.log`, `chat-build.log` in external customer-ui-verification directory. No commit/push/deployment.

## Current follow-up — 2026-10-30

- Continue only in `D:/WorkSpace/PlanetToken/PlanetToken-customer-ui`, branch `codex/customer-ui-rightapi`. Contact and pricing follow-up is implemented, checked and uncommitted. No push/deployment authorized or performed.
- User clarified contacts must be configurable by an administrator and support WeChat QR images. Implemented `/contact` and `/system-settings/site/contact`, persisted `Contact` option and public `/api/contact`; reuse existing root-only option updates, ETag handler, SettingsForm and sanitized RichContent. Multiple hosted HTTP(S) image URLs can be inserted and moved using their Markdown lines. No upload service. Blank configuration shows an empty state; no fabricated channels. Save failure retains drafts. Memoized defaults prevent post-save preview reset.
- Four catalog token-price columns, billing-method filter and full tier conditions/details now use selected key/group rates and display currency. Free and unavailable values remain distinct. Request/task prices stay beside billing method, not in token-price cells. Existing admin price presentation defaults remain intact.
- Latest checks: 77 distinct frontend cases across six suites pass (final pricing rerun 11/11, contact editor rerun 4/4); typecheck 1,430 files; lint of 54 changed TS files; protected-header-safe format, diff whitespace and production build pass. Contact controller test and full router test package pass. Full Go suite completed non-green: controller Windows audit.db cleanup failures and 90-second timeout; initial new public-route coverage omission fixed and router rerun passes. No new browser QA: CUA/node runtime initialization still errors with Windows error 3.
- Fixture server restarted at `http://127.0.0.1:4191/catalog`, synthetic tiered model plus empty contact configuration. Site settings can save local fixture contacts (in memory). Server script and evidence logs remain in `D:/WorkSpace/PlanetToken/customer-ui-verification`. Preview serves actual latest build, but HTTP verification does not replace visual inspection.
- Before release: resolve/accept full-suite verification limits, visually inspect customer contact and four-column/tiered pricing on desktop/mobile, configure real contact content/image hosting, publish exact modified AGPL source and configure `VITE_SOURCE_URL`. Feature stays `in-progress`; no database migration/dependency changes.

## Current work — 2026-10-29 customer UI

- User authorized implementation after the RightAPI interview. Feature: `feat-customer-ui-rightapi`. UI is implemented and ready for review; keep harness status `in-progress` because full required verification is not green.
- Workdir: `D:/WorkSpace/PlanetToken/PlanetToken-customer-ui`; branch `codex/customer-ui-rightapi`; base `fb45dff52`. Changes remain uncommitted. Do not switch to the older `PlanetToken` checkout.
- Customer-only warm neutral/terracotta theme, compact overview and simplified navigation. Admin theme/navigation stay original. Separate `/billing` page for recharge bills, including old customer `show_history` links. Hide referral, Playground, audit and Flow; profile/security/language live in the account menu.
- `/catalog` prices follow selected Key group; persist only Key ID per user. No keys use default. Auto groups respect permissions/limits, unknown groups show unavailable, model limits apply, billing cells reuse existing formulas. No backend/schema/dependency changes.
- Verification: 56 frontend tests (7 files) pass; final wallet rerun 8 pass. Typecheck 1,421 files, changed-file oxlint 39 TS files, protected-header-safe format, diff whitespace and production build pass. Focused Go model/controller log-visibility tests pass. Browser QA uses synthetic fixtures; no production writes/payments.
- Full checks remain non-green: `init.sh` initially hit C: disk/pagefile pressure; serialized Go run with D: temporary storage reported controller SQLite cleanup failures and controller/relay timeouts. Other packages including service completed. Earlier baseline reproduced SQLite cleanup failures; do not claim every timeout is proven pre-existing. Full frontend lint reports errors in untouched areas (utils, confirm-dialog, system-settings); changed files pass.
- Fixture preview: `http://127.0.0.1:4191/catalog`; script `D:/WorkSpace/PlanetToken/customer-ui-verification/preview.cjs` outside repo. Restart with Node if necessary. Screenshots and Go/build logs are in that directory. `web/node_modules` links to existing dependencies; no lockfile change.
- Before deployment: publish exact modified Corresponding Source, then configure `VITE_SOURCE_URL` (build environment or `.env.production`) for that release. Attribution/AGPL source footer retained. Default fork URL does not yet contain this unpushed implementation. No push or deployment performed.
- Remaining acceptance: user visual review, staging with representative data (payment return/configured chat iframe or external site), and resolution or explicit acceptance of full-suite limitations before release. Payment/chat end-to-end and production rollout are unverified.

Historical handoffs follow.

## 2026-10-27 Upstream merge into isolated branch

- Baseline authorized by user: GitHub committed branch db887f185 (origin/codex/responses-retry-failover). Uncommitted edits were explicitly discarded before this merge.
- Workspace: D:/WorkSpace/PlanetToken/PlanetToken-upstream-merge; branch codex/upstream-sync-20261027.
- Upstream target: 34003e9c2 (313 new commits). Original application workspace D:/WorkSpace/PlanetToken/PlanetToken remains clean at db887f185.
- Preserved Codex Image 2 generation/editing, SSE/heartbeats/error passthrough, configurable main model, Responses same-channel retries/failover/cooldown/affinity, and ordinary-user model-mapping hiding.
- Adapted the three local retry options to upstream request-policy API and Sessions and retries UI. Retained upstream usage accumulator and recorded usage before downstream writes; stopped retry after semantic output. New response_model log/export data is admin-only under the local visibility requirement.
- Upstream additions include request policies and decision records, Responses WebSocket, scoped access tokens, usage log export and response-model observations, new billing/cache accounting, providers/plugins/UI/security improvements.
- Actual verification: Go 1.26.10, Bun 1.4.4; frontend frozen install, typecheck (1,407 files), changed-file oxlint, production build pass. Focused frontend tests: 41 pass plus routing suite 30 pass; initial worker startup timeouts resolved by serial reruns. GOWORK=off go build ./... in relaykit and tokenkit both pass. go test ./relay/channel/codex ./relay/channel/openai pass; relay and service package tests pass. Focused model/controller tests (log visibility, retry options, request policy SQLite, pin/output boundaries) pass.
- Root go build -buildvcs=false ./... passed. Ordinary VCS stamping is blocked by Windows Git ownership; relaykit/tokenkit independent builds also passed.
- Full go test ./... -timeout 4m ran after the real frontend build and failed: controller SQLite audit.db cleanup reported open-file handles repeatedly and the package timed out at four minutes; relay/channel graceful-GOAWAY test reported Windows TCP reset/abort. Other packages completed successfully. The unmodified upstream 34003e9c2 archive also reproduced the GOAWAY failure with -count=20 (five failed iterations), so it is not introduced by this merge. Full-suite verification is not green.
- Focused TestRequestPolicyDatabaseMatrix passed on SQLite 3.50.4; MySQL and PostgreSQL subtests explicitly skipped because their DSNs are not configured.
- Upstream baseline comparison: unmodified 34003e9c2 also fails TestAuditDatabaseMatrix on Windows audit.db cleanup (both upgrade=false and upgrade=true), and reproduces the GOAWAY TCP reset. Neither failure requires the PlanetToken merge changes.
- MySQL/PostgreSQL test instances and upgrade migration matrix unavailable (no TEST_MYSQL_DSN/TEST_POSTGRES_DSN). Do not declare database compatibility or production rollout verified. No production deployment or push performed.


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
