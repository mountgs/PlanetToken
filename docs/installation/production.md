# 生产环境部署指南

本文档描述在 **宝塔面板 + Docker Compose + PostgreSQL + nginx + 官方镜像** 下的生产部署、升级、备份与回滚流程。

> 入门教程（应用商店一键装）：[`BT.md`](./BT.md)  
> 环境变量完整说明：[官方文档](https://docs.newapi.pro/zh/docs/installation/config-maintenance/environment-variables)

---

## 架构概览

```
Internet (HTTPS)
      │
      ▼
  nginx (宝塔网站 / SSL)
      │  proxy_pass → 127.0.0.1:3000
      ▼
  new-api (calciumion/new-api)
      ├── PostgreSQL (Docker 内网，不暴露端口)
      └── Redis (Docker 内网，不暴露端口)
```

| 组件 | 说明 |
|------|------|
| 镜像 | 官方 `calciumion/new-api`（后期可换私有 registry，改 `.env.production` 中 `NEW_API_IMAGE`） |
| 数据库 | PostgreSQL 15（持久化卷 `pg_data`） |
| 缓存 | Redis 7（带密码，仅内网） |
| TLS | nginx 终止 HTTPS（宝塔申请 Let's Encrypt） |
| 密钥 | `.env.production`（**勿提交 Git**） |

---

## 前置要求

| 项目 | 要求 |
|------|------|
| 宝塔面板 | ≥ 9.2.0 |
| Docker | 通过宝塔 Docker 插件安装 |
| 域名 | 已解析到服务器（用于 HTTPS） |
| 服务器 | 建议 ≥ 2 核 4G（PostgreSQL + Redis + 应用） |
| 防火墙 | **仅开放 80/443**；不要对公网开放 3000/5432/6379 |

---

## 一、首次部署

### 1.1 准备目录

在宝塔 **文件** 或 SSH 中创建部署目录：

```bash
mkdir -p /www/wwwroot/new-api
cd /www/wwwroot/new-api
```

将以下文件放到该目录（可从本仓库复制）：

- `docker-compose.prod.yml`
- `.env.production.example` → 复制为 `.env.production`
- `scripts/backup-db.sh`
- `scripts/restore-db.sh`

### 1.2 生成密钥

```bash
cp .env.production.example .env.production
chmod 600 .env.production

# 生成并填入 .env.production（每项不同）：
openssl rand -hex 32   # POSTGRES_PASSWORD
openssl rand -hex 32   # REDIS_PASSWORD
openssl rand -hex 32   # SESSION_SECRET
openssl rand -hex 32   # CRYPTO_SECRET
```

**必须修改** 所有 `CHANGE_ME` 值。`SESSION_SECRET` 不能使用 `random_string`（应用会拒绝启动）。

### 1.3 启动服务

```bash
docker compose -f docker-compose.prod.yml --env-file .env.production up -d
docker compose -f docker-compose.prod.yml --env-file .env.production ps
curl -s http://127.0.0.1:3000/api/status
```

确认 `success: true` 后再配置 nginx。

### 1.4 配置 nginx（HTTPS）

1. 宝塔 → **网站** → **添加站点**（域名如 `api.example.com`）
2. **SSL** → 申请 Let's Encrypt → 开启 **强制 HTTPS**
3. 站点 → **配置文件**，参考仓库 [`deploy/nginx/new-api.conf.example`](../../deploy/nginx/new-api.conf.example)：
   - `proxy_pass` 指向 `127.0.0.1:3000`
   - 开启 `proxy_buffering off`（流式响应必需）
   - `proxy_read_timeout` 建议 ≥ 600s
4. 保存并重载 nginx

### 1.5 初始化管理员

浏览器访问 `https://api.example.com`，按引导完成首次设置。

---

## 二、生产安全清单

部署完成后逐项确认：

- [ ] `.env.production` 权限为 `600`，且未提交到 Git
- [ ] `POSTGRES_PASSWORD`、`REDIS_PASSWORD`、`SESSION_SECRET`、`CRYPTO_SECRET` 均为随机强密钥
- [ ] 应用仅监听 `127.0.0.1:3000`（`docker-compose.prod.yml` 已配置）
- [ ] PostgreSQL / Redis **未**映射到宿主机端口
- [ ] 公网仅 80/443 开放；3000 不对公网暴露
- [ ] HTTPS 已启用；`TLS_INSECURE_SKIP_VERIFY` 未设为 `true`
- [ ] 若使用支付回调，配置 `TRUSTED_REDIRECT_DOMAINS`
- [ ] 宝塔面板、SSH 使用强密码或密钥登录

---

## 三、备份策略

### 3.1 手动备份

```bash
cd /www/wwwroot/new-api
chmod +x scripts/backup-db.sh scripts/restore-db.sh

# 升级前务必备份
./scripts/backup-db.sh --label pre-upgrade
```

备份输出：

- `backups/new-api_<db>_<timestamp>_<label>.sql.gz` — 数据库 dump
- 同名 `.meta.json` — 镜像 tag、时间等元数据

### 3.2 定时备份（宝塔计划任务）

宝塔 → **计划任务** → **Shell 脚本**，例如每天 3:00：

```bash
cd /www/wwwroot/new-api && ./scripts/backup-db.sh
```

建议：

- 保留至少 7 天备份（可配合宝塔「备份到云存储」或 `find backups -mtime +7 -delete`）
- **每次升级前** 额外执行 `--label pre-upgrade-$(date +%F)`

### 3.3 备份内容说明

| 数据 | 位置 | 备份方式 |
|------|------|---------|
| PostgreSQL | Docker 卷 `pg_data` | `scripts/backup-db.sh`（推荐） |
| 应用本地文件 | `./data`、`./logs` | 可选：打包目录或卷快照 |
| 配置 | `.env.production` | **单独加密备份**（含全部密钥） |

---

## 四、升级流程

### 4.1 标准升级（官方镜像）

```bash
cd /www/wwwroot/new-api

# 1. 备份
./scripts/backup-db.sh --label pre-upgrade

# 2. 拉取新镜像（可选：在 .env.production 中 pin 具体 tag，如 v0.x.x）
docker compose -f docker-compose.prod.yml --env-file .env.production pull new-api

# 3. 滚动重启
docker compose -f docker-compose.prod.yml --env-file .env.production up -d

# 4. 验证
curl -s http://127.0.0.1:3000/api/status
# 浏览器登录、抽查关键 API
```

### 4.2 固定版本（推荐生产）

在 `.env.production` 中：

```env
NEW_API_IMAGE=calciumion/new-api:v0.10.0
```

升级时改 tag → `pull` → `up -d`，便于与备份 metadata 对应。

---

## 五、回滚流程

回滚分两层：**应用镜像** 与 **数据库**。

### 5.1 仅回滚应用（数据库未破坏）

```bash
# 在 .env.production 改回旧 tag
NEW_API_IMAGE=calciumion/new-api:v0.9.0

docker compose -f docker-compose.prod.yml --env-file .env.production pull new-api
docker compose -f docker-compose.prod.yml --env-file .env.production up -d
```

### 5.2 数据库回滚（升级后数据异常）

```bash
cd /www/wwwroot/new-api

# 使用升级前备份
./scripts/restore-db.sh backups/new-api_new-api_YYYYMMDD..._pre-upgrade.sql.gz

# 若需同时回滚镜像，改 .env.production 中 NEW_API_IMAGE 后再 up -d
```

恢复脚本会：停止 `new-api` → 重建数据库 → 导入 dump → 启动 `new-api`。

---

## 六、故障排查

| 现象 | 排查 |
|------|------|
| 502 Bad Gateway | `docker compose ps`；`curl 127.0.0.1:3000/api/status` |
| 登录后立即失效 | 检查 `SESSION_SECRET` 是否设置且非默认值 |
| Redis 相关解密错误 | 确认 `CRYPTO_SECRET` 与备份时一致，未随意更改 |
| 流式响应中断 | nginx 需 `proxy_buffering off`，增大 `proxy_read_timeout` |
| 容器反复重启 | `docker compose logs new-api --tail 100` |

---

## 七、后期更换镜像源

当前使用官方 `calciumion/new-api`。迁移到私有 registry 时：

1. 将镜像推送到私有仓库
2. 修改 `.env.production`：`NEW_API_IMAGE=registry.example.com/new-api:v1.0.0`
3. `docker login` → `docker compose pull` → `up -d`

数据库与 Redis 配置无需变更。

---

## 八、相关文件

| 文件 | 用途 |
|------|------|
| `docker-compose.prod.yml` | 生产编排（PostgreSQL + Redis + app） |
| `.env.production.example` | 环境变量模板 |
| `scripts/backup-db.sh` | PostgreSQL 备份 |
| `scripts/restore-db.sh` | PostgreSQL 恢复 |
| `deploy/nginx/new-api.conf.example` | nginx 反向代理示例 |
| `docs/installation/operations.md` | 日常运维操作手册 |

---

## 九、检查清单（Definition of Done）

- [ ] `./scripts/backup-db.sh` 成功且可 `restore-db.sh` 演练通过
- [ ] HTTPS 访问正常，流式聊天无中断
- [ ] 安全清单第二节全部勾选
- [ ] 计划任务已配置定时备份
