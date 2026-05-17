# 故障排查

排查的**第一性原理**:**从下往上看日志**(网络层 → nginx → api → 数据库 → 缓存),哪一层报错就修哪层。

## 常用诊断命令

```bash
# 1. api 是不是活的
systemctl status dujiao-api
ps -ef | grep dujiao-api | grep -v grep
curl -fsS http://127.0.0.1:8080/health      # {"status":"ok"}

# 2. 监听端口
ss -ltnp | grep -E ':80|:8080|:8081|:8082'

# 3. 实时 api 日志
journalctl -u dujiao-api -f
# 或 (docker)
docker compose logs -f api
# 或 (二进制 + 我们设的 log 文件)
tail -f /var/log/dujiao/api.log

# 4. nginx 错误日志
tail -f /var/log/nginx/error.log
# 宝塔
tail -f /www/wwwlogs/dujiao-user.error.log

# 5. redis
systemctl status redis-server
redis-cli ping
redis-cli -n 1 keys '*' | head -20       # 看 asynq queue 数据

# 6. 数据库连得通吗
sqlite3 /opt/dujiao/db/dujiao.db 'SELECT count(*) FROM orders;'

# 7. 资源
top -bn1 -o %CPU | head -15
free -h
df -h /
```

## 按现象排查

### api 启动就挂

```bash
# 看 journal 完整启动日志(从最近一次启动开始)
journalctl -u dujiao-api -b --since '10 min ago'

# 常见挂法:
# - config.yml 路径找不到 → 看错误指明的查找路径,放对位置
# - jwt_secret 为空 → 必填,生成: openssl rand -hex 32
# - 数据库连接失败 → driver / dsn 字段对照 reference/config
# - redis 连接失败 → host: 127.0.0.1 (非 docker), 6379 端口通,redis-cli ping 能通
```

### 接口返回 502 Bad Gateway

nginx 反代到 8080 但 api 进程挂了。
```bash
systemctl is-active dujiao-api
# 不 active 就重启
systemctl restart dujiao-api
journalctl -u dujiao-api -f
```

### 接口返回 401 但密码明明对

可能是 JWT secret 变了导致所有现有 token 失效。让用户**重新登录**即可。
```bash
# 看是不是因为 secret 改了
grep -i 'jwt.*invalid\|token.*invalid' /var/log/dujiao/api.log | tail -10
```

### 接口返回 500 / panic

api 进程内部错误。
```bash
# 翻 journal 找 panic stack
journalctl -u dujiao-api --no-pager | grep -A 20 'panic:\|runtime error' | tail -40
```

最常见 panic:
- nil pointer:某字段从老数据库迁移时为 NULL,新代码访问时崩
- gorm "record not found":未捕获的 NotFound,通常配合 401/404 一起出现

### 前端白屏 / JS 加载失败

F12 → Network 看具体哪个资源 404。
- `/assets/index-xxx.js` 404 → 站点根目录没解前端 dist,或解错位置(应该解到 `/www/wwwroot/dujiao-user/` 不是 `dujiao-user/dist/`)
- `/api/v1/public/config` 404 → 伪静态没贴对
- `/api/v1/public/config` 500 → api 内部错,看 api 日志

### 上传图片失败

```bash
# 看 nginx body 大小
grep -i 'client_max_body' /www/server/nginx/conf/nginx.conf /www/server/panel/vhost/nginx/*.conf

# 伪静态里必须有 client_max_body_size 100M;
# api 这边 config.yml upload.max_size 也得够大
```

### 订单卡 pending

参考 [FAQ](/faq#q-用户付了钱-订单还是-pending)。

### redis i/o timeout(高 CPU)

`config.yml` `redis.host` 设错。改 `127.0.0.1`,重启 api。详见 [FAQ 安装段](/faq)。

### 磁盘满了 / 日志爆炸

```bash
# 看哪里占
du -sh /opt/dujiao/* | sort -h
du -sh /var/log/dujiao/*

# 多半是 logs/。改 config.yml:
#   log.max_size_mb: 50  (默认 100)
#   log.max_backups: 3   (默认 7)

# 或者干脆截断当前日志:
truncate -s 0 /var/log/dujiao/api.log
```

### 数据库锁 / "database is locked"

SQLite 出现并发写时。检查:
```yaml
database:
  pool:
    max_open_conns: 1    # 必须 1
```

如果还是出现,临时:
```bash
sqlite3 /opt/dujiao/db/dujiao.db 'PRAGMA journal_mode=WAL;'
```
WAL 模式 SQLite 并发性能好很多。

## 健康检查端点

写监控就用这两个:

| 端点 | 含义 | 期望响应 |
|---|---|---|
| `GET /health` | api 进程存活 + DB 可连 | `200 {"status":"ok"}` |
| `GET /api/v1/public/config` | 业务路由通且配置加载 | `200 {"status_code":0,...}` |

Cron 监控:
```cron
*/5 * * * * curl -fsS http://127.0.0.1:8080/health > /dev/null || systemctl restart dujiao-api
```

## 完整日志归档导出(报 issue 用)

```bash
mkdir /tmp/diag-$(date +%s) && cd /tmp/diag-*
journalctl -u dujiao-api --since '1 hour ago' > journal.log
cp /var/log/dujiao/*.log .
cp /opt/dujiao/config.yml config.yml.txt   # 提交前删 secret!
systemctl status dujiao-api > status.txt
ss -ltnp > ports.txt
free -h > mem.txt
docker compose ps > docker.txt 2>/dev/null || true
tar czf ../diag.tar.gz .
echo "上传 /tmp/diag.tar.gz 给 issue"
```

> 提交 issue 前**务必把 config.yml 里的 secret/jwt/admin_password 删了再传**。
