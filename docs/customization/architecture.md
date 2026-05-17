# 项目结构

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
├── docs/                           # 本文档站 (VitePress)
└── .github/workflows/
    ├── release.yml                 # tag v* 自动出 release
    └── docs.yml                    # docs/ 改动自动发到 GitHub Pages
```

## 主要二次开发

| 改造 | 实现位置 | 详细 |
|---|---|---|
| **Codex 号池** | `src/api/internal/service/codex_account_service.go` | 自动 token 轮换 + 额度刷新 + 状态识别 |
| **号池型商品** | `src/api/internal/service/order_service.go:488` | 下单事务里原子预占账号,付款转 sold |
| **文件型卡密** | `src/api/internal/service/fulfillment_service.go` | 1GB 内文件,买家订单页可下载 |
| **CpaMC / Sub2api 双格式** | `src/admin/src/views/orders/CodexAccountsModal.vue` | 单账号或全部打包下载 |
| **订单号 base32 高熵** | `src/api/internal/utils/orderno.go` | 防扫单遍历 |
| **JWT + 2FA** | `src/api/internal/service/auth_service.go` | TOTP + recovery codes |

## 关键约定

- **数据库**: 默认 SQLite(`./db/dujiao.db`),可换 MySQL
- **缓存/队列**: Redis(异步队列用 [Asynq](https://github.com/hibiken/asynq))
- **JWT secret**: 必须够长,推荐 `openssl rand -base64 32`
- **bcrypt cost**: 默认 10,改密前要确认服务器 CPU 跑得动
