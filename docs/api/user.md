# User 前台 API 文档

dujiao-shop **公开 API** 一览,供你做自定义前端 / 移动端 / 小程序接入。

> 完整 OpenAPI / Swagger 还在整理中。本页是高频接口的速查表。所有路径以 `/api/v1/` 开头,默认监听 `localhost:8080`。

## 公开接口 (无需登录)

| Method | Path | 用途 |
|---|---|---|
| `GET` | `/api/v1/public/config` | 站点配置(名称 / 货币 / 启用的支付通道等)|
| `GET` | `/api/v1/public/products` | 商品列表(分页 / 分类筛选 / 搜索)|
| `GET` | `/api/v1/public/products/:slug` | 商品详情 |
| `GET` | `/api/v1/public/posts` | 文章列表 |
| `GET` | `/api/v1/public/posts/:slug` | 文章详情 |
| `GET` | `/api/v1/public/banners` | Banner 列表 |
| `GET` | `/api/v1/public/categories` | 分类列表 |
| `GET` | `/api/v1/public/captcha/image` | 图形验证码(base64) |
| `GET` | `/api/v1/public/member-levels` | 会员等级展示 |

## 游客下单 (无需注册,凭 guest_token 查订单)

| Method | Path | 用途 |
|---|---|---|
| `POST` | `/api/v1/guest/orders` | 创建游客订单 |
| `GET` | `/api/v1/guest/orders/:order_no` | 查游客订单(带 `Authorization: Bearer <guest_token>`) |

guest_token 在创建订单的响应里返回,有效期跟订单生命周期一致,丢了用 email 找回。

## 注册 / 登录

| Method | Path | 用途 |
|---|---|---|
| `POST` | `/api/v1/auth/register` | 注册(邮箱 + 验证码)|
| `POST` | `/api/v1/auth/login` | 登录(邮箱密码)|
| `POST` | `/api/v1/auth/email/send-code` | 发邮箱验证码 |
| `POST` | `/api/v1/auth/password/forgot` | 找回密码 |
| `POST` | `/api/v1/auth/refresh` | 刷新 token |
| `POST` | `/api/v1/auth/login/totp` | 2FA 校验(若用户启用了 TOTP)|

## 已登录用户 (前缀同上,需 `Authorization: Bearer <user_jwt>`)

| Method | Path | 用途 |
|---|---|---|
| `GET` | `/api/v1/me` | 当前用户信息 |
| `PUT` | `/api/v1/me/profile` | 改昵称 / 头像 |
| `PUT` | `/api/v1/me/password` | 改密码 |
| `GET` | `/api/v1/me/login-logs` | 登录历史 |
| `GET` | `/api/v1/orders` | 我的订单列表 |
| `POST` | `/api/v1/orders` | 创建订单 |
| `GET` | `/api/v1/orders/:order_no` | 订单详情(含卡密) |
| `GET` | `/api/v1/wallet/balance` | 钱包余额 |
| `POST` | `/api/v1/wallet/recharge` | 充值订单 |
| `GET` | `/api/v1/wallet/transactions` | 钱包流水 |
| `POST` | `/api/v1/gift-cards/redeem` | 兑换礼品卡 |
| `GET` | `/api/v1/coupons/check` | 校验优惠码可用性 |

## 支付回调 (公网,各支付商访问)

| Method | Path | 用途 |
|---|---|---|
| `GET/POST` | `/api/v1/payments/callback?provider=...&id=...` | 支付商异步通知 |

详见 [支付配置与回调指南](/payment/setup)。

## 通用响应格式

```json
{
  "status_code": 0,             // 0 = success, 非 0 = 业务错误
  "msg": "success",
  "data": { ... },              // 业务数据
  "request_id": "uuid"          // 排查用
}
```

错误:
```json
{
  "status_code": 401,           // HTTP 状态码同步,也可以业务码不同
  "msg": "邮箱或密码错误",
  "data": null,
  "request_id": "uuid"
}
```

## curl 示例

```bash
# 站点配置
curl https://your.com/api/v1/public/config | jq

# 登录
curl -X POST https://your.com/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"test@example.com","password":"yourpw"}'

# 用拿到的 token 查订单
TOKEN=eyJhbGc...
curl https://your.com/api/v1/orders \
  -H "Authorization: Bearer $TOKEN" | jq
```

## CORS

默认允许 `*`。生产环境改 `config.yml` 的 `cors.allowed_origins` 限定你的域名。
