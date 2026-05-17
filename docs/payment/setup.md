# 支付通道接入

dujiao-shop 后台 → **支付通道管理** 可挂接多个支付渠道,每个订单付款时**用户可选**或**系统按规则路由**。

## 已支持

| 通道 | 配置入口 | 适合 |
|---|---|---|
| **易支付** (PayJS / 码支付等通用) | 通道类型选 `epay` | 个人站快速接入 |
| **Stripe** | `stripe` | 国际信用卡 / Apple Pay |
| **PayPal** | `paypal` | 海外用户 |
| **加密货币** (USDT-TRC20 / BEP20) | `crypto` | 匿名 / 跨境 |
| **TokenPay** | `tokenpay` | USDT 等数字货币聚合 |
| **BEpusdt** | `bepusdt` | USDT-BEP20 / 自建 |

## 通用步骤

1. 后台 → **支付通道管理** → **添加通道**
2. 选**通道类型**,会自动出对应字段
3. 填**通道名称**(用户结账时看到的标签)
4. 填**密钥 / API key / merchant id** 等
5. **回调地址** 一般是自动生成的 `https://your.com/api/v1/payments/callback`
6. **测试**:用户下单 → 选这个通道 → 跳转支付 → 回来看订单变 `paid`

## 易支付(epay) 详细

接入码支付 / 易支付 / PayJS 等"易支付协议"通用网关:

| 字段 | 值 |
|---|---|
| API URL | `https://pay.example.com/submit.php`(去你支付商后台拿) |
| 商户 ID (`pid`) | 商户号 |
| 商户密钥 (`key`) | API key,32 位 hex |
| 异步通知 URL | `https://your.com/api/v1/payments/callback?provider=epay&id=<channel_id>` |
| 同步跳转 URL | `https://your.com/orders/{order_no}`(用户付完跳哪) |

> 务必让你的支付商把**异步通知 URL** 加白名单/配好,否则订单永远卡 `pending`。

### 测试通道

```bash
# 后台 → 支付通道 → "测试" 按钮 (会发一个 1 分钱测试请求)
# 或者 cURL 直接 hit 回调:
curl -X POST 'https://your.com/api/v1/payments/callback?provider=epay&id=1' \
  -d 'pid=xxx&trade_no=test123&out_trade_no=...&type=alipay&trade_status=TRADE_SUCCESS&money=0.01&sign=...'
```

成功标志:返回 `success`(纯文本)。

## Stripe 详细

1. https://dashboard.stripe.com → API keys 拿 `Publishable key` + `Secret key`
2. **Webhooks** → Add endpoint
   - URL: `https://your.com/api/v1/payments/callback?provider=stripe&id=<channel_id>`
   - Events: `checkout.session.completed`, `payment_intent.succeeded`
   - 拿 **Signing secret**(`whsec_...`)
3. 后台配置填:

| 字段 | 值 |
|---|---|
| Publishable key | `pk_live_xxx` |
| Secret key | `sk_live_xxx` |
| Webhook secret | `whsec_xxx` |
| Currency | `USD` / `CNY` / ... |

## PayPal 详细

1. https://developer.paypal.com → My Apps → Create App(选 `Live`)
2. 拿 **Client ID** + **Secret**
3. 后台配置:

| 字段 | 值 |
|---|---|
| Client ID | `Acgj-...` |
| Secret | `EHxj-...` |
| Mode | `live`(测试用 `sandbox`)|
| Currency | `USD`(国内人民币需要先在 PayPal 后台开通)|

PayPal 不需要单独配 webhook,api 会主动查支付状态。

## 加密支付 (USDT-TRC20)

通过 [BEpusdt](https://github.com/v03413/BEpusdt) 或 [TokenPay](https://github.com/LightCountry/TokenPay) 接入:

### BEpusdt

1. 在你自己服务器装 BEpusdt(几行 docker compose)
2. 后台支付通道选 `bepusdt`
3. 填:
   - API URL: 你 BEpusdt 实例地址,如 `https://pay.your.com`
   - Token: BEpusdt 后台拿到的 API token
   - Network: `TRC20` / `BEP20` / `Polygon` 等

### TokenPay

同 BEpusdt 流程,API URL + token 不同。

## 多通道路由策略

后台支付通道管理 → 每个通道可设:

- **是否启用**
- **最小订单金额**:小订单不走某通道(避免手续费比订单高)
- **最大订单金额**
- **优先级**:同时多个通道符合时显示顺序
- **支持的币种**

用户结账时**自动只显示符合的通道**。

## 回调安全

每个支付商的回调签名机制不同,**dujiao-api 内置全部验签**:

- **易支付**: MD5(`商户密钥` + 排序后参数)
- **Stripe**: HMAC-SHA256(`whsec_` + body)
- **PayPal**: API 反查
- **加密通道**: API 反查 + token

签名验失败会直接 400 拒绝,**不会**把订单标 paid。

## 排查

| 现象 | 排查 |
|---|---|
| 订单永远 pending | 看支付商后台是否发送回调成功,看 `/api/v1/payments/callback` 路由通否 |
| 回调到了但订单没变 | 看 api 日志找 `payment callback`,有 `signature mismatch` 就是密钥不对 |
| 用户付了钱看不到通道 | 后台启用通道 / 检查 min/max 金额 / 检查币种 |
| 通道列表为空 | 后台没启用任何通道 → 至少建一个 |

```bash
# 日志关键字
grep -i "payment callback\|signature\|payment.*failed" /var/log/dujiao/api.log | tail -30
```
