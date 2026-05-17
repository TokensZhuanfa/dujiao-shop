# 术语统一表

阅读本文档前先眼熟这些术语。

## 商品类

| 术语 | 解释 |
|---|---|
| **SKU** | Stock Keeping Unit。一个商品可以有多个 SKU(同款不同规格,如"3 个月 / 半年 / 1 年")|
| **库存** | 该 SKU 当前可售数量。文本/文件卡密 = 卡密池可用条数;号池 = pool 里 `ok` 且未预占的账号数 |
| **单笔限购** | 防黄牛,限制单订单某 SKU 数量上限 |
| **划线价 / 原价** | 商品页打折效果用的对照价,大于实际售价 |
| **活动价** | 通过优惠券或会员等级触发的折扣价 |

## 交付类

| 术语 | 解释 |
|---|---|
| **卡密** | 通用术语,泛指交付给买家的"密钥"。dujiao-shop 三种实现:文本 / 文件 / 号池账号 |
| **`auto_secret_kind`** | 商品表字段,定义交付方式:`text` / `file` / `codex_pool` |
| **号池(Pool)** | 一组同质的可分配账号(典型:Codex / OpenAI 账号),库存自动按可用数 |
| **预占 (Reservation)** | 下单时占住一条库存,付款 → sold,超时 → 归还 |
| **CpaMC / Sub2api** | 号池账号的两种打包导出格式,买家可下载 |

## 订单类

| 术语 | 解释 |
|---|---|
| **`order_no`** | 订单号,base32 高熵,如 `O-3KZJ7HW9MX4VPQ8R` |
| **pending** | 待支付状态 |
| **paid** | 已付款,卡密已交付 |
| **cancelled** | 订单取消(超过 `payment_expire_minutes` 自动 cancel + 归还库存)|
| **refunded** | 已退款,可全额或部分退 |
| **guest order** | 游客订单(未注册用户下单),凭 `guest_token` 查订单 |

## 账户类

| 术语 | 解释 |
|---|---|
| **admin** | 后台管理员,走 `jwt.secret` |
| **user** | 前台用户,走 `user_jwt.secret`,有钱包余额 |
| **2FA / TOTP** | 双因素认证,用 Authenticator app 生成 6 位验证码 |
| **recovery codes** | 备份码,丢了 Authenticator 时一次性救命用 |
| **token_version** | 改密 / 启 2FA / 退出全部设备时 +1,**让现有 JWT 失效** |
| **session 过期** | JWT exp 时间到了 → 重新登录 |

## 系统类

| 术语 | 解释 |
|---|---|
| **asynq** | Go 异步任务库,跑订单超时 / 邮件发送 / 上游同步等 |
| **upstream-sync** | 站点对接,允许 dujiao 站点之间互联,如分销联盟 |
| **审计日志** | 后台谁在何时改了什么,可溯源 |

## 部署类

| 术语 | 解释 |
|---|---|
| **fullstack** binary | api 二进制内嵌 admin + user 前端,单进程跑全栈 |
| **headless** binary | api 二进制不内嵌前端,前端给 nginx 静态托管 |
| **install.sh** | release tarball 自带的一键安装脚本 |
| **systemd unit** | `/etc/systemd/system/dujiao.service` 进程守护 |
