# Codex Image 2 流式生成验证记录

**记录日期：** 2026-08-13  
**功能范围：** Codex Plus `gpt-image-2` 的 Images SSE、心跳、取消、重试边界和按次计费  
**当前结论：** 本地基础验证完成；本地 E2E 和生产环境最终验收尚未完成

## 0. 本地测试环境状态

2026-08-13 已在 WSL2 Ubuntu Docker 中从当前工作区源码构建并启动：

| 服务 | 容器 | 地址/用途 |
|---|---|---|
| 前端开发服务器 | `planettoken-web-dev` | `http://localhost:5173` |
| PlanetToken 后端 | `new-api-dev` | `http://localhost:3000` |
| PostgreSQL | `new-api-dev-pg` | Compose 内网 |
| Redis | `new-api-dev-redis` | Compose 内网 |

- `/api/status` 返回 HTTP 200，前端代理 `/api/status` 同样返回 HTTP 200。
- 初始化页面为 `http://localhost:5173/setup`，已识别 PostgreSQL。
- Compose 项目名为 `planettoken-image-test`，数据位于独立 Docker 命名卷，不接触生产数据。
- 前端依赖位于 Docker 卷 `planettoken-web-node-modules`，未写入工作区 `node_modules`。
- 当前尚未创建管理员或添加 Codex Plus 凭据，文档不得记录密码、OAuth JSON、token 或 API key。

### WSL Docker 出站代理

本机 Windows VPN 提供 HTTP `10809` 和 SOCKS5 `10808`。WSL/Docker 容器不能使用
Windows 的 `127.0.0.1`，当前 WSL 网关为 `172.19.240.1`，因此应用使用：

```text
HTTP_PROXY=http://172.19.240.1:10809
HTTPS_PROXY=http://172.19.240.1:10809
NO_PROXY=localhost,127.0.0.1,postgres,redis,new-api
```

`docker-compose.dev.yml` 通过可选的 `LOCAL_HTTP_PROXY` 透传这些变量。WSL 网关可能在
重启后变化，应从 `wsl.exe -d Ubuntu -- ip route` 的默认路由读取，不应把当前地址作为
跨机器固定配置。

2026-08-13 诊断证据：未配置代理时，容器访问 `chatgpt.com:443` 和渠道测试稳定在
30 秒后 `i/o timeout`；配置 HTTP 10809 后 CONNECT 和 TLS 成功，渠道凭据刷新快速返回
HTTP 401。网络问题已解决，401 表示导入的 refresh token 被 OAuth 服务拒绝。生产环境
可能已自动刷新并轮换 refresh token，本地应重新导入生产数据库中当前最新的 OAuth JSON，
且不应让两套持续运行的环境长期共享并竞争轮换同一个 refresh token。

常用维护命令（在 Windows PowerShell 中执行）：

```powershell
# 查看后端日志
wsl.exe -d Ubuntu -u root -- docker logs -f new-api-dev

# 查看前端日志
wsl.exe -d Ubuntu -u root -- docker logs -f planettoken-web-dev

# 停止后端、数据库和 Redis，但保留数据卷
wsl.exe -d Ubuntu -u root -- docker-compose -p planettoken-image-test -f /mnt/d/WorkSpace/PlanetToken/PlanetToken/docker-compose.dev.yml down
wsl.exe -d Ubuntu -u root -- docker stop planettoken-web-dev
```

## 1. 已完成验证

本机使用 Go `1.25.1 windows/amd64` 完成以下验证：

```text
go test ./relay/channel/codex ./relay/helper ./constant ./relay ./controller -count=1

ok  github.com/QuantumNous/new-api/relay/channel/codex
ok  github.com/QuantumNous/new-api/relay/helper
ok  github.com/QuantumNous/new-api/constant
ok  github.com/QuantumNous/new-api/relay
ok  github.com/QuantumNous/new-api/controller
```

同时完成：

- `relaykit` 使用 `GOWORK=off go build ./...` 独立构建通过。
- `partial_images` 的 JSON、multipart 和越界校验测试通过。
- SSE frame、partial/completed/error 映射、`[DONE]` 和大 frame 测试通过。
- 心跳配置、客户端断开、错误脱敏和重试边界测试通过。
- completed 图片数参与固定价格结算的测试通过。
- `git diff --check` 通过。
- `feature_list.json` UTF-8 JSON 解析通过。
- Harness 结构验证为 `100/100`。

## 2. 本地 Go 环境

便携版 Go 位于：

```text
C:\Users\shan.gao\AppData\Local\Temp\codex-go-1.25.1-tar\go
```

新 PowerShell 会话中使用：

```powershell
$goRoot = Join-Path $env:TEMP 'codex-go-1.25.1-tar\go'
$env:GOROOT = $goRoot
$env:PATH = (Join-Path $goRoot 'bin') + ';' + $env:PATH
$env:GOPROXY = 'https://goproxy.cn,direct'
go version
```

