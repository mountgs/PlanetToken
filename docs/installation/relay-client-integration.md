# 中转站客户端接入指南

本文档面向**使用中转站 API 的终端用户**，说明如何在 Claude Code、Codex、Gemini CLI、Cursor 等工具中完成接入。

> **服务地址：** `{BASE_URL}`（由服务方提供，例如 `https://api.example.com`，不要末尾 `/`）  
> **API 密钥：** 登录中转站网站 → **令牌** 页面创建或复制（格式 `sk-xxxxxxxx`）  
> 下文用 `{API_KEY}` 表示你的密钥；把示例中的 `{BASE_URL}` 换成服务方给出的地址即可。若暂用 IP + HTTP，填 `http://你的服务器IP`。  
> **管理员（配置 Plus 渠道）：** 见 [Codex CLI 生产接入与校验规范](./codex-relay-production.md)

---

## 目录

1. [接入前须知](#接入前须知)
2. [方式一：网站一键导入 CC Switch（推荐）](#方式一网站一键导入-cc-switch推荐)
3. [方式二：CC Switch 手动配置](#方式二cc-switch-手动配置)
4. [方式三：直接配置（不用 CC Switch）](#方式三直接配置不用-cc-switch)
5. [连接测试与常见问题](#连接测试与常见问题)
6. [关于 HTTPS 与网络](#关于-https-与网络)

---

## 接入前须知

### 端点地址（务必区分）

| 用途 | 填写地址 | 适用工具 |
|------|----------|----------|
| **Anthropic 协议** | `{BASE_URL}`（**不要**加 `/v1`） | Claude Code、Claude Desktop |
| **OpenAI 协议（Codex 推荐）** | `{BASE_URL}/openai` | Codex CLI、CC Switch Codex 配置 |
| **OpenAI 协议（通用）** | `{BASE_URL}/v1` | Cursor（OpenAI 模式）、Cline、Continue 等 |
| **Gemini 协议** | `{BASE_URL}` | Gemini CLI |

### 密钥说明

- 从中转站 **令牌** 页面复制完整密钥。
- 若内容不带 `sk-` 前缀，接入时在前面补上 `sk-`。
- 请勿将密钥分享给他人或提交到公开仓库。

---

## 方式一：网站一键导入 CC Switch（推荐）

适合已安装 [CC Switch](https://github.com/farion1231/cc-switch) 的用户，可自动填好地址与密钥。

### 步骤

1. 浏览器打开 `{BASE_URL}` 并登录。
2. 进入 **令牌** → 找到你的密钥 → 点击 **CC Switch**。
3. 选择要接入的工具：
   - **Claude** → Claude Code / Claude Desktop
   - **Codex** → OpenAI Codex CLI
   - **Gemini** → Gemini CLI
4. 选择模型（列表中为当前账号可用的模型）。
5. 点击 **打开 CC Switch**，在弹出窗口中确认导入。
6. 在 CC Switch 中 **启用** 该供应商。

### 安装 CC Switch

| 平台 | 安装 |
|------|------|
| **Windows** | 从 [Releases](https://github.com/farion1231/cc-switch/releases) 下载 `CC-Switch-v3.16.2-Windows.msi` 或 Portable 版并安装 |
| **macOS** | `brew install --cask cc-switch`，或下载 `.dmg` 安装包 |
| **官网** | https://ccswitch.io |

### 生效方式

| 工具 | 说明 |
|------|------|
| Claude Code | 多数情况下切换后可直接使用 |
| Codex / Gemini | 需 **关闭并重新打开终端** 后再运行 CLI |

---

## 方式二：CC Switch 手动配置

无法使用一键导入时，在 CC Switch 中 **添加供应商 → 自定义**。

### Claude Code

CC Switch → **Claude** → **+** → 预设选 **自定义**：

```json
{
  "env": {
    "ANTHROPIC_API_KEY": "{API_KEY}",
    "ANTHROPIC_BASE_URL": "{BASE_URL}"
  }
}
```

填写 **API Key**、**端点**（`{BASE_URL}`）及可用 **模型 ID**，保存后启用。

### Codex CLI

> **范围说明：** Plus 账号中转当前仅支持 **Codex CLI**（`/v1/responses`）。ChatGPT 桌面/Web 内置 Codex、Cursor 等 Chat Completions 工具不在本方案内。管理员配置见 [codex-relay-production.md](./codex-relay-production.md)。

CC Switch → **Codex** → **+** → **自定义**：

**auth.json：**

```json
{
  "OPENAI_API_KEY": "{API_KEY}"
}
```

**config.toml：**

```toml
model_provider = "relay"
model = "gpt-4o"
model_reasoning_effort = "high"
disable_response_storage = true
preferred_auth_method = "apikey"

[model_providers.relay]
name = "relay"
base_url = "{BASE_URL}/openai"
wire_api = "responses"
requires_openai_auth = true
```

- Codex 的 `base_url` 使用 **`/openai`**（不是 `/api`）；通用 OpenAI 兼容工具仍可用 **`/v1`**。
- `model` 改为你套餐内可用的模型 ID。
- 若模型仅支持 Chat 协议，在 CC Switch 中开启 **需要本地路由映射**（界面会自动提示）。

保存并启用后，**重启终端** 再运行 `codex`。

### Gemini CLI

```json
{
  "env": {
    "GEMINI_API_KEY": "{API_KEY}",
    "GOOGLE_GEMINI_BASE_URL": "{BASE_URL}"
  }
}
```

### 统一供应商（一套密钥给多个工具）

1. CC Switch → **统一供应商** → **添加**
2. 名称、API Key、端点（`{BASE_URL}`）
3. 勾选 Claude Code / Codex / Gemini
4. **保存并同步**

### 其他 CC Switch 支持的工具

OpenCode、OpenClaw、Hermes 等可在对应 Tab 中选择 **OpenAI Compatible** 或 **自定义**：

- Base URL：`{BASE_URL}/v1`
- API Key：`{API_KEY}`

更多说明见 [CC Switch 用户手册（中文）](https://github.com/farion1231/cc-switch/tree/main/docs/user-manual/zh)。

---

## 方式三：直接配置（不用 CC Switch）

### Claude Code

| 平台 | 配置文件 |
|------|----------|
| Windows | `%USERPROFILE%\.claude\settings.json` |
| macOS / Linux | `~/.claude/settings.json` |

```json
{
  "env": {
    "ANTHROPIC_API_KEY": "{API_KEY}",
    "ANTHROPIC_BASE_URL": "{BASE_URL}"
  }
}
```

保存后重新打开终端，运行 `claude`。

### OpenAI Codex CLI

| 平台 | 目录 |
|------|------|
| Windows | `%USERPROFILE%\.codex\` |
| macOS / Linux | `~/.codex/` |

`auth.json`：

```json
{
  "OPENAI_API_KEY": "{API_KEY}"
}
```

`config.toml`：

```toml
model_provider = "relay"
model = "gpt-4o"
model_reasoning_effort = "high"
disable_response_storage = true
preferred_auth_method = "apikey"

[model_providers.relay]
name = "relay"
base_url = "{BASE_URL}/openai"
wire_api = "responses"
requires_openai_auth = true
```

> Codex 推荐 `base_url` 为 `{BASE_URL}/openai`；`/v1` 路径同样可用。

### Gemini CLI

| 平台 | 文件 |
|------|------|
| Windows | `%USERPROFILE%\.gemini\.env` |
| macOS / Linux | `~/.gemini/.env` |

```bash
GEMINI_API_KEY={API_KEY}
GOOGLE_GEMINI_BASE_URL={BASE_URL}
GEMINI_MODEL=gemini-2.0-flash
```

`GEMINI_MODEL` 请改为你套餐内可用的模型 ID。

### Cursor IDE

1. **Cursor Settings → Models**
2. **OpenAI API Key** → `{API_KEY}`
3. 开启 **Override OpenAI Base URL** → `{BASE_URL}/v1`
4. 若使用 Claude 模型：**Anthropic API Key** → `{API_KEY}`，**Override Anthropic Base URL** → `{BASE_URL}`（不带 `/v1`）

### VS Code 插件

| 插件 | 配置 |
|------|------|
| **Cline** | API Provider 选 OpenAI Compatible；Base URL `{BASE_URL}/v1`；Key `{API_KEY}` |
| **Continue** | `apiBase`: `{BASE_URL}/v1`；Key `{API_KEY}` |
| **Roo Code** | Custom OpenAI endpoint，同上 |

使用 Anthropic 类模型时，Base URL 填 `{BASE_URL}`（无 `/v1`）。

### 环境变量（可选）

**Windows PowerShell：**

```powershell
$env:ANTHROPIC_API_KEY = "{API_KEY}"
$env:ANTHROPIC_BASE_URL = "{BASE_URL}"
$env:OPENAI_API_KEY = "{API_KEY}"
$env:OPENAI_BASE_URL = "{BASE_URL}/v1"
```

**macOS / Linux：**

```bash
export ANTHROPIC_API_KEY="{API_KEY}"
export ANTHROPIC_BASE_URL="{BASE_URL}"
export OPENAI_API_KEY="{API_KEY}"
export OPENAI_BASE_URL="{BASE_URL}/v1"
```

---

## 连接测试与常见问题

### 快速测试（可选）

将 `sk-xxx` 换成你的密钥：

```bash
curl -s {BASE_URL}/v1/models -H "Authorization: Bearer sk-xxx"
```

返回模型列表或 JSON 即表示密钥与网络正常；若提示 `Invalid token`，请检查密钥是否完整、是否带 `sk-` 前缀。

### 常见问题

| 现象 | 可能原因 | 处理办法 |
|------|----------|----------|
| `Invalid token` | 密钥错误或已失效 | 在网站 **令牌** 页重新复制或新建密钥 |
| 连接超时 / 无法访问 | 本机或公司网络限制 | 确认能打开 `{BASE_URL}`；公司网络需放行 **出站** 80 端口（见下节） |
| Claude Code 报 404 | Base URL 多写了 `/v1` | 改为 `{BASE_URL}` |
| Codex 报 404 | Base URL 路径错误 | 改为 `{BASE_URL}/openai`（Codex）或 `.../v1`（通用 OpenAI） |
| 模型不存在 | 套餐未包含该模型 | 换文档/控制台中列出的模型 ID，或联系服务提供方 |
| CC Switch 链接无反应 | 未安装 CC Switch | 先安装后再点击，或改用手动配置 |
| 对话中途断开 | 网络不稳定或代理干扰 | 关闭 VPN/系统代理后重试；持续出现请联系服务提供方 |

### CC Switch 参考链接

- 官方仓库：https://github.com/farion1231/cc-switch
- 用户手册：https://github.com/farion1231/cc-switch/tree/main/docs/user-manual/zh
- 深度链接工具：https://farion1231.github.io/cc-switch/deplink.html

---

## 关于 HTTPS 与网络

### 当前接入地址

若服务方当前仅提供 **HTTP**，`{BASE_URL}` 会形如 `http://…`；有域名后应改为 `https://…`。

你在客户端填写的 Base URL 与服务方告知的地址一致即可；**无需自行配置证书或服务器**。

### 若服务方提供了 HTTPS 域名

当服务方通知已启用 `https://你的域名` 时：

- 将所有配置中的旧 `{BASE_URL}` 替换为 `https://你的域名`
- 规则不变：Claude **不带** `/v1`，Codex / OpenAI 兼容 **带** `/v1`

### 客户端网络要求

你只需保证本机能够 **向外访问** 中转站，通常意味着：

| 项目 | 说明 |
|------|------|
| 出站端口 | 当前使用 HTTP 时，需能访问 **TCP 80**；若已切换 HTTPS，需能访问 **TCP 443** |
| 入站端口 | **无需** 在你电脑上开放任何入站端口 |
| 公司 / 校园网 | 若无法打开 `{BASE_URL}`，请联系网络管理员放行对上述地址的出站访问 |
| 代理 / VPN | 部分代理会导致 CLI 连接异常，可尝试关闭后重试 |

### 常见问题：能不能直接用 IP 上 HTTPS？

HTTPS 证书通常绑定**域名**，不要指望用裸 IP 的 `https://` 就拥有正规证书。  
若你需要 HTTPS，请联系服务提供方是否提供域名接入；在未提供前，按服务方给出的 `{BASE_URL}` 配置即可。

---

## 配置速查

| 工具 | API Key | Base URL |
|------|---------|----------|
| Claude Code | `{API_KEY}` | `{BASE_URL}` |
| Codex | `{API_KEY}` | `{BASE_URL}/openai` |
| Gemini CLI | `{API_KEY}` | `{BASE_URL}` |
| Cursor（OpenAI） | `{API_KEY}` | `{BASE_URL}/v1` |
| Cursor（Claude） | `{API_KEY}` | `{BASE_URL}` |
