# Codex Image 2 流式生成需求

**状态：** 代码完成，待生产验收
**基线：** `main` @ `721c1a65`
**范围：** 在现有 Codex `gpt-image-2` Images API 实现上增量添加流式能力

## 1. 目标

为 Codex 渠道的 `/v1/images/generations` 与 `/v1/images/edits` 增加显式
`stream:true` 支持，使复杂、长耗时图片生成期间能够持续向客户端发送真实图片事件或
SSE 心跳，避免下游代理因响应静默而超时。

本功能不得替换现有 Codex 适配器，也不得改变未请求流式输出的客户端行为。

## 2. API 契约

### 2.1 显式启用

- 只有客户端显式传入 `stream:true` 时返回 `text/event-stream`。
- `stream` 缺失或为 `false` 时，继续返回当前 OpenAI Images JSON。
- 本阶段不启用非流式 JSON 前导空白心跳；非流式请求继续依赖足够长的代理超时。

### 2.2 对外事件

Codex Responses 事件只在适配器内部解析，对外保持 Images API 语义：

| 端点 | 中间图片 | 完成图片 |
|---|---|---|
| `/v1/images/generations` | `image_generation.partial_image` | `image_generation.completed` |
| `/v1/images/edits` | `image_edit.partial_image` | `image_edit.completed` |

- 上游没有 partial 时不得伪造进度，只发送心跳并等待真实完成事件。
- 每个完成事件携带完整图片、创建时间以及上游提供的尺寸、质量、格式、修订提示词和 usage。
- `response_format:b64_json` 返回 `b64_json`。
- `response_format:url` 返回 `data:<mime>;base64,...`。
- 正常结束或流内错误结束后发送 `data: [DONE]`。

### 2.3 `partial_images`

- 客户端未提供时默认 `0`，不得因 `stream:true` 自动开启预览图。
- 显式值必须为整数 `0..3`，否则在请求上游前返回 HTTP 400。
- 仅转发上游真实返回的 partial image。

## 3. 流处理

- 在 `relay/channel/codex` 内新增 Codex 图片专用 SSE frame 处理器。
- 不直接使用会丢弃 `event:` 行的通用行级 scanner。
- 正确解析完整 SSE frame，包括 `event:`、多行 `data:`、注释、空行边界和 `[DONE]`。
- 单个 SSE 事件最大 `128 MiB`；超过上限必须受控失败，不得无限分配内存。
- 不再对流式请求执行 `io.ReadAll(resp.Body)`；只保留当前 frame 和完成事件所需状态。
- 所有下游事件与心跳由同一个 writer loop 串行写入并立即 `Flush()`。

## 4. 心跳配置

新增独立环境配置 `IMAGE_STREAM_PING_INTERVAL`：

- 默认值：`10` 秒。
- `0`：禁用。
- 非零有效范围：`5..60` 秒。
- 非法值必须在启动配置校验阶段报错，不得静默回退。
- 仅影响 Codex Images `stream:true` 请求，不复用聊天流全局 ping 开关。
- 每次真实 SSE 事件写出后重置心跳计时。
- 连续静默达到间隔时写入标准 SSE 注释 `:\n\n` 并立即 Flush。

## 5. 取消、错误与重试

### 5.1 客户端断开

- 客户端断开后立即取消上游请求并关闭 `resp.Body`。
- 不为获取 usage 而继续 drain 或继续生成。

### 5.2 错误输出

- 尚未写出任何下游字节时，使用正确 HTTP 状态码和 OpenAI JSON 错误。
- 流已提交后，发送 `event:error`，其 `data` 为脱敏的 OpenAI 错误对象，然后发送 `[DONE]` 并终止。
- 不得泄露 OAuth token、ChatGPT account ID、内部 URL 或敏感响应头。
- 错误之后不得继续发送 partial 或 completed。

### 5.3 重试边界

- 未写真实图片事件前，可沿用现有可重试错误的渠道重试/切换机制。
- SSE 注释心跳不算真实输出，不得单独阻止重试。
- 写出任意 partial 或 completed 后，禁止重试或切换渠道，避免结果混流及重复生成。
- 参数错误、内容审核拒绝和账号缺少 Image 2 权限不可重试。
- 上游 5xx、传输中断或无图空终态，仅在未写真实图片事件时可重试。

## 6. 按次计费

- 管理员通过 `ModelPrice["gpt-image-2"]` 配置单张固定价格。
- 计费公式为 `ModelPrice * n * GroupRatio`。
- 请求开始时按请求 `n` 预扣。
- 正常完成时按实际 completed 图片数结算。
- 心跳和 partial image 不计入图片数量。
- 客户端断开时按请求 `n` 保守计费，不因缺少 completed 自动退款。
- 上游在开始生成前明确失败时，按现有结算机制退还预扣。

## 7. 范围边界

- 仅修改 Codex 渠道的 `gpt-image-2` Images 请求。
- 不改变 OpenAI、Gemini、Replicate 或其他图片渠道。
- 不改变 `/v1/responses` 的原生 `response.*` SSE 契约。
- 不新增非流式 JSON 空白心跳。
- 不引入异步图片任务 API。
- 暂不抽取跨渠道公共图片流抽象；实现稳定后再评估。

## 8. 验收门槛

### 自动化

1. 上游分段发送 SSE 时，第一个下游事件必须在上游 EOF 前可见。
2. 上游静默时按配置发送 `:\n\n` 并 Flush；使用可控时钟/ticker，不写真实 sleep 测试。
3. generation/edit 的 partial、completed、error 和 `[DONE]` 映射正确。
4. 多行 `data:` 和超过 64 KiB 的 Base64 frame 可处理；超过 128 MiB 受控失败。
5. 客户端取消会关闭上游 body，且计费倍率保留请求 `n`。
6. 心跳后仍允许重试；真实 partial/completed 后禁止重试。
7. `partial_images` 仅接受 `0..3`。
8. `stream` 缺失和 `stream:false` 的 JSON 响应保持现有契约。
9. 定向测试通过：

   ```bash
   go test ./relay/channel/codex ./relay/helper ./relay/constant
   ```

10. 按仓库规则运行 `./init.sh`；若被已知环境问题阻塞，必须记录完整命令、错误和定向测试结果。

### 生产 E2E

- 使用复杂提示词，使生成耗时超过此前出现 504 的时间窗口。
- 客户端在等待期间持续收到心跳或真实图片事件。
- 最终收到 completed 和 `[DONE]`。
- Nginx 日志不得出现该请求的 `upstream timed out`。
- 生产 E2E 完成前只能标记“代码完成、待线上验收”，不得宣称故障完全解决。
