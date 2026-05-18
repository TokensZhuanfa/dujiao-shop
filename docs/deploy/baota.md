# 宝塔面板部署

适合已经用宝塔管 nginx 网站的人,不用动 Docker。

**整体架构**:

```
┌─────────────────────────────────────────────────────────────┐
│  宿主 Linux (Debian 12 / Ubuntu)                            │
│                                                              │
│  ┌────────────┐  ┌────────────┐                             │
│  │ 站点 ①     │  │ 站点 ②     │   ← 宝塔面板加 2 个站      │
│  │ user.x.com │  │ admin.x.com│                             │
│  │ :80/443    │  │ :80/443    │                             │
│  └─────┬──────┘  └─────┬──────┘                             │
│        │ /api/ proxy   │ /api/ proxy                        │
│        └───────┬───────┘                                    │
│                ▼                                            │
│        dujiao-api-headless   ──────►  redis-server          │
│        (systemd, 监听 127.0.0.1:8080)  (apt / 宝塔商店)     │
└─────────────────────────────────────────────────────────────┘
```

**前置**:
- 宝塔 7.x+ 已装好(`/usr/bin/bt` 存在)
- 宝塔内置 nginx
- 准备 2 个域名指到本机(没域名也可以,看 [步骤 2](#_2-在宝塔面板加-2-个站点) 里的备选)

整套流程的逻辑顺序:**先在宝塔加站点(空容器)→ 往每站塞前端 dist → 装后端 → 配 nginx 反代 → SSL → 上线**。

---

## 1. 装 Redis (二选一)

dujiao-api 需要 redis 做异步队列 + 缓存,绑定 `127.0.0.1:6379`,无密码。挑一个:

### 方法 A — 宝塔商店装 (推荐,无坑)

宝塔面板 → **软件商店** → 搜 `Redis` → 选 6.x 或 7.x → 安装(编译需要 5-10 分钟)。装完默认就监听 `127.0.0.1:6379`,完全可用。

```bash
redis-cli ping     # PONG
```

> 宝塔编译的 redis 自带配套 jemalloc,不会跟系统库打架。

### 方法 B — `apt` 装 (需要修一个冲突)

```bash
apt update && apt install -y redis-server
```

⚠️ apt 装的 redis 7.x 会被宝塔自带的 `/usr/local/lib/libjemalloc.so.2`(旧版)劫持,启动报 `error while loading shared libraries: libjemalloc.so.2: failed to map segment from shared object`。一行 systemd drop-in 修好:

```bash
mkdir -p /etc/systemd/system/redis-server.service.d
cat > /etc/systemd/system/redis-server.service.d/override.conf <<'EOF'
[Service]
Environment="LD_PRELOAD=/usr/lib/x86_64-linux-gnu/libjemalloc.so.2"
EOF
systemctl daemon-reload
systemctl restart redis-server
redis-cli ping     # PONG
```

---

## 2. 在宝塔面板加 **2 个站点**

**先把两个"空站点容器"建好**,后面再把对应的前端文件塞进去。

进宝塔面板 → **网站 → 添加站点**,分别建:

| 字段 | **站点 ① 用户前台 (user)** | **站点 ② 管理后台 (admin)** |
|---|---|---|
| 用途 | 顾客访问,浏览 / 下单 / 付款 | 你自己进,管理商品 / 订单 / 卡密 |
| 域名 | `user.your.com`(你的主域名) | `admin.your.com`(后台子域) |
| 备注 | dujiao-shop 用户端 | dujiao-shop 管理端 |
| **根目录** | **`/www/wwwroot/dujiao-user`** | **`/www/wwwroot/dujiao-admin`** |
| FTP | 不创建 | 不创建 |
| 数据库 | **不创建**(数据库在后端 SQLite 里) | **不创建** |
| **PHP 版本** | **纯静态**(前端是 HTML/JS,不跑 PHP) | **纯静态** |

保存后宝塔会自动生成两个**空目录**:`/www/wwwroot/dujiao-user/`、`/www/wwwroot/dujiao-admin/`。下一步往里塞东西。

> **没域名时的替代**:`域名` 字段填 `94.16.112.46:8082`(IP+端口),宝塔会让站点监听 8082;admin 站用 8083。直接用 `http://你的IP:8082` 访问。

---

## 3. 往每个站点下载对应前端 dist

`unzip` 没装的先装:

```bash
command -v unzip >/dev/null || apt install -y unzip
```

> 下面的 URL 都基于 **v1.0.0**(当前最新)。升级新版时,把 URL 里的 `v1.0.0` 整体替换成新 tag(如 `v1.0.1`)即可,文件名规则不变。最新 tag 看 [Releases 页面](https://github.com/TokensZhuanfa/Dujiao-Shop/releases)。

### 3.1 站点 ① user 站 ← 用户前台 dist

```bash
curl -fsSL -o /tmp/user.zip https://github.com/TokensZhuanfa/Dujiao-Shop/releases/download/v1.0.0/dujiao-shop-user-v1.0.0.zip
unzip -q /tmp/user.zip -d /www/wwwroot/dujiao-user/
chown -R www:www /www/wwwroot/dujiao-user
ls /www/wwwroot/dujiao-user   # 应看到 index.html + assets/
```

### 3.2 站点 ② admin 站 ← 管理后台 dist

```bash
curl -fsSL -o /tmp/admin.zip https://github.com/TokensZhuanfa/Dujiao-Shop/releases/download/v1.0.0/dujiao-shop-admin-v1.0.0.zip
unzip -q /tmp/admin.zip -d /www/wwwroot/dujiao-admin/
chown -R www:www /www/wwwroot/dujiao-admin
ls /www/wwwroot/dujiao-admin
```

::: warning v1.0.0 已知问题
`dujiao-shop-admin-v1.0.0.zip` 里 HTML/JS 含 fullstack 模式的占位符 `__DJ_ADMIN_BASE__`,nginx 静态托管会导致**打开 admin 白屏**(F12 看到 URL 含 `__DJ_ADMIN_BASE__` 全部 404)。

**workaround**(v1.0.0 解压后立刻跑):

```bash
find /www/wwwroot/dujiao-admin -type f \( -name '*.html' -o -name '*.js' -o -name '*.css' \) \
  -exec sed -i 's|__DJ_ADMIN_BASE__||g' {} +
```

v1.0.1 起本 zip 已修(CI 改 build 两次,普通 build 不含占位符),无需此 workaround。
:::

现在两个站点已经"有 HTML 文件"了,但还没接后端,**直接访问会显示空白**(SPA 启动后第一件事就是请求 `/api/`,拿不到就卡住)。下一步装后端。

---

## 4. 装后端二进制 (headless 版)

后端跑在**站点外**的独立目录 `/www/server/dujiao/`,只对内监听 `127.0.0.1:8080`,让两个站的 nginx 反代到它。

```bash
mkdir -p /www/server/dujiao /var/log/dujiao
cd /www/server/dujiao

# 后端二进制(~28 MB,含 admin-tool + install.sh + config 模板)
curl -fsSL -O https://github.com/TokensZhuanfa/Dujiao-Shop/releases/download/v1.0.0/dujiao-shop-headless_v1.0.0_linux_amd64.tar.gz
tar xzf dujiao-shop-headless_*.tar.gz
chmod +x dujiao-api-headless admin-tool install.sh

# 数据目录(api 写 SQLite / 上传 / 卡密 / 日志)
mkdir -p db uploads credentials logs

ls -la   # 应看到 dujiao-api-headless / admin-tool / config.template.yml / install.sh ...
```

> ARM 服务器把 URL 里 `linux_amd64` 改成 `linux_arm64`。

**3 个产物各放各位置,小结**:

| 文件 | 解压到 | 用途 |
|---|---|---|
| `dujiao-shop-user-*.zip` | `/www/wwwroot/dujiao-user/` | 用户前台 SPA |
| `dujiao-shop-admin-*.zip` | `/www/wwwroot/dujiao-admin/` | 管理后台 SPA |
| `dujiao-shop-headless_*_linux_amd64.tar.gz` | `/www/server/dujiao/` | 后端二进制 |

---

## 5. 生成 `config.yml`

`config.template.yml` 自带 4 个 `__XX__` 占位符,自动渲染填充:

```bash
cd /www/server/dujiao
APP_SECRET=$(openssl rand -hex 32)
JWT_SECRET=$(openssl rand -hex 32)
USER_JWT_SECRET=$(openssl rand -hex 32)
ADMIN_PWD="Dj$(openssl rand -base64 18 | tr -d '/+=' | head -c14)1A"

# 备份 secrets (这是首次登录的 admin 密码,务必记下来)
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

⚠️ **第 5 行 sed 必须有**:`config.template.yml` 里 `host: redis` 是给 docker-compose 用的(内网 DNS 名),宝塔部署里这个主机名解析失败,导致 api 启动后 CPU 100% 跑 redis 重试。详见 [config.yml 详解 → redis](/config/yaml#redis-queue)。

---

## 6. systemd 拉起 dujiao-api

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

systemctl daemon-reload
systemctl enable --now dujiao-api
sleep 5
curl -fsS http://127.0.0.1:8080/health    # 应返 {"status":"ok"}
```

> 也可以用宝塔的"软件商店 → 进程守护 (Supervisor)"。systemd 更轻、更稳,本文用它。

---

## 7. 给**每个**站贴**同一份**伪静态(Rewrite)

> **两个站伪静态完全一样**,因为它们都要做同样的两件事:把 `/api/*` 反代到 `127.0.0.1:8080`,其他 URL 走 SPA fallback。站点的不同已经在 [步骤 2](#_2-在宝塔面板加-2-个站点) 的 `root` 字段处理过了。

**操作**:宝塔面板 → 网站 → **每个站点逐个点进去** → **设置 → 伪静态** → 清空原内容 → 贴下面**这一整段一字不差**:

```nginx
# /api/ 反代到本机 dujiao-api (后端二进制监听 127.0.0.1:8080)
location /api/ {
    proxy_pass http://127.0.0.1:8080;
    proxy_http_version 1.1;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
    proxy_buffering off;
}

# /uploads/ 让 nginx 直接 serve api 工作目录里的图片
# (商品图 / Logo / Banner / 卡密图等都落在这,不反代去走 api 更高效, 还能 30 天缓存)
# ⚠️ 必须用 `^~` 前缀,优先级高于宝塔默认带的 `\.(jpg|png|gif)$` regex location;
#    用普通 `location /uploads/` 会被 regex 抢走 → 站点根目录找不到 → 404
location ^~ /uploads/ {
    alias /www/server/dujiao/uploads/;
    try_files $uri =404;
    expires 30d;
    add_header Cache-Control "public, immutable";
}

# 上传文件接口需要更大 body (商品图、卡密文件)
client_max_body_size 100M;

# Vue Router history 模式:任何前端路由 fallback 到 /index.html
location / {
    try_files $uri $uri/ /index.html;
}
```

保存后宝塔自动 reload nginx。**两个站都贴一遍,内容完全相同**。

::: tip 关于 /uploads/ 权限
nginx 默认用 `www` 用户运行,但 `/www/server/dujiao/uploads/` 是 root 写的。
如果浏览器访问 `/uploads/xxx.png` 返 403:

```bash
chmod -R o+rX /www/server/dujiao/uploads
```
:::

保存后宝塔自动 reload nginx。**两个站都贴一遍,内容完全相同**。

---

## 8. 申请 SSL (有域名才用)

每个站点 → **SSL → Let's Encrypt → 申请**。两站各申一次,**勾"强制 HTTPS"**。

> 没域名跳过本步,直接用 `http://IP:8082`(user)/ `http://IP:8083`(admin)访问。

---

## 9. 验证 + 浏览器登录

```bash
# 后端探活
curl http://127.0.0.1:8080/health                                # {"status":"ok"}
curl http://127.0.0.1:8080/api/v1/public/config | head -c 300    # 看 app_version

# 前端走宝塔 nginx
curl -I http://user.your.com/                                    # 200
curl -I http://admin.your.com/                                   # 200

# 浏览器打开 https://admin.your.com → 登录
#   用户名: admin
#   密码:   见 /www/server/dujiao/.secrets 的 ADMIN_PASSWORD
```

---

## 10. 运维 CLI

```bash
# 列管理员
/www/server/dujiao/admin-tool list-admins

# 重置某管理员的 2FA (TOTP 丢失场景)
/www/server/dujiao/admin-tool reset-2fa --username admin
```

---

## 11. 升级到新版

后端二进制 + 前端 dist 分开升。把下面所有 URL 里的 `v1.0.0` 整体改成新 tag(如 `v1.0.1`):

```bash
cd /www/server/dujiao

# 后端
curl -fsSL -O https://github.com/TokensZhuanfa/Dujiao-Shop/releases/download/v1.0.0/dujiao-shop-headless_v1.0.0_linux_amd64.tar.gz
tar xzf dujiao-shop-headless_*.tar.gz
chmod +x dujiao-api-headless admin-tool
systemctl restart dujiao-api

# 前端 (覆盖式解压)
curl -fsSL -o /tmp/admin.zip https://github.com/TokensZhuanfa/Dujiao-Shop/releases/download/v1.0.0/dujiao-shop-admin-v1.0.0.zip
curl -fsSL -o /tmp/user.zip  https://github.com/TokensZhuanfa/Dujiao-Shop/releases/download/v1.0.0/dujiao-shop-user-v1.0.0.zip
unzip -qo /tmp/admin.zip -d /www/wwwroot/dujiao-admin/
unzip -qo /tmp/user.zip  -d /www/wwwroot/dujiao-user/
chown -R www:www /www/wwwroot/dujiao-{user,admin}

# config.yml + 数据库 + 上传文件全部保留不动
```
