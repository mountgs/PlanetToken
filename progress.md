# Session Progress Log

## 2026-10-30 Customer UI production release completed

- Deployed source `dd3ca371675a03cf4b862f2b034582a094ed7c58`, version `customer-ui-20261030-dd3ca3716`, at 18:44:37 Asia/Shanghai. Running image is pinned to `ghcr.io/mountgs/planet-token@sha256:dfcfd30e281e81190c0abc9e2691d7dd2b092be27d39333deedc07857946634c`. Published source: https://github.com/mountgs/PlanetToken/tree/dd3ca371675a03cf4b862f2b034582a094ed7c58 . Source symlink on server points to this exact checkout.
- GitHub Actions https://github.com/mountgs/PlanetToken/actions/runs/41041503869 succeeded: Linux frontend typecheck/customer suites and affected router/controller/model checks, linux/amd64 build and GHCR publish. Local consolidated frontend rerun: 118 tests across 11 files passed; typecheck 1,431 files, changed-file lint 54 TS files and whitespace check passed.
- Protected configuration/data, database and old-image archives were created and checksummed. Isolated internal-network PostgreSQL restore passed two application startups, core row counts and full user/token/channel row fingerprints. Production cutover stopped only the app, took a final dump, updated only NEW_API_IMAGE and recreated only the app; PostgreSQL/Redis were retained.
- Final snapshot and immediate post-cutover counts: 157 users, 108 tokens, 6 channels, 19,807 logs. Existing API key `/v1/models` returns 200, unauthenticated request returns 401, Contact API returns 200, customer routes/main resources pass, and 60-second observation reports healthy with zero restarts. Final public reverse-proxy verification also passed for version, Contact, authentication and the actual lazy sidebar asset with exact source URL and upstream attribution.
- Existing public site uses an HTTP IP address; no domain/HTTPS is configured on this server. Reverse proxy was not changed. Configure TLS before commercial customer launch. No paid generation or payment-flow test was made during this release; browser visual QA remains limited as recorded below. Full Windows Go suite is still non-green, so feature status remains in-progress despite a successful deployment.
- Server archive: `/www/wwwroot/new-api/backups/release-20261030-dd3ca3716`; manual rollback: `sh /www/wwwroot/new-api/backups/release-20261030-dd3ca3716/rollback.sh`. It preserves the current database and saves a failed-release snapshot before restoring the old app/config. Old image is also archived and tagged `planet-token:rollback-20261030-pre-customer-ui`. The isolated rehearsal resources were removed; no production data or historical images were pruned. About 1.3 GiB free remained after release.
- Private deployment scripts/records and off-server backup are under ignored `.local-tests/deploy-20261030/`. Initial configuration/database backup copies and all 17 exported final-bundle files were locally checksum-verified; final backup is in its `final-backup/` directory. Workstation-to-public-site status independently confirms the deployed version. These assets include secrets and must never be committed. Earlier preparation/no-deployment notes below are historical.

## 2026-10-30 Customer UI production release preparation

- User explicitly authorized updating the existing remote server to the current image. This supersedes earlier no-push/no-deployment notes. Existing SSH key and deployment directory were recovered from prior records; credentials and host details remain outside Git.
- Release includes the customer UI, complete group-aware pricing, configurable Contact page, and original Playground entry described below. No database schema or dependency changes. Preserve production configuration, accounts, balances, API keys, channels and logs.
- Local release verification: typecheck passes for 1,431 files; changed-file lint (54 TS files) and whitespace check pass. Prior focused frontend/backend tests and build pass as recorded below. Consolidated frontend rerun has not completed; full Windows Go suite remains non-green as documented.
- Added a branch-specific GHCR workflow with Linux customer tests, typecheck and affected backend checks before publishing linux/amd64. Docker build accepts VITE_SOURCE_URL pointing to the exact published commit. Deployment requires immutable image identity, protected configuration/database backup, isolated PostgreSQL restore rehearsal and health/authentication checks before cutover.
- Production deployment is not yet complete. Server storage is limited; retain rollback image and backups, and avoid pruning production data.

