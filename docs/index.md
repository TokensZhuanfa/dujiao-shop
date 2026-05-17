---
layout: home

hero:
  name: dujiao-shop
  text: 自托管自动发卡商店
  tagline: 基于 dujiao-next fork · Codex 号池 · 文件型卡密 · 安全收紧
  image:
    src: /logo.svg
    alt: dujiao-shop
  actions:
    - theme: brand
      text: 快速开始
      link: /install/docker
    - theme: alt
      text: 在 GitHub 查看
      link: https://github.com/TokensZhuanfa/Dujiao-Shop

features:
  - icon: 🛒
    title: 完整商城
    details: 商品 / 订单 / 支付 / 库存 / 会员体系,继承 dujiao-next 全部核心功能
  - icon: 🔐
    title: 多种卡密交付
    details: 文本卡密 · 文件型卡密 · 号池型(账号) · CpaMC/Sub2api 双格式打包下载
  - icon: 🤖
    title: Codex 号池
    details: 自动维护 OpenAI/ChatGPT 账号 token 轮换、额度刷新、状态识别,库存即时反映可用账号数
  - icon: 🛡️
    title: 安全收紧
    details: JWT + 2FA · 订单号 base32 高熵 · guest 接口限流 · bcrypt cost 10 · 密码最小 8 位
  - icon: 💼
    title: 运维 CLI
    details: admin-tool 二进制提供 list-admins / reset-2fa 等离线运维操作
  - icon: 🚀
    title: 四种部署
    details: Docker Compose / 宝塔面板 / 单文件二进制 fullstack / 源码编译,按场景选
---

## 选你的部署方式

| 场景 | 推荐 |
|---|---|
| 第一次部署、不熟运维 | [**Docker Compose**](/install/docker) |
| 已经在用宝塔面板管站 | [**宝塔面板**](/install/baota) |
| 小内存 VPS、不想跑 docker | [**二进制 fullstack**](/install/binary) |
| 想改代码 / 二次开发 | [**源码编译**](/install/source) |

## 装完看这些

- 📖 [**config.yml 详解**](/reference/config) — 每个字段干嘛的
- 💳 [**支付通道接入**](/admin/payment) — 易支付 / Stripe / PayPal / USDT
- 🎛️ [**后台使用速览**](/admin/usage) — 商品 / 订单 / 卡密 / 号池
- 💾 [**备份恢复**](/ops/backup-restore) — 每天 cron + 异地
- 🛡️ [**安全加固 checklist**](/ops/security) — 上线前过一遍
- 🩺 [**故障排查**](/ops/troubleshooting) — 出问题对照看
- ❓ [**FAQ**](/faq) — 高频问题速答

## 最新 Release

最新版本在 [GitHub Releases](https://github.com/TokensZhuanfa/Dujiao-Shop/releases) 页面下载。
所有发布构件经 GitHub Actions 自动 build,**包含 SHA-256 校验文件**。
