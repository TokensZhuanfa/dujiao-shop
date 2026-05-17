# 环境要求

不同部署方式对环境的要求差别很大,按你的目标方式看对应列。

## 整体硬件参考

| 项 | 最低 | 推荐 |
|---|---|---|
| OS | Linux x86_64 / arm64 | Debian 12 / Ubuntu 22.04+ |
| CPU | 1 核 | 2 核 |
| RAM | 1 GB(开 4 GB swap) | 2 GB+ |
| 磁盘 | 5 GB | 30 GB+(看商品图 + 卡密文件量) |
| 网络 | 公网 IP / 域名 | 加 Cloudflare 反代防爬 |

实测 1 GB RAM 机能稳定跑 1k 日订单 + 几百个号池账号,**只要 swap 给够**。

## 按部署方式区分

|  | Docker Compose | 宝塔面板 | 单二进制 fullstack | 源码编译 |
|---|---|---|---|---|
| Docker + Compose plugin | ✅ 必须 | ❌ 不用 | ❌ 不用 | ❌ 不用 |
| nginx | ❌ 容器内置 | ✅ 宝塔自带 | ❌ 二进制内含 | ✅ dev 用 vite |
| Redis | ❌ 容器内置 | ✅ 宝塔商店 / apt | ✅ apt 或自编 | ✅ 自装 |
| systemd | ❌ | ✅ 装 api 用 | ✅ install.sh 自动 | 自选 |
| 宝塔面板 | ❌ | ✅ 必须装好 | ❌ | ❌ |
| Go 工具链 | ❌ build 在容器内 | ❌ 用 release tarball | ❌ | ✅ **Go 1.25+** |
| Node.js | ❌ | ❌ | ❌ | ✅ **Node 24+** |
| openssl | 推荐 | 推荐 | 推荐 | 推荐 |

## Docker Compose

- Debian 12 / Ubuntu 22.04+(`deploy.sh` 默认按 Debian 12 适配)
- Docker 24+ 且 Compose plugin V2
- RAM 至少 2 GB(单容器 build 时峰值 1.5 GB),小机要 swap

## 宝塔面板

- 宝塔 7.x+ 已装(`/usr/bin/bt` 存在)
- 宝塔自带的 nginx + redis(或 apt 装 redis,但要修 libjemalloc 冲突,见 [宝塔部署](/deploy/baota))
- 域名 / SSL 都通过宝塔面板申

## 单二进制 fullstack

- 任意 64 位 Linux(amd64 / arm64)
- 不需要 nginx / Go / Node 任何编译时依赖
- 唯一外部依赖是 **Redis**,本机 apt 装即可

## 源码编译(开发)

- **Go 1.25+**(`go build -tags release fullstack`)
- **Node.js 24+**(`vite build` 前端)
- Redis 任意版本 ≥ 6
- Linux / macOS / Windows + WSL 都能跑

## 数据库选择(任何部署方式都适用)

| 数据库 | 适合 | 备注 |
|---|---|---|
| **SQLite**(默认) | < 5 万订单 | 部署最简,单文件 |
| **MySQL 8** | > 5 万订单或多副本 | 需要单独装,见 [备份与恢复 → 数据库迁移](/ops/backup-restore) |
| PostgreSQL | 实验性 | 同上 |

## 防火墙(任何部署方式)

| 端口 | 用途 | 公网开放? |
|---|---|---|
| 22 | SSH | ✓ |
| 80 | HTTPS redirect | ✓ |
| 443 | HTTPS | ✓ |
| 8080 | api 直连(裸 IP 访问) | ❌(被 nginx 代理) |
| 6379 | Redis | ❌(必须 127.0.0.1 绑定) |

```bash
# ufw 标准最小集
ufw default deny incoming
ufw allow 22/tcp
ufw allow 80/tcp
ufw allow 443/tcp
ufw enable
```
