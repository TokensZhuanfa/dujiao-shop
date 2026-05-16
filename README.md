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

适合已经在用宝塔管 PHP / nginx 网站的人。不用动 docker、不写 systemd,完全在面板里操作。

**前置**：宝塔 7.x+,已装好 nginx 和 redis(应用商店搜"Redis"一键装)。

#### 2.1 准备文件

```bash
# 1. 上传二进制 (headless 版,前端给宝塔站点托管)
mkdir -p /www/server/dujiao && cd /www/server/dujiao
curl -L -o pkg.tar.gz https://github.com/TokensZhuanfa/dujiao-shop/releases/latest/download/dujiao-shop-headless_v1.0.0_linux_amd64.tar.gz
tar xzf pkg.tar.gz && rm pkg.tar.gz

# 2. 准备 admin / user 前端 dist
cd /www/wwwroot && mkdir -p dujiao-admin dujiao-user
curl -L -o /tmp/admin.zip https://github.com/TokensZhuanfa/dujiao-shop/releases/latest/download/dujiao-shop-admin-v1.0.0.zip
curl -L -o /tmp/user.zip  https://github.com/TokensZhuanfa/dujiao-shop/releases/latest/download/dujiao-shop-user-v1.0.0.zip
unzip -q /tmp/admin.zip -d dujiao-admin/
unzip -q /tmp/user.zip  -d dujiao-user/
chown -R www:www dujiao-admin dujiao-user
```

#### 2.2 改配置 + 启 api

```bash
cp /www/server/dujiao/config.template.yml /www/server/dujiao/config.yml
vi /www/server/dujiao/config.yml
# 至少改:
#   jwt_secret  →  openssl rand -base64 32
#   redis.address: 127.0.0.1:6379
#   server.port: 8080
```

把 api 挂到宝塔进程守护:**软件商店 → 系统加固 → 进程守护 (Supervisor)**,新增:

| 字段 | 值 |
|---|---|
| 名称 | dujiao-api |
| 启动用户 | www |
| 运行目录 | `/www/server/dujiao` |
| 启动命令 | `./dujiao-api-headless` |
| 进程数量 | 1 |
| 自启 | ✓ |

保存后点"启动",到日志页看是否正常监听 8080。

#### 2.3 建两个网站 (admin / user)

**宝塔 → 网站 → 添加站点**,各建一个,PHP 选**纯静态**:

| | admin 站 | user 站 |
|---|---|---|
| 域名 | `admin.your.com` | `your.com` |
| 根目录 | `/www/wwwroot/dujiao-admin` | `/www/wwwroot/dujiao-user` |
| PHP 版本 | 纯静态 | 纯静态 |

然后给每个站**伪静态(Rewrite)**贴这套规则,把 api 路径反代到 8080,其他走 SPA:

```nginx
# admin / user 站通用伪静态
location /api/ {
    proxy_pass http://127.0.0.1:8080;
    proxy_http_version 1.1;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
    proxy_buffering off;
    client_max_body_size 100M;
}

# SPA 路由 fallback
location / {
    try_files $uri $uri/ /index.html;
}
```

#### 2.4 SSL

宝塔站点 → SSL → 用 Let's Encrypt 一键申请,**记得勾"强制 HTTPS"**。两个站各申一次。

#### 2.5 验证 + 收尾

```bash
# api 进程
ps aux | grep dujiao-api-headless
curl -fsSL http://127.0.0.1:8080/health   # 应返 OK

# 浏览器访问 https://admin.your.com 看到登录页 = 成功
```

**运维 CLI**:
```bash
sudo -u www /www/server/dujiao/admin-tool list-admins
sudo -u www /www/server/dujiao/admin-tool reset-2fa --username admin
```

**升级**:替换 `dujiao-api-headless` 二进制后,在宝塔 Supervisor 点"重启"。前端站升级则覆盖 `/www/wwwroot/{admin,user}/` 即可。

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
