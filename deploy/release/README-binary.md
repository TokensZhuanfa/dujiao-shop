# dujiao-shop 二进制版安装

你下载的这个 tarball 已经包含**所有需要的二进制 + systemd unit + nginx 模板 + 配置示例**。

## 两种版本对比

| 包名 | 包含 | 适合 |
|---|---|---|
| `dujiao-shop-fullstack_*` | `dujiao-api` (内嵌 admin+user 前端) + `admin-tool` | **单文件全栈**,不需 nginx,5 分钟跑起 |
| `dujiao-shop-headless_*`  | `dujiao-api-headless` + `admin-tool` + nginx 模板 | 已有 nginx,想把 admin/user 静态托管在 nginx |

> 想要 nginx 托管前端的,还需要单独下载 `dujiao-shop-admin-*.zip` 和 `dujiao-shop-user-*.zip` 解到 nginx root。

## 系统要求

- Linux 64 位 (x86_64 / aarch64)
- Redis 7+ (`apt install redis-server` 或 docker)
- 1 GB RAM、1 GB 磁盘
- 可选: nginx (headless 版需要)

## 安装步骤

```bash
# 1. 解压
tar xzf dujiao-shop-fullstack_v1.0.0_linux_amd64.tar.gz
cd dujiao-shop-fullstack_v1.0.0_linux_amd64/

# 2. 看这份安装说明 (你正在看)
cat INSTALL.md

# 3. 一键安装 (创建用户 / 复制二进制 / 装 systemd unit)
sudo ./install.sh

# 4. 编辑配置 (jwt_secret / redis / 支付通道 / 邮件等)
sudo vi /opt/dujiao/config.yml

# 5. 起服务
sudo systemctl enable --now dujiao
sudo journalctl -u dujiao -f
```

服务起来后浏览器访问 `http://你的IP:8080`,默认 admin 账号在数据库 migration 时生成,详见日志。

## 升级

下载新版 tarball → 解压 → 同样跑 `sudo ./install.sh`,会:
- 覆盖二进制
- **保留** config.yml + 数据库 + 上传文件
- 提示你 `systemctl restart dujiao`

## 数据存放位置

```
/opt/dujiao/
├── dujiao-api            # 主二进制
├── admin-tool            # 运维 CLI
├── config.yml            # 你的配置 (不会被升级覆盖)
├── db/                   # SQLite 数据库
├── uploads/              # 商品图 / 用户上传
├── credentials/          # 文件型卡密
└── logs/
```

## 运维 CLI 用法

```bash
# 列出所有管理员
sudo -u dujiao /opt/dujiao/admin-tool list-admins

# 重置某管理员的 2FA (TOTP 丢了用)
sudo -u dujiao /opt/dujiao/admin-tool reset-2fa --username admin
```

## 反向代理 (HTTPS)

二进制本身只跑 HTTP。生产环境前面要套 nginx + cert (Let's Encrypt 或 Cloudflare)。

### Fullstack 版的最小 nginx 配置

```nginx
server {
    listen 443 ssl http2;
    server_name your.domain.com;

    ssl_certificate     /etc/letsencrypt/live/your.domain.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/your.domain.com/privkey.pem;

    client_max_body_size 100M;

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

### Headless 版 (admin / user 分域名)

参考 `nginx/admin.conf` 和 `nginx/user.conf`(本 tarball 含)。

## 配置文件关键字段

参考 `config.template.yml`,**必改**的:
- `jwt_secret`: `openssl rand -base64 32`
- `redis.address`: 默认 `127.0.0.1:6379`
- `server.port`: 默认 8080
- `payment.*`: 支付通道密钥
- `smtp.*`: 邮件发送(找回密码、订单通知)

## 排查

```bash
# 看服务状态
sudo systemctl status dujiao

# 看实时日志
sudo journalctl -u dujiao -f

# 检查端口
sudo ss -ltnp | grep 8080

# 看配置语法
sudo -u dujiao /opt/dujiao/dujiao-api --check-config
```

## 卸载

```bash
sudo systemctl disable --now dujiao
sudo rm /etc/systemd/system/dujiao.service
sudo rm -rf /opt/dujiao    # ⚠️ 含数据库,删前先备份
sudo userdel dujiao
```