## 2026-10-30 Online Chat now reuses Playground

- User explicitly chose the original Playground for customer Online Chat. This supersedes the earlier decision to hide Playground and use external chat presets for that customer entry.
- Customer navigation now links Online Chat directly to `/playground`, independent of external preset configuration. Restored the original Playground route guard (administrator module toggle only; authenticated parent retained), removing the customer-role redirect introduced by the redesign. Existing `chat.playground` configuration controls entry visibility and route availability; administrator navigation and existing backend `/pg/chat/completions` behavior are unchanged. Reused the complete Playground without duplicating chat logic.
- Regression-first verification: before implementation 3 cases failed (destination, disabled-module entry and customer route access). After implementation, all 23 cases in customer-navigation/sidebar-config/Playground route-access pass. Typecheck passes in 1,431 files, changed-file lint 54 TS files, protected-header-safe formatting, `git diff --check` and production build pass. No backend changes this follow-up; previous full Go failures remain documented above/below.
- Restarted fixture preview with latest build at `http://127.0.0.1:4191/playground` and added synthetic model choices. HTTP verifies page and model fixtures; no new browser visual QA and no actual model generation: the fixture server does not connect upstream AI. Real deployment uses the existing session-authenticated Playground endpoint and billing. Evidence: `chat-before.log`, `chat-after.log`, `chat-typecheck.log`, `chat-build.log` in external customer-ui-verification directory. No commit/push/deployment.

## 2026-10-30 Contact page and complete customer pricing

- Added customer sidebar entry `/contact`. Root administrators edit it at `/system-settings/site/contact` (Site Settings → Contact Us), using the existing settings form, unsaved-change guard, option mutation and persisted `Contact` option. Supports Markdown text, email/support links, multiple images via public HTTP(S) URLs, manual placement and a live preview. No actual channels are configured; customer view has a clear empty state. Image uploads/storage are not implemented; the editor inserts hosted image URLs.
- Reused sanitized RichContent for both editor preview and customer output, with constrained image sizes. Invalid image schemes/credentials are rejected by the insertion control; renderer strips executable markup. Failed saves retain the draft and allow retry. Memoized form defaults prevent stale defaults from replacing saved content. Existing root-only option-write authorization is retained; `/api/contact` exposes only intentional contact content and uses existing ETag revalidation. Registered its public-route exemption in the route coverage contract. No database schema or dependency changes.
- Model table now has four independent token prices: input, output, cache read and cache write. Added billing-method filtering and expandable tier/request/time-rule details using the existing expression parser and pricing components. Selected key/group multiplier and display currency also apply to all expanded tier prices. Explicit zero prices remain visible; missing prices remain absent/dashes. Per-request/task prices and units appear with the billing method, so they are not mistaken for output-token prices. Administrator pricing defaults retain their previous rendering.
- Reference evidence: read publicly served RightAPI model/pricing/contact assets. Browser automation remains unavailable (`failed to write kernel assets`, Windows error 3), so no new visual/browser verification is claimed for this follow-up.
- Verification: 77 distinct frontend cases across six affected suites pass, including final customer pricing rerun 11/11 and contact editor rerun 4/4 (Chinese resource labels, save/refresh, image preview, unsafe URLs, reset and failed-save retry). Existing dynamic/task price suites pass. Typecheck: 1,430 files; changed-file oxlint: 54 TS files; protected-header-safe format, `git diff --check` and production build pass. New contact controller test and full `go test -p 1 ./router -count=1 -timeout 90s` pass.
- Required full `go test -p 1 ./... -timeout 90s` completed non-green: controller reports Windows SQLite `audit.db` open-handle cleanup failures and a 90-second package timeout. The initial new-route coverage failure was fixed by classifying the intentionally public contact endpoint; full router rerun passes. Other packages, including model, relay and service, passed this run. Keep feature `in-progress` under repository verification gate; do not claim all backend checks pass.
- Latest built fixture preview restarted on `http://127.0.0.1:4191/catalog`; `/contact` is initially empty and admin settings support local fixture edits. Pricing fixture includes a tiered model. HTTP checks confirm built page/contact/settings/tier fixtures are served; these are synthetic data, not production. Evidence: `D:/WorkSpace/PlanetToken/customer-ui-verification` (`pricing-tests-final2.log`, `contact-tests-final.log`, `followup-tests-final.log`, `followup-typecheck-final.log`, `followup-build-final.log`, `contact-go-test2.log`, `router-tests-final.log`, `followup-go-full.log`). Changes remain uncommitted, unpushed and not deployed.

