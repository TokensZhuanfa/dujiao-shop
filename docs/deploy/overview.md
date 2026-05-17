# 部署总览

dujiao-shop 支持 4 种部署方式,按你的场景选:

| 场景 | 推荐 | 优点 | 缺点 |
|---|---|---|---|
| 不熟运维 / 想 5 分钟跑起来 | [Docker Compose](/deploy/docker-compose) | 一键起 4 个容器,环境隔离 | 吃 RAM(docker 自身 200MB+) |
| 已用宝塔面板管站 | [宝塔面板](/deploy/baota) | 复用现有 nginx / SSL,面板友好 | 跟其他站共用 nginx,需调端口 |
| 小内存 VPS / 想最轻量 | [单二进制 fullstack](/deploy/single-binary) | 单文件全栈,~70MB RAM | 升级要手动覆盖 |
| 想改代码 / 二次开发 | [源码编译](/deploy/manual) | 完全可控 | 需要 Go + Node 工具链 |

## 资源消耗对比

| 部署方式 | api 进程 | redis | nginx | docker 自身 | 合计 RAM |
|---|---|---|---|---|---|
| Docker Compose | ~80 MB | ~30 MB | ~10 MB (user/admin 各一) | ~200 MB | **~330 MB** |
| 宝塔 headless | ~80 MB | ~30 MB | (共用宝塔 nginx) | 0 | **~110 MB** |
| 单二进制 fullstack | ~80 MB(含前端)| ~30 MB | 0 | 0 | **~110 MB** |
| 源码 dev | ~80 MB | ~30 MB | (vite dev server)| 0 | ~150 MB |

## 部署后必做

不管哪种方式,装完都需要:

1. **改 secret 三件套**:见 [config.yml 详解 → 必改的 5 项](/config/yaml#-必改的-5-项-新装必看)
2. **接支付通道**:[支付配置与回调指南](/payment/setup)
3. **首次登录改密码**:用 `bootstrap.default_admin_password` 进后台后立刻改 + 启 2FA
4. **配 SMTP**:邮件验证码 / 找回密码 / 订单通知
5. **配备份**:[备份与恢复](/ops/backup-restore)
6. **过一遍安全清单**:[安全最佳实践](/guide/security)

## 升级与迁移

- 同方式内升级:见 [升级与迁移](/ops/upgrade-migration)
- 跨方式迁移(如 Docker → 二进制):停老 → 备份 → 在新方式起 → 恢复数据库 + 上传文件 + config.yml
