# 生产运维操作手册

面向已按 [`production.md`](./production.md) 完成部署的 **宝塔 + Docker Compose + PostgreSQL** 环境。

---

## 1. 日常访问

| 项目 | 说明 |
|------|------|
| 管理后台 | `https://你的域名` |
| 健康检查 | `https://你的域名/api/status` → 应返回 `"success":true` |
| Codex CLI 生产校验 | `./scripts/verify-codex-relay.sh --url https://你的域名 --token sk-xxx --model gpt-5.3-codex` |
| 服务器本地检查 | `curl -s http://127.0.0.1:3000/api/status` |

---

## 2. 常用命令（在部署目录执行）

部署目录默认为 `/www/wwwroot/new-api`：

```bash
cd /www/wwwroot/new-api
COMPOSE="docker compose -f docker-compose.prod.yml --env-file .env.production"
```

### 2.1 查看状态

```bash
$COMPOSE ps
$COMPOSE logs new-api --tail 100 -f
./scripts/verify-deployment.sh --url https://你的域名
```

### 2.2 重启服务

```bash
# 仅重启应用
$COMPOSE restart new-api

# 全部重启
$COMPOSE restart
```

### 2.3 停止 / 启动

```bash
$COMPOSE stop
$COMPOSE up -d
```

---

## 3. 升级新版本

```bash
cd /www/wwwroot/new-api

# 1. 升级前备份（必须）
./scripts/backup-db.sh --label pre-upgrade-$(date +%F)

# 2. 拉取镜像（若 .env.production 中 pin 了 tag，先改 NEW_API_IMAGE）
$COMPOSE pull new-api

# 3. 重新创建容器
$COMPOSE up -d

# 4. 验证
./scripts/verify-deployment.sh --url https://你的域名
```

---

## 4. 回滚

### 4.1 只回滚程序版本

编辑 `.env.production`：

```env
NEW_API_IMAGE=calciumion/new-api:v0.9.0   # 改回旧版本 tag
```

然后：

```bash
$COMPOSE pull new-api
$COMPOSE up -d
./scripts/verify-deployment.sh
```

### 4.2 回滚数据库（升级后数据异常）

```bash
ls -lt backups/    # 找到升级前备份
./scripts/restore-db.sh backups/xxx_pre-upgrade.sql.gz
```

恢复后建议同步回滚 `NEW_API_IMAGE` 到对应版本。

---

## 5. 备份与恢复

### 5.1 手动备份

```bash
./scripts/backup-db.sh
./scripts/backup-db.sh --label manual-$(date +%F)
```

备份位置：`backups/*.sql.gz` 与 `backups/*.meta.json`

### 5.2 定时备份（宝塔）

**计划任务** → **Shell 脚本** → 每天执行：

```bash
cd /www/wwwroot/new-api && ./scripts/backup-db.sh
```

建议同时把 `backups/` 同步到对象存储或另一台机器。

### 5.3 恢复

```bash
./scripts/restore-db.sh backups/你的备份文件.sql.gz
# 按提示输入 RESTORE 确认
```

### 5.4 必须单独备份的文件

| 文件 | 原因 |
|------|------|
| `.env.production` | 含全部密钥，丢失无法解密 Redis 缓存数据 |
| `backups/` | 数据库历史 |

---

## 6. nginx / HTTPS（宝塔）

1. **网站** → 选择域名 → **SSL** → 续签 Let's Encrypt
2. **配置文件** 需保留：
   - `proxy_pass http://127.0.0.1:3000`
   - `proxy_buffering off`
   - `proxy_read_timeout 600s`
3. 参考模板：[`deploy/nginx/new-api.conf.example`](../../deploy/nginx/new-api.conf.example)

修改 nginx 后：**保存** → **重载配置**。

---

## 7. 故障排查

| 现象 | 处理步骤 |
|------|---------|
| 502 Bad Gateway | `$COMPOSE ps`；`curl 127.0.0.1:3000/api/status`；查 `$COMPOSE logs new-api` |
| 无法登录 / 会话失效 | 检查 `.env.production` 中 `SESSION_SECRET` 是否为空或为 `random_string` |
| 页面能开但聊天流中断 | nginx 加 `proxy_buffering off`，增大 `proxy_read_timeout` |
| 数据库连接失败 | `$COMPOSE logs postgres`；确认 `SQL_DSN` 与 PG 密码一致 |
| 磁盘满 | 清理 `logs/`、旧 `backups/`；`docker system prune`（谨慎） |

---

## 8. 安全运维要点

- 不要对公网开放 **3000 / 5432 / 6379**
- 不要提交 `.env.production` 到 Git
- 定期更新宝塔、系统补丁、Docker 镜像
- 管理员账号启用强密码；建议开启 2FA
- 升级前 **必须** 备份数据库

---

## 9. 首次部署检查清单

- [ ] `cp .env.production.example .env.production` 且所有密钥已替换
- [ ] `./scripts/deploy-prod.sh` 成功
- [ ] 宝塔 nginx + HTTPS 配置完成
- [ ] `./scripts/verify-deployment.sh --url https://域名` 通过
- [ ] 浏览器可登录管理后台
- [ ] `./scripts/backup-db.sh` 成功
- [ ] 宝塔计划任务已配置定时备份

---

## 10. 相关文档

| 文档 | 内容 |
|------|------|
| [relay-client-integration.md](./relay-client-integration.md) | **面向客户的** CLI / IDE 接入说明 |
| [codex-relay-production.md](./codex-relay-production.md) | **Codex CLI 生产配置与校验规范**（管理员） |
| [production.md](./production.md) | 首次部署、架构、安全清单 |
| [BT.md](./BT.md) | 宝塔入门 |
| [deploy/nginx/new-api.conf.example](../../deploy/nginx/new-api.conf.example) | nginx 配置模板 |
