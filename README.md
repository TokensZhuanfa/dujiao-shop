# dujiao-shop

> **自托管自动发卡 / 卡密商店**,基于 [dujiao-next](https://github.com/dujiao-next) 上游 fork,加上 Codex 号池、文件型卡密、支付订单安全收紧等二次开发。

[![Release](https://img.shields.io/github/v/release/TokensZhuanfa/dujiao-shop?style=flat-square)](https://github.com/TokensZhuanfa/dujiao-shop/releases)
[![License](https://img.shields.io/github/license/TokensZhuanfa/dujiao-shop?style=flat-square)](LICENSE)

## ✨ 特性

- 🛒 **完整商品/订单/支付/库存** — 上游 dujiao-next 的核心功能
- 🔐 **多种卡密交付** — 文本卡密、文件型卡密、号池型(账号)、CpaMC / Sub2api 双格式
- 🤖 **Codex 号池** — 自动维护 OpenAI/ChatGPT 账号的 token 轮换、额度刷新、状态识别(banned / needs_refresh / ok 等),库存即时反映可用账号数
- 🛡️ **安全收紧** — JWT + 2FA、订单号 base32 高熵、guest 接口限流、bcrypt cost 10、密码最小 8 位
- 💼 **后台运维 CLI** (`admin-tool`) — list-admins / reset-2fa
- 🚀 **四种部署方式** — Docker Compose / 宝塔面板 / 单文件二进制 / 源码编译,按场景选

## 🚀 快速开始

### 选你的部署方式

| 场景 | 推荐 |
|---|---|
| 第一次部署、不熟运维 | [**Docker Compose**](#1-docker-compose-推荐) |
| 已经在用宝塔面板管站 | [**宝塔面板**](#2-宝塔面板部署) |
| 小内存 VPS、不想跑 docker | [**二进制 fullstack**](#3-二进制部署-轻量) |
| 想改代码 / 二次开发 | [**源码编译**](#4-从源码编译-开发者) |

---

### 1. Docker Compose 推荐

最简单,5 分钟跑起来。要求: Debian 12 / Ubuntu 22.04+,Docker + Compose plugin。

```bash
# 1. 克隆
git clone https://github.com/TokensZhuanfa/dujiao-shop.git /opt/dujiao-shop
cd /opt/dujiao-shop/deploy

# 2. 改配置 (jwt_secret / 支付通道 / 邮件等)
cp config.template.yml config.yml
vi config.yml

# 3. 一键部署 (装 docker + 起 4 个容器: redis + api + admin + user)
sudo bash deploy.sh

# 4. 浏览器访问
#   admin: http://你的IP:8081
#   user : http://你的IP:80
```

升级:
```bash
cd /opt/dujiao-shop && git pull
cd deploy && docker compose up -d --build
```

---

### 2. 宝塔面板部署

适合已经用宝塔管 nginx 网站的人,不用动 Docker。**整体架构**:

```
┌─────────────────────────────────────────────────────────────┐
│  宿主 Linux (Debian 12 / Ubuntu)                            │
│                                                              │
│  ┌────────────┐  ┌────────────┐                             │
│  │ 站点 1     │  │ 站点 2     │   ← 宝塔面板加 2 个站      │
│  │ user.x.com │  │ admin.x.com│                             │
│  │ :80/443    │  │ :80/443    │                             │
│  └─────┬──────┘  └─────┬──────┘                             │
│        │ /api/ proxy   │ /api/ proxy                        │
│        └───────┬───────┘                                    │
│                ▼                                            │
│        dujiao-api-headless   ──────►  redis-server          │
│        (systemd, 监听 127.0.0.1:8080)  (apt 装, 127.0.0.1)  │
└─────────────────────────────────────────────────────────────┘
```

**前置**:
- 宝塔 7.x+ 已装好 (`/usr/bin/bt` 存在)
- 宝塔内置 nginx
- 准备好 2 个域名指到本机(可后补,先用 IP+端口验证也行)

#### 2.1 装 Redis (二选一)

dujiao-api 需要 redis 做异步队列 + 缓存,绑定 `127.0.0.1:6379`,无密码。两种装法挑一个:

**方法 A — 宝塔商店装(推荐,无坑)**

宝塔面板 → 软件商店 → 搜 `Redis` → 选 6.x 或 7.x → 安装(编译需要 5-10 分钟)。装完默认就监听 `127.0.0.1:6379` 无密码,完全可用。

```bash
# 验证
redis-cli ping     # 应返 PONG
```

> 宝塔商店的 redis 是自编译版本,**不会跟宝塔自带的 libjemalloc 打架**。

**方法 B — `apt` 装(需要修一个冲突)**

```bash
apt update && apt install -y redis-server
```

⚠️ apt 装的 redis 7.x 会被宝塔自带的 `/usr/local/lib/libjemalloc.so.2`(旧版)劫持,启动报 `error while loading shared libraries: libjemalloc.so.2: failed to map segment from shared object`。一行 systemd drop-in 强制用 apt 的 jemalloc 修好:

```bash
mkdir -p /etc/systemd/system/redis-server.service.d
cat > /etc/systemd/system/redis-server.service.d/override.conf <<'EOF'
[Service]
Environment="LD_PRELOAD=/usr/lib/x86_64-linux-gnu/libjemalloc.so.2"
EOF
systemctl daemon-reload
systemctl restart redis-server
redis-cli ping     # 应返 PONG
```

#### 2.2 下载二进制 + 前端 dist

```bash
# unzip 用来解前端 zip
command -v unzip >/dev/null || apt install -y unzip

mkdir -p /www/server/dujiao /www/wwwroot/dujiao-user /www/wwwroot/dujiao-admin
cd /www/server/dujiao

VERSION=v1.0.0   # 看 https://github.com/TokensZhuanfa/Dujiao-Shop/releases 最新版
BASE=https://github.com/TokensZhuanfa/Dujiao-Shop/releases/download/${VERSION}

# headless 二进制 (api 后端,~28MB)
curl -fsSL -O "${BASE}/dujiao-shop-headless_${VERSION}_linux_amd64.tar.gz"
tar xzf dujiao-shop-headless_*.tar.gz
chmod +x dujiao-api-headless admin-tool install.sh

# 前端 dist (静态文件,给宝塔 nginx 托管)
curl -fsSL -o admin.zip "${BASE}/dujiao-shop-admin-${VERSION}.zip"
curl -fsSL -o user.zip  "${BASE}/dujiao-shop-user-${VERSION}.zip"
unzip -q admin.zip -d /www/wwwroot/dujiao-admin/
unzip -q user.zip  -d /www/wwwroot/dujiao-user/
chown -R www:www /www/wwwroot/dujiao-admin /www/wwwroot/dujiao-user

# 数据目录
mkdir -p /www/server/dujiao/{db,uploads,credentials,logs}
```

#### 2.3 生成配置 config.yml

`config.template.yml` 自带 4 个 `__XX__` 占位符,自动生成填充:

```bash
cd /www/server/dujiao
APP_SECRET=$(openssl rand -hex 32)
JWT_SECRET=$(openssl rand -hex 32)
USER_JWT_SECRET=$(openssl rand -hex 32)
ADMIN_PWD="Dj$(openssl rand -base64 18 | tr -d '/+=' | head -c14)1A"

# 备份 secrets (这是首次安装的 admin 密码,务必记下来)
cat > .secrets <<EOF
APP_SECRET=$APP_SECRET
JWT_SECRET=$JWT_SECRET
USER_JWT_SECRET=$USER_JWT_SECRET
ADMIN_PASSWORD=$ADMIN_PWD
EOF
chmod 600 .secrets

# 渲染配置 + 把 docker 默认的 redis host 改成 127.0.0.1 (重要!)
sed -e "s|__APP_SECRET__|$APP_SECRET|g" \
    -e "s|__JWT_SECRET__|$JWT_SECRET|g" \
    -e "s|__USER_JWT_SECRET__|$USER_JWT_SECRET|g" \
    -e "s|__ADMIN_PASSWORD__|$ADMIN_PWD|g" \
    -e "s|^  host: redis$|  host: 127.0.0.1|g" \
    config.template.yml > config.yml
chmod 600 config.yml
```

⚠️ **第 5 行 sed 必须有**:`config.template.yml` 的 `host: redis` 是给 docker-compose 用的(内网 DNS 名),非 Docker 部署下解析失败甚至被某些服务商的反向 DNS 劫持到外部 IP,导致 api 启动后 CPU 100% 跑 redis 重试。

#### 2.4 systemd 拉起 dujiao-api

```bash
cat > /etc/systemd/system/dujiao-api.service <<'EOF'
[Unit]
Description=dujiao-shop API (headless)
After=network-online.target redis-server.service
Wants=network-online.target

[Service]
Type=simple
WorkingDirectory=/www/server/dujiao
ExecStart=/www/server/dujiao/dujiao-api-headless
Restart=always
RestartSec=5
LimitNOFILE=65535
StandardOutput=append:/var/log/dujiao/api.log
StandardError=append:/var/log/dujiao/api.log

[Install]
WantedBy=multi-user.target
EOF

mkdir -p /var/log/dujiao
systemctl daemon-reload
systemctl enable --now dujiao-api
sleep 5
curl -fsS http://127.0.0.1:8080/health    # 应返 {"status":"ok"}
```

> 不想用 systemd?可以走宝塔面板:**软件商店 → 进程守护 (Supervisor)**,新增任务:运行目录 `/www/server/dujiao`,启动命令 `./dujiao-api-headless`,用户 root。systemd 更轻、更稳,推荐它。

#### 2.5 在宝塔面板加 **2 个站点**

进**宝塔面板 → 网站 → 添加站点**,各填一个:

| 字段 | **站点 1 (user 前台)** | **站点 2 (admin 后台)** |
|---|---|---|
| 域名 | `user.your.com` (你的域名) | `admin.your.com` |
| 备注 | dujiao-shop 用户端 | dujiao-shop 管理端 |
| 根目录 | `/www/wwwroot/dujiao-user` | `/www/wwwroot/dujiao-admin` |
| FTP | 不创建 | 不创建 |
| 数据库 | 不创建 | 不创建 |
| PHP 版本 | **纯静态** | **纯静态** |

> 没域名也行:**域名**字段填 `94.16.112.46:8082`(写 IP:端口),宝塔会自动把站点监听 8082,直接用 IP 访问。

#### 2.6 给每站贴**伪静态**(Rewrite)

**宝塔面板 → 网站 → user 站点 → 设置 → 伪静态**,清空原内容,贴下面这一整段:

```nginx
# /api/ 反代到本机 dujiao-api
location /api/ {
    proxy_pass http://127.0.0.1:8080;
    proxy_http_version 1.1;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
    proxy_buffering off;
}

# 上传文件接口需要更大 body
client_max_body_size 100M;

# SPA 路由 fallback (Vue Router history 模式必须)
location / {
    try_files $uri $uri/ /index.html;
}
```

**保存后宝塔会自动 reload nginx**。**admin 站点贴同样一份**(两个站伪静态完全一致)。

#### 2.7 申请 SSL (有域名才用)

每个站 → **SSL → Let's Encrypt → 申请**。两站各申一次,**勾"强制 HTTPS"**。

#### 2.8 验证 + 浏览器登录

```bash
# 后端探活
curl http://127.0.0.1:8080/health             # {"status":"ok"}
curl http://127.0.0.1:8080/api/v1/public/config | head -c 300  # 看 app_version

# 前端走宝塔 nginx
curl -I http://user.your.com/                  # 200
curl -I http://admin.your.com/                 # 200

# 浏览器打开 https://admin.your.com → 登录
#   用户名: admin
#   密码:   见 /www/server/dujiao/.secrets ADMIN_PASSWORD
```

#### 2.9 运维 CLI

```bash
# 列管理员
/www/server/dujiao/admin-tool list-admins

# 重置某管理员的 2FA (TOTP 丢失场景)
/www/server/dujiao/admin-tool reset-2fa --username admin
```

#### 2.10 升级到新版

```bash
cd /www/server/dujiao
VERSION=v1.0.1   # 改成最新 tag
curl -fsSL -O https://github.com/TokensZhuanfa/Dujiao-Shop/releases/download/${VERSION}/dujiao-shop-headless_${VERSION}_linux_amd64.tar.gz
tar xzf dujiao-shop-headless_*.tar.gz
chmod +x dujiao-api-headless admin-tool
systemctl restart dujiao-api

# 前端
curl -fsSL -o /tmp/admin.zip https://github.com/TokensZhuanfa/Dujiao-Shop/releases/download/${VERSION}/dujiao-shop-admin-${VERSION}.zip
curl -fsSL -o /tmp/user.zip  https://github.com/TokensZhuanfa/Dujiao-Shop/releases/download/${VERSION}/dujiao-shop-user-${VERSION}.zip
unzip -qo /tmp/admin.zip -d /www/wwwroot/dujiao-admin/
unzip -qo /tmp/user.zip  -d /www/wwwroot/dujiao-user/
chown -R www:www /www/wwwroot/dujiao-{user,admin}

# config.yml + 数据库 + 上传文件不会动
```

---

### 3. 二进制部署 轻量

适合不想装 docker 的小内存 VPS。**单文件全栈**,内嵌 admin/user 前端,**无需 nginx**。

```bash
# 1. 下载最新 release 的 fullstack 版
curl -L -O https://github.com/TokensZhuanfa/dujiao-shop/releases/latest/download/dujiao-shop-fullstack_v1.0.0_linux_amd64.tar.gz

# 2. 解压 + 安装 (装 systemd unit + 系统用户 dujiao)
tar xzf dujiao-shop-fullstack_*.tar.gz
cd dujiao-shop-fullstack_*/
sudo ./install.sh

# 3. 改配置
sudo vi /opt/dujiao/config.yml

# 4. 启服务
sudo systemctl enable --now dujiao
sudo journalctl -u dujiao -f

# 5. 浏览器访问 http://你的IP:8080
#    admin 界面在 /admin/  user 界面在 /
```

ARM 服务器换成 `linux_arm64` 包,其他步骤一致。详见 [`deploy/release/README-binary.md`](deploy/release/README-binary.md)。

> 想用 nginx 自己托管前端? 下载 `dujiao-shop-headless_*` 包 + `dujiao-shop-admin-*.zip` + `dujiao-shop-user-*.zip`。

---

### 4. 从源码编译 开发者

要求: Go 1.25+, Node.js 24+, Redis 7+。

```bash
git clone https://github.com/TokensZhuanfa/dujiao-shop.git
cd dujiao-shop

# api
cd src/api
go run ./cmd/server                  # 开发模式
# 或 build:
go build -tags release -o dujiao-api ./cmd/server

# admin (默认走独立 nginx)
cd ../admin && npm ci && npm run dev      # localhost:5173
# 或 build:
npm run build                              # 产物在 dist/

# user
cd ../user && npm ci && npm run dev       # localhost:5174
```

想编出**单文件全栈**(api 内嵌前端):
```bash
cd src/admin && npm run build:fullstack && cd -
cd src/user  && npm run build              && cd -
mkdir -p src/api/internal/web/dist
cp -r src/admin/dist src/api/internal/web/dist/admin
cp -r src/user/dist  src/api/internal/web/dist/user
cd src/api && go build -tags 'release fullstack' -o dujiao-api ./cmd/server
```

---

## 📁 项目结构

```
.
├── src/
│   ├── api/      # Go 后端 (Gin + GORM + SQLite + Redis + Asynq)
│   ├── admin/    # 后台 Vue 3 + Vite
│   └── user/     # 前台 Vue 3 + Vite
├── deploy/
│   ├── docker-compose.yml          # 容器编排
│   ├── deploy.sh                   # Debian 一键脚本
│   ├── config.template.yml         # 配置模板
│   ├── admin-nginx.conf / user-nginx.conf
│   └── release/
│       ├── install.sh              # 二进制版安装脚本
│       ├── dujiao.service          # systemd unit
│       └── README-binary.md        # 二进制版安装说明
└── .github/workflows/release.yml   # tag v* 自动出 release
```

## 🔄 上游同步

monorepo 之前是三个子仓 (`dujiao-next/{dujiao-next,admin,user}`),现在合并到本仓。同步上游:

```bash
git remote add upstream-api   https://github.com/dujiao-next/dujiao-next.git
git remote add upstream-admin https://github.com/dujiao-next/admin.git
git remote add upstream-user  https://github.com/dujiao-next/user.git

git fetch upstream-api   main && git subtree pull --prefix=src/api   upstream-api   main -m "merge: upstream api"
git fetch upstream-admin main && git subtree pull --prefix=src/admin upstream-admin main -m "merge: upstream admin"
git fetch upstream-user  main && git subtree pull --prefix=src/user  upstream-user  main -m "merge: upstream user"
```

> 当前 monorepo 基于 dujiao-next/dujiao-next@faacb3f、admin@5d0ce38、user@6b67ef9 (2026-05-14)。

## 🧑‍💻 二次开发要点

- **Codex 号池**: 后台 `号池管理 → Codex`,管理 OpenAI/ChatGPT 账号; 商品 `auto_secret_kind=codex_pool` 自动对接。源码在 `src/api/internal/service/codex_account_service.go`
- **号池预占**: 下单事务里原子占住账号(`reserved_order_id`),付款转 sold,超时/取消归还(对齐文本卡密 reservation 语义,见 `order_service.go:488`)
- **CpaMC / Sub2api 双格式**: 买家订单页可单账号或全部打包下载,`fulfillment_service.go`

## 📦 发布流程

```bash
# tag 一个新版本,推到 GitHub
git tag v1.2.3 && git push origin v1.2.3

# GitHub Actions 自动:
#   1. build admin/user 前端
#   2. 把 dist 嵌入 api 的 internal/web/
#   3. goreleaser 出:
#      - dujiao-shop-fullstack_v1.2.3_linux_amd64.tar.gz
#      - dujiao-shop-fullstack_v1.2.3_linux_arm64.tar.gz
#      - dujiao-shop-headless_v1.2.3_linux_amd64.tar.gz
#      - dujiao-shop-headless_v1.2.3_linux_arm64.tar.gz
#      - dujiao-shop-admin-v1.2.3.zip   (前端 dist)
#      - dujiao-shop-user-v1.2.3.zip
#      - checksums.txt
#   4. 上传到 https://github.com/TokensZhuanfa/dujiao-shop/releases/tag/v1.2.3
```

## 📜 License

MIT,参见 [LICENSE](LICENSE)。基于 [dujiao-next](https://github.com/dujiao-next) 修改与扩展。
