# 服务器上线操作步骤

> 在 **宝塔终端** 或 **你本机 SSH** 中执行（Cursor 环境可能无法直连你的服务器）。  
> **面向终端用户的客户端接入文档：** [中转站客户端接入指南](./relay-client-integration.md)  
> **Codex CLI 生产配置与校验：** [codex-relay-production.md](./codex-relay-production.md)

---

## 方式 A：上传部署包（推荐）

### 1. 在本机生成部署包

```bash
cd /path/to/new-api
chmod +x scripts/make-deploy-bundle.sh
./scripts/make-deploy-bundle.sh
```

得到 `deploy-bundle.tar.gz`。

### 2. 宝塔上传并解压

1. 登录宝塔：`https://{SERVER_HOST}:{BT_PANEL_PORT}`（端口以面板实际为准，勿对公网长期暴露）
2. **文件** → 进入 `/www/wwwroot/new-api`（没有则新建）
3. 上传 `deploy-bundle.tar.gz`
4. **终端** 中执行：

```bash
cd /www/wwwroot/new-api
tar xzf deploy-bundle.tar.gz
chmod +x scripts/*.sh
./scripts/bootstrap-on-server.sh
```

脚本会自动：生成 `.env.production` 随机密钥 → 拉镜像 → 启动 PostgreSQL + Redis + new-api。

### 3. 验证 Docker 层

```bash
cd /www/wwwroot/new-api
curl -s http://127.0.0.1:3000/api/status
# 应看到 "success":true
./scripts/verify-deployment.sh
```

---

## 方式 B：Git 克隆（推荐）

```bash
mkdir -p /www/wwwroot/new-api && cd /www/wwwroot/new-api
git clone git@github.com:mountgs/NewapiConfig.git .
chmod +x scripts/*.sh
./scripts/bootstrap-on-server.sh
```

---

## 4. 宝塔配置网站（浏览器可访问）

### 4.1 添加站点

- **网站** → **添加站点**
- 域名：你的域名，或暂时填 `{SERVER_HOST}`
- PHP：纯静态 / 不创建 PHP

### 4.2 反向代理

站点 → **反向代理** → 添加：

| 项 | 值 |
|----|-----|
| 目标 URL | `http://127.0.0.1:3000` |
| 发送域名 | `$host` |

在 **配置文件** 的 `location` 中确保有：

```nginx
proxy_buffering off;
proxy_cache off;
proxy_read_timeout 600s;
proxy_send_timeout 600s;
proxy_http_version 1.1;
proxy_set_header Connection "";
```

参考：[`deploy/nginx/new-api.conf.example`](../../deploy/nginx/new-api.conf.example)

### 4.3 HTTPS（有域名时）

**SSL** → **Let's Encrypt** → 申请 → 开启 **强制 HTTPS**

### 4.4 防火墙

- 放行：**80、443**（网站）
- **不要**对公网放行 3000、5432、6379
- SSH（端口 `{SSH_PORT}`，若已改非默认）仅允许可信 IP；禁止密码登录、仅用密钥

### 4.5 最终验证

```bash
cd /www/wwwroot/new-api
./scripts/verify-deployment.sh --url {BASE_URL}
# 有域名后：
./scripts/verify-deployment.sh --url https://你的域名
```

浏览器打开站点，完成首次管理员注册。

---

## 5. 首次备份

```bash
cd /www/wwwroot/new-api
./scripts/backup-db.sh --label initial
```

宝塔 **计划任务** 每天执行：

```bash
cd /www/wwwroot/new-api && ./scripts/backup-db.sh
```

---

## 6. 日常运维

详见 [`operations.md`](./operations.md)：

| 操作 | 命令 |
|------|------|
| 查看状态 | `docker compose -f docker-compose.prod.yml --env-file .env.production ps` |
| 查看日志 | `docker compose ... logs new-api --tail 100` |
| 升级 | 先 `backup-db.sh`，再 `pull` + `up -d` |
| 回滚 DB | `./scripts/restore-db.sh backups/xxx.sql.gz` |

---

## 7. 重要文件位置

| 路径 | 说明 |
|------|------|
| `/www/wwwroot/new-api/.env.production` | **密钥**（chmod 600，务必备份） |
| `/www/wwwroot/new-api/backups/` | 数据库备份 |
| `/www/wwwroot/new-api/data/` | 应用数据 |
| `/www/wwwroot/new-api/logs/` | 应用日志 |

---

## 8. 安全提醒

- 部署完成后请 **修改 SSH 密码** 和 **宝塔面板密码**（已在对话中暴露）
- 勿将 `.env.production` 提交到 Git 或发给他人
