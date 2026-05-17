# 关于 dujiao-shop

**dujiao-shop** 是一套自托管的自动发卡 / 卡密商店,基于上游 [dujiao-next](https://github.com/dujiao-next) 二次开发,主要面向想售卖**数字交付**(卡密、账号、文件、订阅码等)的中小站长。

## 它能干什么

- **完整商城**:商品 / SKU / 分类 / 订单 / 支付 / 退款 / 库存
- **多种卡密交付**:
  - **文本卡密**:经典的字符串密钥,适合软件序列号、激活码
  - **文件卡密**:1 GB 内任意文件,买家付款后从订单页下载
  - **号池(账号)**:OpenAI / ChatGPT 账号体系,自动维护 token 轮换、额度刷新
- **会员体系**:等级 / 钱包 / 充值 / 礼品卡
- **营销**:优惠券 / 活动价 / 分销推广(联盟营销)
- **支付通道**:易支付 / Stripe / PayPal / USDT (TRC20/BEP20) 等
- **安全收紧**:JWT + 2FA / 订单号高熵 / guest 接口限流 / bcrypt cost 10 / 暴破锁定

## 跟上游 dujiao-next 的差别

| | dujiao-next | dujiao-shop |
|---|---|---|
| 三仓 vs 单仓 | api / admin / user 三独立仓 | monorepo (`src/{api,admin,user}`) |
| Codex 号池 | 无 | ✅ 自动 token 轮换 + 额度刷新 |
| 文件型卡密 | 无 | ✅ 单文件 ≤ 1 GB,从订单页下载 |
| CpaMC / Sub2api 双格式 | 无 | ✅ 号池账号单 / 批量打包下载 |
| 订单号熵 | 较短 | base32 高熵,防扫单 |
| 暴破锁定 | 无 | 5 分钟内 5 次失败锁 15 分钟 |
| 二进制发布 | 无 | ✅ fullstack 单文件 + headless 两种 |

## 快速上手

1. 选你的部署方式(见 [部署总览](/deploy/overview))
2. 改 [config.yml](/config/yaml)
3. 起服务,登录 [admin 后台](/guide/admin-getting-started)
4. 接 [支付通道](/payment/setup) + 加商品
5. 上线

## 源码

GitHub: <https://github.com/TokensZhuanfa/Dujiao-Shop>
License: MIT
