# Codex Plus 图像生成（gpt-image-2）

PlanetToken 支持将 OpenAI Images API 请求转换为 Codex Responses API 的内置 `image_generation` 工具，从而通过 Codex Plus 通道调用 `gpt-image-2`。

## 通道配置

在管理后台新增 `codex` 通道：

- **Base URL**：填写 `https://chatgpt.com`（或你的兼容代理地址）。程序会自动追加 `/backend-api/codex/responses`。
- **Key**：必须是 JSON 对象，至少包含 `access_token` 和 `account_id`：

  ```json
  {
    "access_token": "<ChatGPT access token>",
    "account_id": "<ChatGPT account id>"
  }
  ```

- **模型**：使用 `gpt-image-2`。该模型已加入 Codex 内置模型列表。

请求头中的 `Authorization`、`chatgpt-account-id`、`OpenAI-Beta`、`originator` 和严格的 `Content-Type` 由适配器自动设置。

## 调用方式

### 文生图

```bash
curl https://your-planet-token.example.com/v1/images/generations \
  -H "Authorization: Bearer $PLANET_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-image-2",
    "prompt": "a watercolor illustration of a mountain lake",
    "size": "1024x1024",
    "quality": "high",
    "response_format": "b64_json"
  }'
```

### 图片编辑

使用 `multipart/form-data` 请求 `/v1/images/edits`，字段与 OpenAI Images API 兼容，例如 `model`、`prompt`、`image[]`、`mask`、`size`、`quality` 和 `response_format`。

适配器会把提示词、输入图片和遮罩转换成 Responses API 的 `input`，并以 `image_generation` 工具调用上游。上游 SSE 结果会聚合为标准 OpenAI Images 响应；默认返回 `b64_json`，也支持请求中的 `response_format`。

## Responses API

`/v1/responses` 同样支持 Codex 通道。请求中的 `image_generation` 工具若未填写模型，会自动补为 `gpt-image-2`。Codex 原生 Responses 请求会转发到：

```text
/backend-api/codex/responses
```

## 限制与排查

- Codex 通道不支持 `/v1/chat/completions`、`/v1/embeddings`、`/v1/audio` 等接口。
- `access_token` 或 `account_id` 缺失时，请求会在发送上游前失败。
- 上游返回 SSE 时，保持客户端请求为流式；图片接口最终仍会返回聚合后的 Images JSON。
- 若返回认证错误，请确认 token 属于对应的 ChatGPT 账号，并检查 `account_id` 是否匹配。
