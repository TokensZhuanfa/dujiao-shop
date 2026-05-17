---
layout: home

hero:
  name: dujiao-shop
  text: 自托管自动发卡商店
  tagline: Codex 号池 · 文件型卡密 · 多支付通道 · 安全收紧
  image:
    src: /logo.svg
    alt: dujiao-shop
  actions:
    - theme: brand
      text: 快速开始
      link: /intro/about
    - theme: alt
      text: 在 GitHub 查看
      link: https://github.com/TokensZhuanfa/Dujiao-Shop

features:
  - icon: 🛒
    title: 完整业务闭环
    details: 商品 / 订单 / 支付 / 库存 / 会员 / 优惠券 / 分销推广,可直接开站接单
  - icon: 🤖
    title: Codex 号池
    details: OpenAI/ChatGPT 账号 token 自动轮换、额度刷新、状态识别,库存即时反映可用账号数
  - icon: 🔐
    title: 多种交付 + 安全收紧
    details: 文本 / 文件 / 号池账号 / CpaMC·Sub2api 双格式 + JWT+2FA + 暴破锁定
  - icon: 🚀
    title: 四种部署方式
    details: Docker Compose / 宝塔面板 / 单二进制 fullstack / 源码编译,按场景选
---

## 选你的部署方式

| 场景 | 推荐 |
|---|---|
| 第一次部署、不熟运维 | [**Docker Compose**](/deploy/docker-compose) |
| 已经在用宝塔面板管站 | [**宝塔面板**](/deploy/baota) |
| 小内存 VPS、不想跑 docker | [**单二进制 fullstack**](/deploy/single-binary) |
| 想改代码 / 二次开发 | [**源码编译**](/deploy/manual) |

## 装完看这些

- 📖 [**config.yml 详解**](/config/yaml) — 每个字段干嘛的
- 💳 [**支付配置与回调**](/payment/setup) — 易支付 / Stripe / PayPal / USDT
- 🎛️ [**后台管理入门**](/guide/admin-getting-started) — 商品 / 订单 / 卡密 / 号池
- 💾 [**备份与恢复**](/ops/backup-restore) — 每天 cron + 异地
- 🛡️ [**安全最佳实践**](/guide/security) — 上线前 checklist
- 🩺 [**故障排查**](/ops/troubleshooting) — 出问题对照看
- ❓ [**FAQ**](/guide/faq) — 高频问题速答

## 最新 Release

最新版本在 [GitHub Releases](https://github.com/TokensZhuanfa/Dujiao-Shop/releases) 页面下载。
所有发布构件经 GitHub Actions 自动 build,**包含 SHA-256 校验文件**。
