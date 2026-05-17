# config.yml 配置详解

dujiao-shop 所有运行期参数都集中在 **`config.yml`** 一个文件里。本页按区块说明每个字段。

模板:[`deploy/config.template.yml`](https://github.com/TokensZhuanfa/Dujiao-Shop/blob/main/deploy/config.template.yml)。**生产部署的 `config.yml` 不要进 git。**

## 🔑 必改的 5 项 (新装必看)

```yaml
app.secret_key:            # openssl rand -hex 32
jwt.secret:                # openssl rand -hex 32 (不同的 32 字节)
user_jwt.secret:           # openssl rand -hex 32 (再来一个不同的)
bootstrap.default_admin_password:  # 首次启动会创建 admin 用户用的密码
redis.host:                # docker compose 默认 "redis";二进制部署改 "127.0.0.1"
```

> **`secret_key` / `jwt.secret` / `user_jwt.secret` 必须三个都不同**,任何一个被推测出来都会导致严重安全问题。

---

## app

```yaml
app:
  secret_key: __APP_SECRET__   # AES/HMAC 全局加密密钥, 32 字节 hex
  totp_issuer: Dujiao-Shop      # 用户在 Authenticator app 里看到的项目名
```

## server

```yaml
server:
  host: 0.0.0.0        # 监听地址。生产前面套 nginx 时可改 127.0.0.1
  port: 8080           # HTTP 端口
  mode: release        # release / debug。debug 会打印 SQL 全文,生产请用 release
```

## log

```yaml
log:
  dir: ""              # 空 = stderr。指定路径如 /var/log/dujiao 写文件
  filename: app.log
  max_size_mb: 100     # 单文件大小上限
  max_backups: 7       # 保留几份历史
  max_age_days: 30     # 多老的删
  compress: true       # 旧文件 gzip
```

## database

默认 SQLite,小流量足够。要 MySQL/PostgreSQL 改 driver + dsn:

```yaml
database:
  driver: sqlite                 # sqlite | mysql | postgres
  dsn: ./db/dujiao.db            # SQLite 路径; MySQL: user:pass@tcp(host:3306)/db?charset=utf8mb4&parseTime=True
  pool:
    max_open_conns: 1            # SQLite 必须 1; MySQL 推荐 20-50
    max_idle_conns: 1
    conn_max_lifetime_seconds: 0
    conn_max_idle_time_seconds: 0
```

> **SQLite 的 `max_open_conns: 1` 不要改**——SQLite 不支持并发写,改大反而引入锁竞争。

## jwt / user_jwt

后台管理员和前台用户用**两套独立的 JWT secret**。

```yaml
jwt:                           # admin 后台
  secret: __JWT_SECRET__
  expire_hours: 24             # 24 小时不动就过期

user_jwt:                      # user 前台
  secret: __USER_JWT_SECRET__
  expire_hours: 24
  remember_me_expire_hours: 168   # "记住我" 复选框打钩则 7 天
```

改 secret 等于**强制所有现有 session 失效**,所有人重新登录。

## bootstrap

首次启动时如果数据库没有 admin 用户,自动用这个建一个:

```yaml
bootstrap:
  default_admin_username: admin
  default_admin_password: __ADMIN_PASSWORD__
```

**已存在 admin 后这两个字段无效**。改密码请进后台或用 `admin-tool` CLI。

## redis / queue

`redis` 用作缓存 + session,`queue` 用作 [Asynq](https://github.com/hibiken/asynq) 异步任务队列。**默认共用同一 redis 实例,不同 db 编号**。

```yaml
redis:
  enabled: true
  host: redis          # docker = "redis"; 二进制 = "127.0.0.1"  ← 必改
  port: 6379
  password: ""
  db: 0
  prefix: "dj"

queue:
  enabled: true
  host: redis          # 同上
  port: 6379
  password: ""
  db: 1                # 跟 redis 用不同 db 防互相污染
  concurrency: 10      # 同时跑几个 worker
  queues:
    default: 10
    critical: 5
  upstream_sync_interval: "5m"
```

## upload

商品图、用户头像等上传约束:

```yaml
upload:
  max_size: 10485760           # 10 MB
  allowed_types:               # MIME 白名单
    - image/jpeg
    - image/png
    - image/gif
    - image/webp
    - image/svg+xml
  allowed_extensions:          # 后缀白名单
    - .jpg
    - .jpeg
    - .png
    - .gif
    - .webp
    - .svg
  max_width: 4096
  max_height: 4096
```

> 推荐**关掉 `svg`**(改 `allowed_types` / `allowed_extensions` 都删 svg),SVG 可以内嵌 `<script>` 造成 XSS。

## credentials

文件型卡密(`auto_secret_kind=file`)的存储:

```yaml
credentials:
  root: ./credentials
  max_size: 1073741824         # 1 GB / 单文件上限
```

文件路径不进数据库,只存文件名映射;`root` 必须 web 不可直接访问。

## cors

跨域。生产建议把 `"*"` 改成你的实际域名:

```yaml
cors:
  allowed_origins:
    - "https://your.com"
    - "https://admin.your.com"
```

## security

```yaml
security:
  login_rate_limit:            # 暴力破解防护
    window_seconds: 300        # 5 分钟内
    max_attempts: 5            # 5 次失败
    block_seconds: 900         # 锁 15 分钟
  password_policy:
    min_length: 8
    require_upper: true        # 至少 1 个大写
    require_lower: true
    require_number: true
    require_special: false     # 推荐打开
```

## email

SMTP 发件配置(找回密码、订单通知、注册验证码用):

```yaml
email:
  enabled: false                  # 不用邮件功能可保持 false
  host: smtp.example.com
  port: 465
  username: ""
  password: ""
  from: ""                        # 发件人邮箱
  from_name: "Dujiao-Shop"
  use_tls: false                  # 587 端口用 TLS
  use_ssl: true                   # 465 端口用 SSL
  verify_code:
    expire_minutes: 10
    send_interval_seconds: 60     # 同一邮箱 1 分钟只能发一次
    max_attempts: 5
    length: 6
```

常见 SMTP 配置参考:

| 服务商 | host | port | use_ssl | use_tls |
|---|---|---|---|---|
| 阿里云 | `smtpdm.aliyun.com` | 465 | ✓ | |
| 腾讯企业邮 | `smtp.exmail.qq.com` | 465 | ✓ | |
| QQ 邮箱 | `smtp.qq.com` | 465 | ✓ | |
| Gmail | `smtp.gmail.com` | 587 | | ✓ |
| SendGrid | `smtp.sendgrid.net` | 587 | | ✓ |

## order

```yaml
order:
  payment_expire_minutes: 15    # 订单 15 分钟没付款自动取消(释放预占库存)
  max_refund_days: 30           # 多少天内可退款
```

## telegram_auth

Telegram 登录 / Telegram Mini App 集成:

```yaml
telegram_auth:
  enabled: false                # 不用就 false
  bot_username: ""              # 不带 @
  bot_token: ""                 # @BotFather 给的 token
  mini_app_url: ""              # 可选,Telegram Mini App URL
  login_expire_seconds: 300     # 登录 widget hash 有效期
  replay_ttl_seconds: 300       # 防重放窗口
```

## web

```yaml
web:
  admin_path: "/admin"          # 后台路径前缀。改成别的(如 /a-secret-path)
                                # 增加被扫描的难度
```

---

## 改完怎么生效

| 部署方式 | 重载命令 |
|---|---|
| Docker Compose | `docker compose restart api` |
| 二进制 / 宝塔 | `systemctl restart dujiao-api`(或 `dujiao`)|
| 源码 dev | `Ctrl+C` 重跑 `go run` |

> dujiao-api 不监听文件变化,改 `config.yml` 必须**手动重启**才生效。
