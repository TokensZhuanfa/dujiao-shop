# 环境要求

dujiao-shop 设计成尽可能轻量。最小可跑环境:

## 系统

| 项 | 最低 | 推荐 |
|---|---|---|
| OS | Linux x86_64 / arm64 | Debian 12 / Ubuntu 22.04+ |
| CPU | 1 核 | 2 核 |
| RAM | 1 GB(开 4 GB swap)| 2 GB+ |
| 磁盘 | 5 GB | 30 GB+(看商品图 + 卡密文件量)|
| 网络 | 公网 IP / 域名 | 加 Cloudflare 反代防爬 |

实测 1 GB RAM 机能稳定跑 1k 日订单 + 几百个号池账号,只要 swap 给够。

## 必备依赖

| 依赖 | 版本 | 用途 |
|---|---|---|
| Redis | 6.x / 7.x | 缓存 + 异步任务队列 (asynq) |
| nginx | 1.20+ | 反代 / SSL 终止 (Docker / 二进制部署都用) |
| openssl | 1.1+ | 生成 secret + Let's Encrypt |

## 编译时依赖(只有从源码编译需要)

| 依赖 | 版本 |
|---|---|
| Go | 1.25+ |
| Node.js | 24+ |
| npm | 10+ |

直接下 release tarball 的话**不需要** Go/Node。

## 数据库选择

| 数据库 | 适合 | 备注 |
|---|---|---|
| **SQLite** (默认) | < 5 万订单 | 部署最简,单文件 |
| **MySQL 8** | > 5 万订单或多副本 | 需要单独装,见 [备份与恢复 → 数据库迁移](/ops/backup-restore) |
| PostgreSQL | 实验性 | 同上 |

## 防火墙

| 端口 | 用途 | 公网开放? |
|---|---|---|
| 22 | SSH | ✓ |
| 80 | HTTPS redirect | ✓ |
| 443 | HTTPS | ✓ |
| 8080 | api 直连(裸 IP 访问)| ❌(被 nginx 代理) |
| 6379 | Redis | ❌(必须 127.0.0.1 绑定) |

```bash
# ufw 标准最小集
ufw default deny incoming
ufw allow 22/tcp
ufw allow 80/tcp
ufw allow 443/tcp
ufw enable
```
