# 安全建议

dujiao-shop 处理订单 + 卡密 + 资金,安全失守损失实在。**新部署上线前过一遍下面这张 checklist**。

## ✅ 必做(开站前)

### 1. 改默认密码 + 启 2FA

```
登录 → 设置 → 安全 → 修改密码(8 位+大小写+数字)
登录 → 设置 → 安全 → 启用 TOTP(Google Authenticator / 1Password 等)
保存 recovery codes 到密码管理器
```

> **不启 2FA 等于在用户名/密码漏出来时,你的店就被人接管了。**

### 2. 三个 secret 必须独立 + 高熵

```bash
# config.yml 里:
app.secret_key:       $(openssl rand -hex 32)   # ← 不同
jwt.secret:           $(openssl rand -hex 32)   # ← 不同
user_jwt.secret:      $(openssl rand -hex 32)   # ← 不同
```

**不要复制粘贴同一个 secret 到三处**。任一被推断出会引发不同攻击面(伪造 admin token / 伪造 user token / 解 cookie 加密数据)。

### 3. HTTPS 强制

所有 7 个端点(`/`, `/admin/`, `/api/v1/*`)都 HTTPS。

- 宝塔: 站点 → SSL → Let's Encrypt → **强制 HTTPS** 打钩
- Cloudflare: SSL/TLS → **Full(strict)** + **Always Use HTTPS** 打开
- nginx 手动: `listen 443 ssl http2;` + `if ($scheme != "https") { return 301 https://$host$request_uri; }`

### 4. 防火墙最小开放

```bash
# ufw 标准最小集
ufw default deny incoming
ufw allow 22/tcp      # SSH
ufw allow 80/tcp      # HTTPS redirect
ufw allow 443/tcp     # HTTPS
ufw enable
```

数据库 / redis / api 内部端口**绝不要直接公网开放**(127.0.0.1 bind 才安全)。

### 5. SSH 加固

```bash
# /etc/ssh/sshd_config
PermitRootLogin prohibit-password   # 改成只允许 key
PasswordAuthentication no           # 关密码登录(必须先有 key)
Port 22                              # 改成非 22 端口(可选,降低被扫频率)
```

```bash
# 强密码 / 改密码
passwd
# 或上 ssh key (本机)
ssh-copy-id root@your-server
```

## ⚠️ 强烈建议

### 6. 启用登录暴力破解锁定

`config.yml`:
```yaml
security:
  login_rate_limit:
    window_seconds: 300
    max_attempts: 5
    block_seconds: 900
```

5 分钟内同 IP 失败 5 次 → 锁 15 分钟。**默认已开,别关**。

### 7. 隐藏 admin 路径

```yaml
web:
  admin_path: "/my-secret-admin"   # 不写 /admin
```

降低被批量扫的概率。**这不是真安全,只是降扫描噪音**——真安全靠强密码 + 2FA。

### 8. 关闭 SVG 上传

`config.yml`:
```yaml
upload:
  allowed_types:    # 删 image/svg+xml
    - image/jpeg
    - image/png
    - image/gif
    - image/webp
  allowed_extensions:    # 删 .svg
    - .jpg
    - .jpeg
    - .png
    - .gif
    - .webp
```

SVG 可以内嵌 `<script>` 造成存储型 XSS。除非你强需求,**关掉**。

### 9. CORS 限定具体域名

```yaml
cors:
  allowed_origins:
    - "https://your.com"
    - "https://admin.your.com"
  # 不要用 "*"
```

虽然 dujiao-shop 的 admin 接口走 JWT 不依赖 cookie,CORS 改严格仍能挡掉一些路过流量。

### 10. 备份 + 异地

每天备份 + **异地存一份**。详见 [备份恢复](/ops/backup-restore)。**未验证过恢复的备份等于没备份。**

## 🛡️ 加分项(进阶)

### 11. WAF 防扫

挂 Cloudflare(免费版即有 WAF 规则)。或者自建 ModSecurity。
**关键拦截**:`union select`、`../../`、`<script`、`/admin.php`(扫 WordPress 残骸的)。

### 12. 数据库加密静态

SQLite:用 [SQLCipher](https://www.zetetic.net/sqlcipher/) 替代标准 SQLite。备份文件被偷也读不出来。

### 13. CSP(内容安全策略)

nginx 加 header:
```nginx
add_header Content-Security-Policy "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; img-src 'self' data: https:; connect-src 'self'" always;
add_header X-Frame-Options "DENY" always;
add_header X-Content-Type-Options "nosniff" always;
add_header Referrer-Policy "same-origin" always;
```

调试期可先 `Content-Security-Policy-Report-Only` 看哪些规则会误伤。

### 14. fail2ban

```bash
apt install fail2ban
# /etc/fail2ban/jail.local
[sshd]
enabled = true
maxretry = 3
bantime = 1h
```

SSH 22 上的暴力破解会被自动 ban。

### 15. 定期补丁

```bash
# 每周一次
apt update && apt list --upgradable

# 关键组件: 内核 / openssl / openssh / nginx / redis
unattended-upgrades   # debian 推荐开启自动安全补丁
```

## 🔥 上线后日常自查

每月跑一遍:

- [ ] `journalctl -u dujiao-api -f` 看错误率(突增可能在被攻击)
- [ ] 后台审计日志看有没异常操作
- [ ] `last`/`who` 看 SSH 登录历史有没陌生 IP
- [ ] `df -h` / `free -h` 看资源水位
- [ ] 跑一次备份恢复演练
- [ ] 检查 release / 上游 dujiao-next 有没新安全补丁

## ⛔ 不要做

- ❌ **不要把 config.yml 提交进 git** — 即使是私库
- ❌ **不要用 docker root 暴露 8080 到公网** — 必须前面套 nginx + HTTPS
- ❌ **不要在 admin 后台用同一个用户给多人共用** — 用真姓名建多 admin + RBAC
- ❌ **不要复用其他系统的 jwt_secret** — 一个网站一套独立 secret
- ❌ **不要禁用 2FA 来"省事"** — 强密码也会泄漏

## 漏洞披露

发现 dujiao-shop 漏洞,**不要公开发 issue**。私邮 / Telegram DM 联系仓库维护者,我们会优先修复并致谢。
