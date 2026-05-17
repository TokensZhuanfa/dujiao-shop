# 开源仓库与贡献

## 主仓库

GitHub: <https://github.com/TokensZhuanfa/Dujiao-Shop>

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
│   └── release/
│       ├── install.sh              # 二进制版安装脚本
│       ├── dujiao.service          # systemd unit
│       └── README-binary.md        # tarball 内附说明
├── docs/                           # 本文档站 (VitePress)
└── .github/workflows/
    ├── release.yml                 # tag v* 自动出 release
    └── docs.yml                    # docs/ 改动自动发到 GitHub Pages
```

## 上游与同步

`src/api`、`src/admin`、`src/user` 三个目录的代码源自上游 [dujiao-next](https://github.com/dujiao-next)。同步上游变更:

```bash
git remote add upstream-api   https://github.com/dujiao-next/dujiao-next.git
git remote add upstream-admin https://github.com/dujiao-next/admin.git
git remote add upstream-user  https://github.com/dujiao-next/user.git

git fetch upstream-api   main && git subtree pull --prefix=src/api   upstream-api   main -m "merge: upstream api"
git fetch upstream-admin main && git subtree pull --prefix=src/admin upstream-admin main -m "merge: upstream admin"
git fetch upstream-user  main && git subtree pull --prefix=src/user  upstream-user  main -m "merge: upstream user"
```

> 当前 monorepo 基线: `src/api@faacb3f` / `src/admin@5d0ce38` / `src/user@6b67ef9` (2026-05-14)

### 同步时常见冲突

- `package.json` 依赖版本
- `i18n/index.ts` 国际化字串(我们改过的品牌名 `Dujiao-Shop` 会被改回)
- `SiteConnections.vue` 里 `protocol: 'dujiao-next'` **保留**,不要改成 `dujiao-shop`(那是数据库 enum 值,改了破坏跨站点互联)

## 主要二次开发

| 改造 | 实现位置 | 说明 |
|---|---|---|
| **Codex 号池** | `src/api/internal/service/codex_account_service.go` | token 轮换 + 额度刷新 + 状态识别 |
| **号池预占** | `src/api/internal/service/order_service.go:488` | 原子占住账号,付款转 sold |
| **文件型卡密** | `src/api/internal/service/fulfillment_service.go` | 1 GB 内文件交付 |
| **CpaMC / Sub2api 双格式** | `src/admin/src/views/orders/CodexAccountsModal.vue` | 单 / 全部打包下载 |
| **订单号 base32 高熵** | `src/api/internal/utils/orderno.go` | 防扫单 |
| **JWT + 2FA** | `src/api/internal/service/auth_service.go` | TOTP + recovery codes |
| **暴破锁定** | config `security.login_rate_limit` | 5/300s 锁 900s |

## 贡献

欢迎 PR。流程:

1. Fork → 改代码 → 跑通本地测试
2. commit message 用 [Conventional Commits](https://www.conventionalcommits.org):
   ```
   feat: 加 PayPal 退款 API
   fix: 订单超时归还 reservation 锁竞争
   docs: 更新宝塔教程的 redis 部分
   chore: 升级 asynq 到 v0.25
   ```
3. PR 描述带:**改动目的 / 涉及的文件 / 怎么测**
4. 等 review,过 CI 后合并

## License

MIT(详见仓库根目录 `LICENSE`)。基于上游 [dujiao-next](https://github.com/dujiao-next) 改造与扩展。