## 2026-10-29 Customer UI based on RightAPI

- Implemented on `codex/customer-ui-rightapi` in `D:/WorkSpace/PlanetToken/PlanetToken-customer-ui`, based on deployed `fb45dff52`; uncommitted and not deployed/pushed.
- Customer theme tokens are scoped to the authenticated document (including portals) and removed on role/unmount changes. Shared tables/cards/navigation retain existing business behavior; administrator visuals stay original. New compact overview, API endpoint card, independent recharge bills with desktop table/mobile cards, retryable failures, and profile/security/language account menu.
- Customer navigation hides referral, Playground, audit and Flow. Prices follow selected Key ID/group, or default when no keys exist. Auto groups and model allowlists are respected; unavailable groups never receive guessed prices. Existing price/cache/task expressions are reused. Key metadata omits credentials; storage contains only user-scoped Key ID. No backend, database, dependency or payment formula changes.
- Reuse review: customer pricing composes the existing pricing table/cells and Dialog; overview reuses SummaryCards calculations; billing reuses history hook/formatters/status and the original modal content. New desktop billing table fills the existing modal's missing full-page presentation; mobile retains existing cards. New endpoint card shares existing clipboard behavior. No duplicate authentication/payment implementation.
- Verified: 56 frontend tests in 7 files (18 pricing helper, 8 selection, 7 customer navigation, 13 existing sidebar configuration, 2 theme, 5 billing, 3 wallet copy). Final wallet rerun: 8 pass. Typecheck 1,421 files, lint of 39 changed TS files, protected-header-safe formatting, `git diff --check` and production build pass. Focused `go test -p 1 ./model ./controller -run 'TestFormat(User|Admin)Logs|TestTaskLogDTOHidesMappedModelFromUsers' -count=1` passes.
- Browser QA against actual built frontend plus local synthetic fixtures: desktop/mobile model table, premium Key multiplier/input/output/cache and details, compact overview, Chinese billing, keys, recharge, empty usage/task logs, account/language menu, and original admin theme/navigation. Chat preset navigation retained; real chat integration and payment processing were not exercised. Customer old `show_history` link is routed to full billing.
- Required full checks not green: `init.sh` initially hit C: disk/pagefile limits. Full serialized Go test with D: temp storage ended with controller SQLite open-handle cleanup failures and controller/relay timeouts; other packages, including service, completed. Known baseline already reproduced SQLite cleanup issues, but current timeout causes are not fully established. Full frontend lint also fails in untouched areas; changed-file lint passes. Feature stays `in-progress` under harness definition of done.
- Evidence outside repo: `D:/WorkSpace/PlanetToken/customer-ui-verification` contains preview server, screenshots and Go/build logs. Preview is `http://127.0.0.1:4191/catalog` with synthetic data. Before release, publish exact modified source and set `VITE_SOURCE_URL`; upstream-only attribution is insufficient for the modified release. User visual acceptance/staging checks remain.

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

**Last Updated:** 2026-10-27
**Active Epic:** feat-upstream-sync-20261027 — 在隔离分支合并上游并保留已提交功能
**Phase:** **合并冲突已解决，前后端构建与定向回归通过；全量后端验证受 Windows 失败影响，MySQL/PostgreSQL 矩阵待验证**
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
