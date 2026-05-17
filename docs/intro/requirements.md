# 环境要求

每种部署方式对环境的依赖不一样,挑你打算用的看对应章节。

## 通用硬件参考

不管哪种部署方式,机器都要满足:

| 项 | 最低 | 推荐 |
|---|---|---|
| OS | Linux x86_64 / arm64 | Debian 12 / Ubuntu 22.04+ |
| CPU | 1 核 | 2 核 |
| RAM | 1 GB(开 4 GB swap) | 2 GB+ |
| 磁盘 | 5 GB | 30 GB+(看商品图 + 卡密文件量) |
| 网络 | 公网 IP / 域名 | 加 Cloudflare 反代防爬 |

实测 1 GB RAM 机能稳定跑 1k 日订单 + 几百个号池账号,**只要 swap 给够**。

防火墙最小开放(任何方式都一样):

```bash
ufw default deny incoming
ufw allow 22/tcp        # SSH
ufw allow 80/tcp        # HTTPS redirect
ufw allow 443/tcp       # HTTPS
ufw enable
```

## Docker Compose 部署

参考 [Docker Compose 部署](/deploy/docker-compose)。需要装:

- **Docker** 24+
- **Docker Compose plugin** v2
- 不需要单独装 nginx / redis(都是容器内置)
- 不需要 Go / Node 工具链(build 在容器内完成)

RAM 至少 **2 GB**,build admin / user 前端时单容器峰值会到 1.5 GB,小内存机请先把 swap 开到 4 GB。

```bash
# 一行装 docker 全套
curl -fsSL https://get.docker.com | bash
systemctl enable --now docker
docker --version
docker compose version
```

## 宝塔面板部署

参考 [宝塔面板部署](/deploy/baota)。需要:

- **宝塔面板 7.x+** 已装好(`/usr/bin/bt` 存在 / 浏览器能访问 :8888 后台)
- 宝塔自带的 **nginx**(无需另装)
- **Redis 6.x / 7.x**,有两条路:
  - 宝塔商店装(推荐,**无 libjemalloc 冲突**)
  - `apt install redis-server`(需要一行 systemd drop-in 修 libjemalloc,详见部署页)
- 不需要 Docker / Go / Node
- 2 个域名(或 2 个 IP+端口),分别给 user 前台 + admin 后台

## 单二进制 fullstack 部署

参考 [单二进制部署](/deploy/single-binary)。**最省事的部署方式**:

- 任意 64 位 Linux(amd64 / arm64)
- **Redis 6.x / 7.x**(本机 `apt install redis-server` 就行)
- **systemd**(`install.sh` 自动装 unit)
- 不需要 nginx(可选;裸跑 8080 也行,生产建议前面套 nginx + HTTPS)
- 不需要 Go / Node(下载预编译 tarball)
- 不需要宝塔 / Docker

如果想前面套 nginx 加 HTTPS,自装 nginx 或借现成的反代(Caddy / Cloudflare Tunnel)。

## 源码编译(开发者)

参考 [手动部署](/deploy/manual)。需要完整工具链:

- **Go 1.25+**(`go build -tags release fullstack`)
- **Node.js 24+** + **npm 10+**(`vite build` 前端)
- **Redis 6.x / 7.x**
- **git** 2.x
- 操作系统:Linux / macOS / Windows + WSL 都行

dev 模式跑前端会自动起 vite dev server(localhost:5173 / 5174),后端 `go run ./cmd/server` 起在 8080。生产 build 走 [发布流程](/intro/changelog#发布流程) 走 GoReleaser + GitHub Actions。

## 数据库选择(所有方式通用)

| 数据库 | 适合 | 备注 |
|---|---|---|
| **SQLite**(默认) | < 5 万订单 | 部署最简,单文件,无需另装 |
| **MySQL 8** | > 5 万订单或多副本 | 单独装,见 [备份与恢复 → 数据库迁移](/ops/backup-restore#数据库迁移sqlite--mysql) |
| PostgreSQL | 实验性 | 同上 |

如果不确定,**选 SQLite**。后期数据上量再迁。