该环境位于临时目录，不应视为长期系统安装。临时目录被清理后，需要重新安装或解压 Go `1.25.1`。

## 3. 全量测试的已知基线问题

执行 `go test ./... -count=1 -timeout=15m` 时，图片流相关包通过，但全量命令未完全通过：

1. 根包因缺少前端嵌入产物失败：`main.go:42:12: pattern web/dist: no matching files found`。
2. `TestObserveChannelAffinityUsageCacheByRelayFormat_UnsupportedModeKeepsEmpty` 在全量运行中受到共享状态污染；该测试使用 `-count=1` 隔离复跑通过。
3. 仓库的 `init.sh` 还要求 `web/default` 和 Bun，但当前代码树没有 `web/default`，本机也未发现 Bun。

这些问题不属于本次 Codex 图片流实现。修复前不能把 `./init.sh` 记录为通过。

## 4. 可在本地完成的 E2E

以下验证不必直接在生产环境进行，但需要真实可用的 Codex Plus OAuth 渠道：

- 启动本地 PlanetToken，并为渠道启用 `gpt-image-2`。
- 配置 `ModelPrice["gpt-image-2"]` 和测试用户额度。
- 调用 `/v1/images/generations` 与 `/v1/images/edits`。
- 验证只有显式 `stream:true` 返回 SSE；缺失或 `false` 仍返回 JSON。
- 在上游静默期间观察 `:\n\n` 心跳，最终收到 completed 和 `[DONE]`。
- 验证 `response_format:b64_json` 与 `response_format:url`。
- 验证 `n=1`、`n>1` 的预扣和最终扣费。
- 主动断开客户端，确认上游取消且按请求 `n` 保守计费。
- 在本地 Nginx 下缩短代理静默超时，确认心跳能够保持连接。

本地 E2E 可以证明应用逻辑、真实上游兼容性和计费链路，但不能证明生产代理链路配置正确。

### 2026-08-13 本地真实上游 E2E 结果

测试通过本地 `/v1/images/generations` 直连应用执行，使用 Codex Plus `gpt-image-2`、
`stream:true`、`response_format:b64_json` 和固定 `ModelPrice=0.5`。测试 token 与 OAuth
凭据未写入文件或输出记录。

| 场景 | 请求参数 | HTTP | 总耗时 | 心跳 | partial | completed | `[DONE]` | 最大活动间隔 |
|---|---|---:|---:|---:|---:|---:|---:|---:|
| 简单构图 | `1024x1024`, `quality=low`, `partial_images=0` | 200 | 31.896 s | 2 | 1 | 1 | 1 | 9.844 s |
| 复杂构图 | `1536x1024`, `quality=high`, `partial_images=3` | 200 | 74.933 s | 4 | 4 | 1 | 1 | 10.126 s |

复杂构图活动时间线（从请求开始计时）：

```text
heartbeat@17.202s
heartbeat@27.111s
partial@31.687s
heartbeat@41.813s
partial@50.150s
heartbeat@60.211s
partial@64.193s
partial@70.788s
completed@74.671s
[DONE]@74.784s
```

验证结论：

- 两次请求均收到唯一 completed 和唯一 `[DONE]`，没有 SSE error。
- 复杂请求超过一分钟，连接持续稳定，任意两次心跳或真实事件之间未超过 10.126 秒。
- 上游在复杂请求中返回 4 个真实 partial，适配器按契约转发真实事件，没有伪造进度。
- 两条消费日志均记录 `stream_status.status=ok`、`end_reason=done` 和渠道 `2`。
- 两次请求各扣 `250000 quota`；当前 `quota_per_unit=500000`，对应固定价格 $0.50。
- 应用日志没有该两次成功请求的 timeout、broken pipe、connection reset 或 panic。
- 本次尚未经过本地 Nginx/CDN，不能替代生产代理链验收。

## 5. 必须在生产环境最终确认

- 生产 Nginx、CDN 或负载均衡没有缓冲 SSE 心跳。
- 复杂图片生成耗时超过原 504 时间窗口时，客户端仍持续收到心跳或真实事件。
- 请求最终收到 completed 和 `[DONE]`。
- 生产数据库中的实际扣费符合 `ModelPrice * n * GroupRatio`。
- 对应 Nginx 日志中没有 `upstream timed out`。
- 客户端断开后，生产链路能够及时向应用传播取消信号。

## 6. 推荐执行顺序

1. 在本地启动 PlanetToken，接入真实 Codex Plus 渠道。
2. 完成本地 generation/edit、SSE、断连和计费 E2E。
3. 加入本地 Nginx，模拟原来的静默超时场景。
4. 部署到测试或生产环境，以小额度执行复杂生图。
5. 保存客户端 SSE 输出、扣费前后数据和 Nginx 日志作为最终证据。

在第 5 步完成前，项目状态应保持“代码完成、待生产验收”，不能宣称原始复杂生图 504 已完全解决。
